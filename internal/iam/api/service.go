package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/jwt"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/passhash"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/redis"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
	"github.com/casbin/casbin/v2"
)

var (
	ErrInvalidUser  = errors.New("invalid user")
	ErrWeakPassword = errors.New("password must be at least 8 characters and include letters, digits, and symbols")
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
	ErrForbidden    = errors.New("forbidden")
	// ErrReadForbidden 表示读访问被拒绝：按需求对外返回 404（不暴露资源是否存在），
	// 与写/删无权限（ErrForbidden → 403）区分。
	ErrReadForbidden  = errors.New("read not permitted")
	ErrCasbinDisabled = errors.New("casbin not initialized")
	defaultTokenTTL   = time.Hour
)

const defaultJWTIssuer = "opshub-auth"

type Service struct {
	decisionSecret []byte
	jwtSecret      []byte
	tokenTTL       time.Duration
	issuer         string
	engine         GrantEngine              // 配置授权决策引擎（可替换：casbin / 公司权限中心）
	users          store.UserStore          // 用户存储（可替换：MySQL / LDAP / 公司用户源）
	accessKeys     store.AccessKeyStore     // 服务凭证存储（AccessKey，程序化鉴权）
	svcMods        store.ServiceModuleStore // 服务模块订阅存储（注册哪些模块可拉取）
	cache          *redis.Cache
}

// NewService 创建 IAM 服务（casbin 授权引擎）；enf 可为 nil（则仅 IsAdmin 能通过配置鉴权，非管理员一律拒绝配置操作）。
func NewService(enf *casbin.SyncedEnforcer) *Service {
	return newService(&casbinGrantEngine{enf: enf}, authutil.DefaultDecisionSecret, defaultTokenTTL)
}

// NewServiceWithEngine 用自定义授权引擎构建服务（对接公司权限中心时传入其 GrantEngine 实现）。
func NewServiceWithEngine(engine GrantEngine) *Service {
	return newService(engine, authutil.DefaultDecisionSecret, defaultTokenTTL)
}

// SetRedisCache 附加 Redis（可选），用于登出令牌黑名单等场景。
func (s *Service) SetRedisCache(c *redis.Cache) *Service {
	s.cache = c
	return s
}

func (s *Service) tokenBlacklistKey(token string) string {
	return "opshub:iam:blacklist:" + token
}

// Logout 将令牌加入黑名单直至其过期（未配置 Redis 时为空操作，靠令牌自然过期）。
func (s *Service) Logout(ctx context.Context, token string) error {
	if s.cache == nil {
		return nil
	}
	subject, exp, err := s.parseToken(token)
	if err != nil {
		return err
	}
	ttl := time.Until(exp)
	if ttl <= 0 {
		return nil
	}
	return s.cache.SetString(ctx, s.tokenBlacklistKey(token), subject, ttl)
}

func (s *Service) isTokenBlacklisted(token string) bool {
	if s.cache == nil {
		return false
	}
	_, ok := s.cache.GetString(context.Background(), s.tokenBlacklistKey(token))
	return ok
}

func newService(engine GrantEngine, secret []byte, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	if len(secret) == 0 {
		secret = authutil.DefaultDecisionSecret
	}
	return &Service{
		decisionSecret: append([]byte(nil), secret...),
		jwtSecret:      append([]byte(nil), secret...),
		tokenTTL:       ttl,
		issuer:         defaultJWTIssuer,
		engine:         engine,
	}
}

// SetUserStore 注入用户存储。默认使用全局 store.Client().Users()（MySQL）；
// 对接 LDAP / 公司用户源时注入自定义 store.UserStore 实现即可替换。
func (s *Service) SetUserStore(us store.UserStore) *Service {
	s.users = us
	return s
}

// userStore 返回注入的用户存储，未注入时回退到全局 MySQL 存储。
func (s *Service) userStore() store.UserStore {
	if s.users != nil {
		return s.users
	}
	return store.Client().Users()
}

// UserStore 暴露给 Handler（用户管理接口）使用的用户存储。
func (s *Service) UserStore() store.UserStore {
	return s.userStore()
}

// SetAccessKeyStore 注入访问凭证存储（默认全局 MySQL）。
func (s *Service) SetAccessKeyStore(aks store.AccessKeyStore) *Service {
	s.accessKeys = aks
	return s
}

// accessKeyStore 返回注入的凭证存储，未注入时回退到全局 MySQL。
func (s *Service) accessKeyStore() store.AccessKeyStore {
	if s.accessKeys != nil {
		return s.accessKeys
	}
	return store.Client().AccessKeys()
}

