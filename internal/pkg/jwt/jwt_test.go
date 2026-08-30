package jwt

import (
	"testing"
	"time"
)

// TestGenAndVerifyAccessToken 验证 AccessKey 自签 JWT 的生成、kid 提取与验签。
func TestGenAndVerifyAccessToken(t *testing.T) {
	kid := "AK0123456789abcdef"
	secret := []byte("s3cr3t-access-key-secret")
	token, err := GenAccessToken(kid, "svc-pay", secret, 10*time.Minute)
	if err != nil {
		t.Fatalf("GenAccessToken: %v", err)
	}

	claims, gotKid, err := ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if gotKid != kid {
		t.Fatalf("kid = %q, want %q", gotKid, kid)
	}
	if claims.ExpiresAt == nil || time.Now().After(claims.ExpiresAt.Time) {
		t.Fatalf("token should not be expired: %+v", claims.ExpiresAt)
	}

	// 正确密钥验签通过
	if err := VerifyAccessToken(token, secret); err != nil {
		t.Fatalf("VerifyAccessToken(valid) = %v", err)
	}
	// 错误密钥验签失败
	if err := VerifyAccessToken(token, []byte("wrong")); err == nil {
		t.Fatal("VerifyAccessToken(wrong secret) should fail")
	}
}

// TestAccessTokenNotFoundInStandardParse 标准（静态密钥）解析应拒绝 AccessKey 签发的 JWT。
func TestAccessTokenNotFoundInStandardParse(t *testing.T) {
	token, err := GenAccessToken("AKx", "svc", []byte("secret"), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Fatal("standard ParseToken should reject access-key signed token")
	}
}
