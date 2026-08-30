package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
)

// ListAccessKeys 返回某服务账号的全部访问凭证。
func (s *Service) ListAccessKeys(ctx context.Context, username string) ([]*store.AccessKey, error) {
	return s.accessKeyStore().List(ctx, username)
}

// CreateAccessKey 为服务账号创建访问凭证，返回含 Secret 的完整记录（Secret 仅在创建时返回一次）。
func (s *Service) CreateAccessKey(ctx context.Context, username, description string, expires int64) (*store.AccessKey, error) {
	now := time.Now().Unix()
	ak := &store.AccessKey{
		Username:        username,
		AccessKeyID:     "AK" + randomHex(16),
		AccessKeySecret: randomHex(32),
		Description:     description,
		Status:          1,
		Expires:         expires,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.accessKeyStore().Create(ctx, ak); err != nil {
		return nil, err
	}
	return ak, nil
}

// GetAccessKey 返回某服务账号的单个访问凭证。
func (s *Service) GetAccessKey(ctx context.Context, username, keyID string) (*store.AccessKey, error) {
	return s.accessKeyStore().Get(ctx, username, keyID)
}

// DeleteAccessKey 吊销某服务账号的访问凭证。
func (s *Service) DeleteAccessKey(ctx context.Context, username, keyID string) error {
	return s.accessKeyStore().Delete(ctx, username, keyID)
}

// randomHex 生成 n 字节随机数的 hex 串（AccessKeyID / Secret 用）。
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