type LoginResult struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
}

type signupRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

func (s *Service) Signup(c *middleware.Context, req signupRequest) error {
	subject := strings.TrimSpace(req.User)
	if subject == "" {
		return ErrInvalidUser
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}

	_, err := s.userStore().Get(c, subject)
	if err == nil {
		return fmt.Errorf("user %s already exists", subject)
	}

	hashed, err := passhash.Hash(req.Password)
	if err != nil {
		return err
	}

	// 首个注册用户自动成为管理员（本地部署/教学便捷；已有用户后不再生效）。
	// 生产环境请改用 bootstrap_admin 配置或关闭该逻辑。
	firstUser, err := s.isFirstUser(c)
	if err != nil {
		return err
	}

	newUser := &store.User{
		Username: subject,
		Password: hashed,
		Email:    req.Email,
		Phone:    req.Phone,
		IsAdmin:  firstUser,
		Status:   1,
	}

	if err := s.userStore().Create(c, newUser); err != nil {
		return err
	}
	if firstUser {
		logger.Infof("iam: first user %q registered as admin", subject)
	}

	return nil
}

// isFirstUser 判断当前是否还没有任何用户（首个注册用户自动成为管理员）。
func (s *Service) isFirstUser(ctx context.Context) (bool, error) {
	list, err := s.userStore().List(ctx)
	if err != nil {
		return false, err
	}
	return len(list) == 0, nil
}

// validatePassword enforces basic strength requirements for new accounts.
func validatePassword(raw string) error {
	if len(raw) < 8 {
		return ErrWeakPassword
	}
	var hasLetter, hasDigit, hasSymbol bool
	for _, r := range raw {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		case unicode.IsSpace(r):
			// spaces are ignored for symbol requirement
		default:
			hasSymbol = true
		}
	}
	if !hasLetter || !hasDigit || !hasSymbol {
		return ErrWeakPassword
	}
	return nil
}

func (s *Service) Login(c *middleware.Context, req loginRequest) (LoginResult, error) {
	subject := strings.TrimSpace(req.User)
	if subject == "" {
		return LoginResult{}, ErrInvalidUser
	}

	userInfo, err := s.userStore().Get(c, subject)
	if err != nil {
		return LoginResult{}, ErrInvalidUser
	}

	if err := userInfo.ComparePassword(req.Password); err != nil {
		return LoginResult{}, ErrInvalidUser
	}

	if !passhash.LooksBcrypt(userInfo.Password) {
		h, err := passhash.Hash(req.Password)
		if err == nil {
			userInfo.Password = h
		}
	}

	token, err := jwt.GenToken(userInfo.ID, subject, userInfo.IsAdmin)
	if err != nil {
		return LoginResult{}, err
	}

	userInfo.LoginedAt = time.Now().Unix()
	if err := s.userStore().Update(c, userInfo); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: time.Now().Add(jwt.TokenExpireDuration),
	}, nil
}

func (s *Service) Refresh(c *middleware.Context, token string) (LoginResult, error) {
	subject, expiresAt, err := s.parseToken(token)
	if err != nil {
		return LoginResult{}, err
	}
	if time.Now().UTC().After(expiresAt) {
		return LoginResult{}, ErrTokenExpired
	}

	userInfo, err := s.userStore().Get(c, subject)
	if err != nil {
		return LoginResult{}, ErrInvalidUser
	}

	newToken, err := jwt.GenToken(userInfo.ID, subject, userInfo.IsAdmin)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Token:     newToken,
		TokenType: "Bearer",
		ExpiresAt: time.Now().Add(jwt.TokenExpireDuration),
	}, nil
}

// Scope 返回某用户对配置的全部授权（scope）。
// 管理员 → 全量授权；普通用户 → 由其 casbin 策略（含角色继承）推导出 config/ 对象授权。
func (s *Service) Scope(subject string) ([]casbinx.Grant, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, ErrInvalidUser
	}
	userInfo, err := s.userStore().Get(context.Background(), subject)
	if err != nil {
		return nil, ErrInvalidUser
	}
	if userInfo.IsAdmin {
		return casbinx.AdminGrant, nil
	}
	return s.collectGrants(subject)
}

// UserGrants 返回用户实际的 casbin 配置授权（不含管理员短路），供管理端展示。
func (s *Service) UserGrants(subject string) ([]casbinx.Grant, error) {
	return s.collectGrants(strings.TrimSpace(subject))
}

