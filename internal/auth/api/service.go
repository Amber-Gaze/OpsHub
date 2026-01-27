package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/authutil"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/middleware"
)

var (
	ErrInvalidUser  = errors.New("invalid user")
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
	defaultTokenTTL = time.Hour
)

type Service struct {
	decisionSecret []byte
	tokenTTL       time.Duration
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
		tokenTTL:       ttl,
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

	expiresAt := time.Now().Add(s.tokenTTL).UTC()
	payload := fmt.Sprintf("%s:%d", subject, expiresAt.Unix())
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	token := fmt.Sprintf("Bearer %s", encoded)

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
		trimmed = trimmed[7:]
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(trimmed))
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", time.Time{}, ErrInvalidToken
	}
	unixTs, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}
	return parts[0], time.Unix(unixTs, 0).UTC(), nil
}
