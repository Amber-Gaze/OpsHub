package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/casbin/casbin/v2"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/casbinx"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/jwt"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/passhash"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/repository/redis"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
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
	enf            *casbin.SyncedEnforcer
	cache          *redis.Cache
}

// NewService 创建 IAM 服务；enf 可为 nil（则仅 IsAdmin 能通过配置鉴权，非管理员一律拒绝配置操作）。
func NewService(enf *casbin.SyncedEnforcer) *Service {
	return newService(enf, authutil.DefaultDecisionSecret, defaultTokenTTL)
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

func newService(enf *casbin.SyncedEnforcer, secret []byte, ttl time.Duration) *Service {
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
		enf:            enf,
	}
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

	_, err := store.Client().Users().Get(c, subject)
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

	if err := store.Client().Users().Create(c, newUser); err != nil {
		return err
	}
	if firstUser {
		logger.Infof("iam: first user %q registered as admin", subject)
	}

	return nil
}

// isFirstUser 判断当前是否还没有任何用户（首个注册用户自动成为管理员）。
func (s *Service) isFirstUser(ctx context.Context) (bool, error) {
	list, err := store.Client().Users().List(ctx)
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

	userInfo, err := store.Client().Users().Get(c, subject)
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
	if err := store.Client().Users().Update(c, userInfo); err != nil {
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

	userInfo, err := store.Client().Users().Get(c, subject)
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
	userInfo, err := store.Client().Users().Get(context.Background(), subject)
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
	if s.enf == nil {
		return nil, nil
	}
	perms, err := s.enf.GetImplicitPermissionsForUser(subject)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	grants := make([]casbinx.Grant, 0, len(perms))
	for _, p := range perms {
		if len(p) < 3 || !strings.HasPrefix(p[1], "config/") {
			continue
		}
		key := p[1] + "|" + p[2]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		grants = append(grants, casbinx.Grant{Obj: p[1], Act: p[2]})
	}
	return grants, nil
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

	userInfo, err := store.Client().Users().Get(context.Background(), subject)
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
			if s.enf != nil {
				allow, err = s.enf.Enforce(subject, obj, act)
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
	claims, err := jwt.ParseToken(trimmed)
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	if claims == nil {
		return "", time.Time{}, ErrInvalidToken
	}
	if claims.Subject == "" || claims.ExpiresAt == nil {
		return "", time.Time{}, ErrInvalidToken
	}
	return claims.Subject, claims.ExpiresAt.Time, nil
}