func (s *Service) collectGrants(subject string) ([]casbinx.Grant, error) {
	if s.engine == nil {
		return nil, nil
	}
	return s.engine.ImplicitGrants(subject)
}

// Authorize 校验令牌并返回鉴权决策。对配置资源：
//   - 管理员：始终放行（scope=全量）。
//   - 普通用户读操作：只要拥有任意配置授权即放行（由配置中心按 scope 精确过滤）；
//   - 普通用户写/删操作：按具体资源精确校验 casbin。
//
// 决策载体 Decision 为 scope 签名串：scope|<subject>|<grantsJSON>|<unix>。
func (s *Service) Authorize(token, resource, action string) (*middleware.AuthDecision, error) {
	subject, expiresAt, err := s.parseToken(token)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(expiresAt) {
		return nil, ErrTokenExpired
	}
	if s.isTokenBlacklisted(token) {
		return nil, ErrInvalidToken
	}

	resource = strings.TrimSpace(resource)
	if resource == "" {
		resource = "/"
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "UNKNOWN"
	}

	userInfo, err := s.userStore().Get(context.Background(), subject)
	if err != nil {
		return nil, ErrInvalidUser
	}

	allow := false
	act := "" // normalize 后的动作（read/write/delete），供下方无权限分支区分读/写
	if userInfo.IsAdmin {
		allow = true
	} else if casbinx.IsConfigResourcePath(resource) {
		var obj string
		obj, act = casbinx.NormalizeConfigResource(resource, action)
		if isConfigWriteAction(act) {
			// 写/删：精确校验目标资源
			if s.engine != nil {
				allow, err = s.engine.Enforce(subject, obj, act)
				if err != nil {
					return nil, err
				}
			}
		} else {
			// 读：拥有任意配置授权即可进入，由配置中心按 scope 过滤；
			// 一个授权都没有时按需求对外返回 404（读无权限，不暴露资源存在）。
			grants, gerr := s.collectGrants(subject)
			if gerr != nil {
				return nil, gerr
			}
			allow = len(grants) > 0
		}
	}

	if !allow {
		// 注意用 normalize 后的 act（read/write/delete，由 mapHTTPMethod 统一），
		// 而非原始 action（网关传的是大写 HTTP 方法 GET/POST/PUT/DELETE）。
		if isConfigReadAction(act) {
			return nil, ErrReadForbidden
		}
		return nil, ErrForbidden
	}

	scope, err := s.Scope(subject)
	if err != nil {
		return nil, err
	}
	decisionPayload, _ := casbinx.BuildScopePayload(subject, scope)
	signature := authutil.Sign(decisionPayload, s.decisionSecret)

	return &middleware.AuthDecision{
		Allow:     true,
		Subject:   subject,
		Action:    action,
		Resource:  resource,
		Scope:     scope,
		Decision:  decisionPayload,
		Signature: signature,
	}, nil
}

func isConfigWriteAction(act string) bool {
	return act == "write" || act == "delete"
}

func isConfigReadAction(act string) bool {
	return act == "read" || act == "get" || act == "list"
}

func (s *Service) parseToken(token string) (string, time.Time, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return "", time.Time{}, ErrInvalidToken
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		trimmed = strings.TrimSpace(trimmed[7:])
	}

	// 1) 标准账号密码 JWT（服务端静态密钥签发）
	if claims, err := jwt.ParseToken(trimmed); err == nil && claims != nil && claims.Subject != "" && claims.ExpiresAt != nil {
		return claims.Subject, claims.ExpiresAt.Time, nil
	}

	// 2) 服务凭证（AccessKey）自签 JWT：header kid=AccessKeyID，按 kid 查库拿 Secret 验签
	claims, kid, err := jwt.ParseAccessToken(trimmed)
	if err != nil || kid == "" {
		return "", time.Time{}, ErrInvalidToken
	}
	ak, aerr := s.accessKeyStore().GetByKeyID(context.Background(), kid)
	if aerr != nil || ak == nil {
		return "", time.Time{}, ErrInvalidToken
	}
	if ak.Status != 1 || (ak.Expires > 0 && time.Now().UTC().Unix() > ak.Expires) {
		return "", time.Time{}, ErrInvalidToken
	}
	if err := jwt.VerifyAccessToken(trimmed, []byte(ak.AccessKeySecret)); err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	if claims.ExpiresAt == nil || time.Now().UTC().After(claims.ExpiresAt.Time) {
		return "", time.Time{}, ErrTokenExpired
	}
	// 身份以 AccessKey 归属的服务账号为准（不信任客户端 claims）
	return ak.Username, claims.ExpiresAt.Time, nil
}
