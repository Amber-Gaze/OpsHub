package store

import (
	"context"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/passhash"
)

type User struct {
	ID        int64  `gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Username  string `gorm:"column:username;size:128;uniqueIndex;not null"`
	Password  string `gorm:"column:password;size:255;not null"`
	Email     string `gorm:"column:email;size:128;not null"`
	Phone     string `gorm:"column:phone;size:32"`
	IsAdmin   bool   `gorm:"column:is_admin;default:false"`
	Status    int    `gorm:"column:status;default:1"`
	LoginedAt int64  `gorm:"column:logined_at"`
}

func (u *User) TableName() string {
	return "user"
}

type StoreError struct {
	Message string
}

func (e *StoreError) Error() string {
	return e.Message
}

var (
	ErrInvalidPassword = &StoreError{Message: "invalid password"}
)

// ComparePassword 校验密码：支持 bcrypt 与历史明文（登录后会自动升级为 bcrypt）。
func (u *User) ComparePassword(password string) error {
	if err := passhash.Compare(u.Password, password); err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// UserStore defines the user storage interface.
type UserStore interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, username string) error
	Get(ctx context.Context, username string) (*User, error)
	List(ctx context.Context) ([]*User, error)
}
