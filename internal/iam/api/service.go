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
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
)

var (
	ErrInvalidUser     = errors.New("invalid user")
	ErrWeakPassword    = errors.New("password must be at least 8 characters and include letters, digits, and symbols")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
	ErrForbidden       = errors.New("forbidden")
	ErrCasbinDisabled  = errors.New("casbin not initialized")
	defaultTokenTTL    = time.Hour
)

const defaultJWTIssuer = "opshub-auth"

type Service struct {
	decisionSecret []byte
	jwtSecret      []byte
	tokenTTL       time.Duration
	issuer         string
	enf            *casbin.SyncedEnforcer
}

// NewService 创建 IAM 服务；enf 可为 nil（则仅 IsAdmin 能通过配置鉴权，非管理员一律拒绝配置操作）。
func NewService(enf *casbin.SyncedEnforcer) *Service {
	return newService(enf, authutil.DefaultDecisionSecret, defaultTokenTTL)
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

	newUser := &store.User{
		Username: subject,
		Password: hashed,
		Email:    req.Email,
		Phone:    req.Phone,
		IsAdmin:  false,
		Status:   1,
	}

	if err := store.Client().Users().Create(c, newUser); err != nil {
		return err
	}

	return nil
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

func (s *Service) Authorize(token, resource, action string) (*middleware.AuthDecision, error) {
	subject, expiresAt, err := s.parseToken(token)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(expiresAt) {
		return nil, ErrTokenExpired
	}

	resource = strings.TrimSpace(resource)
	if resource == "" {
		resource = "/"
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "UNKNOWN"
	}

	ctx := context.Background()
	userInfo, err := store.Client().Users().Get(ctx, subject)
	if err != nil {
		return nil, ErrInvalidUser
	}

	allow := false
	casbinObj := resource
	casbinAct := action

	if casbinx.IsConfigResourcePath(resource) {
		casbinObj, casbinAct = casbinx.NormalizeConfigResource(resource, action)
		if userInfo.IsAdmin {
			allow = true
		} else if s.enf != nil {
			allow, err = s.enf.Enforce(subject, casbinObj, casbinAct)
			if err != nil {
				return nil, err
			}
		}
	} else {
		if userInfo.IsAdmin {
			allow = true
		}
	}

	if !allow {
		return nil, ErrForbidden
	}

	decisionPayload := fmt.Sprintf("allow|%s|%s|%s|%s|%s|%d", subject, action, resource, casbinObj, casbinAct, time.Now().UTC().Unix())
	signature := authutil.Sign(decisionPayload, s.decisionSecret)

	return &middleware.AuthDecision{
		Allow:     true,
		Subject:   subject,
		Action:    action,
		Resource:  resource,
		Decision:  decisionPayload,
		Signature: signature,
	}, nil
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
