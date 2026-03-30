package bootstrap

import (
	"context"
	"strings"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/bootstrapcipher"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/passhash"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
)

// EnsureAdminFromConfig 若配置了 bootstrap 用户名与密文，且该用户尚不存在，则创建管理员（bcrypt 入库）。
// 主密钥来自 OPSHUB_BOOTSTRAP_CIPHER_KEY 或 auth.bootstrap_cipher_key。
func EnsureAdminFromConfig(ctx context.Context) {
	ac := options.GetAuthConf()
	if ac == nil {
		return
	}
	username := strings.TrimSpace(ac.BootstrapAdminUsername)
	cipher := strings.TrimSpace(ac.BootstrapAdminPasswordCipher)
	if username == "" || cipher == "" {
		return
	}
	master := options.GetBootstrapCipherKey()
	if master == "" {
		logger.Warnf("iam: 已配置 bootstrap_admin 但未设置 OPSHUB_BOOTSTRAP_CIPHER_KEY / bootstrap_cipher_key，跳过引导管理员创建")
		return
	}
	plain, err := bootstrapcipher.Decrypt(master, cipher)
	if err != nil {
		logger.Errorf("iam: 解密 bootstrap 管理员密码失败: %v", err)
		return
	}
	if _, err := store.Client().Users().Get(ctx, username); err == nil {
		return
	}
	hashed, err := passhash.Hash(plain)
	if err != nil {
		logger.Errorf("iam: bootstrap 管理员密码哈希失败: %v", err)
		return
	}
	email := strings.TrimSpace(ac.BootstrapAdminEmail)
	if email == "" {
		email = username + "@bootstrap.opshub.local"
	}
	u := &store.User{
		Username: username,
		Password: hashed,
		Email:    email,
		Phone:    "",
		IsAdmin:  true,
		Status:   1,
	}
	if err := store.Client().Users().Create(ctx, u); err != nil {
		logger.Errorf("iam: 创建 bootstrap 管理员失败: %v", err)
		return
	}
	logger.Infof("iam: 已从配置创建引导管理员 user=%q", username)
}
