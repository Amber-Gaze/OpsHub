package store

import "context"

type User struct {
	ID        int64  `gorm:"column:id;primaryKey;AUTO_INCREMENT"`
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

type StoreError struct {
	Message string
}

func (e *StoreError) Error() string {
	return e.Message
}

var (
	ErrInvalidPassword = &StoreError{Message: "invalid password"}
)

// ComparePassword compares the given password with the user's password.

func (u *User) ComparePassword(password string) error {
	if u.Password != password {
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
