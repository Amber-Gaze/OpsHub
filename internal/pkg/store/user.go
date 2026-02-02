package store

import "context"

type User struct {
	ID        string `gorm:"column:id;primaryKey"`
	Username  string `gorm:"column:username;uniqueIndex;not null"`
	Password  string `gorm:"column:password;not null"`
	Email     string `gorm:"column:email;not null"`
	Phone     string `gorm:"column:phone"`
	IsAdmin   bool   `gorm:"column:is_admin;default:false"`
	Status    int    `gorm:"column:status;default:1"`
	LoginedAt int64  `gorm:"column:logined_at"`
}

func (u *User) TableName() string {
	return "user"
}

// UserStore defines the user storage interface.
type UserStore interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, username string) error
	Get(ctx context.Context, username string) (*User, error)
	List(ctx context.Context) ([]*User, error)
}
