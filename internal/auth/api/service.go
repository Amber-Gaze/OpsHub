package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
	"github.com/golang-jwt/jwt/v5"
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

func (s *Service) Login(user string) (LoginResult, error) {
	subject := strings.TrimSpace(user)
	if subject == "" {
		return LoginResult{}, ErrInvalidUser
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenTTL)
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    s.issuer,
	}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := unsigned.SignedString(s.jwtSecret)
	if err != nil {
		return LoginResult{}, fmt.Errorf("sign token: %w", err)
	}
	token := fmt.Sprintf("Bearer %s", signed)

	return LoginResult{
		Token:     token,
		TokenType: "Bearer",
		ExpiresAt: expiresAt,
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
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(trimmed, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", time.Time{}, ErrTokenExpired
		}
		return "", time.Time{}, ErrInvalidToken
	}
	if parsed == nil || !parsed.Valid {
		return "", time.Time{}, ErrInvalidToken
	}
	if claims.Subject == "" || claims.ExpiresAt == nil {
		return "", time.Time{}, ErrInvalidToken
	}
	return claims.Subject, claims.ExpiresAt.Time, nil
}
