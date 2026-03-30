package passhash

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = bcrypt.DefaultCost

var ErrMismatch = errors.New("password mismatch")

// Hash 生成 bcrypt 哈希，写入数据库 password 字段。
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// LooksBcrypt 判断库存密码是否为 bcrypt 格式（含历史 $2a/$2b/$2y）。
func LooksBcrypt(stored string) bool {
	if len(stored) < 4 {
		return false
	}
	return stored[0] == '$' && stored[1] == '2'
}

// Compare 校验明文：若库存为 bcrypt 则用 bcrypt；否则视为旧版明文（登录成功后会自动升级为哈希）。
func Compare(stored, password string) error {
	if LooksBcrypt(stored) {
		if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)); err != nil {
			return ErrMismatch
		}
		return nil
	}
	if stored == password {
		return nil
	}
	return ErrMismatch
}
