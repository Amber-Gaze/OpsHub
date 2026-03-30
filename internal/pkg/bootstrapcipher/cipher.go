// Package bootstrapcipher 用于配置文件中的「引导管理员密码」密文：
// 主密钥经 OpsHub 固定盐派生为 AES-256 密钥，再使用 AES-GCM 加密（nonce 前置后 base64）。
// 主密钥应放在环境变量 OPSHUB_BOOTSTRAP_CIPHER_KEY 或配置 bootstrap_cipher_key（勿与密文等同）。
package bootstrapcipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

const (
	deriveLabel = "opshub-bootstrap-admin-key-v1\x00"
	aadLabel    = "opshub-bootstrap-admin-aad-v1"
)

func deriveKey(master string) []byte {
	if master == "" {
		return nil
	}
	sum := sha256.Sum256(append([]byte(deriveLabel), []byte(master)...))
	return sum[:]
}

// Encrypt 使用主密钥加密明文密码，返回可写入 YAML 的 base64 串。
func Encrypt(masterKey, plaintext string) (string, error) {
	if masterKey == "" {
		return "", errors.New("bootstrap cipher: empty master key")
	}
	key := deriveKey(masterKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(aadLabel))
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt 用同一主密钥解出明文密码。
func Decrypt(masterKey, cipherB64 string) (string, error) {
	if masterKey == "" {
		return "", errors.New("bootstrap cipher: empty master key")
	}
	raw, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}
	key := deriveKey(masterKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("bootstrap cipher: ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	plain, err := gcm.Open(nil, nonce, ct, []byte(aadLabel))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
