package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/jwt"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
)

var (
	ErrInvalidUser  = errors.New("invalid user")
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
	defaultTokenTTL = time.Hour
)

const defaultJWTIssuer = "opshub-auth"

type Service struct {
	decisionSecret []byte
	jwtSecret      []byte
	tokenTTL       time.Duration
	issuer         string
}

func NewService() *Service {
	return NewServiceWithSecret(authutil.DefaultDecisionSecret, defaultTokenTTL)
}

func NewServiceWithSecret(secret []byte, ttl time.Duration) *Service {
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
	}
}

type LoginResult struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
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

	token, err := jwt.GenToken(userInfo.ID, subject)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Token:     token,
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

	decisionPayload := fmt.Sprintf("allow|%s|%s|%s|%d", subject, action, resource, time.Now().UTC().Unix())
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
