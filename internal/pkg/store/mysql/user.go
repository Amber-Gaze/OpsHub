package mysql

import (
	"context"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"gorm.io/gorm"
)

type users struct {
	db *gorm.DB
}

func newUsers(ds *datastore) *users {
	return &users{ds.db}
}

// Create creates a new user account.
func (u *users) Create(ctx context.Context, user *store.User) error {
	return u.db.Create(&user).Error
}

// Update updates an user account information.
func (u *users) Update(ctx context.Context, user *store.User) error {
	return u.db.Save(user).Error
}

// Delete deletes the user by the user identifier.
func (u *users) Delete(ctx context.Context, username string) error {
	err := u.db.Where("username = ?", username).Delete(&store.User{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	return nil
}

func (u *users) Get(ctx context.Context, username string) (*store.User, error) {
	user := &store.User{}
	err := u.db.Where("username = ? AND status = 1", username).First(&user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *users) List(ctx context.Context) ([]*store.User, error) {
	var users []*store.User
	err := u.db.Where("status = 1").Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}
