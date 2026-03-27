package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID       int64  `json:"uid"`
	Username     string `json:"username"`
	TokenVersion int    `json:"tv"`
	jwt.RegisteredClaims
}
