package mysql

import (
	"context"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"gorm.io/gorm"
)

type accesskeys struct {
	db *gorm.DB
}

func newAccessKeys(ds *datastore) *accesskeys {
	return &accesskeys{ds.db}
}

func (a *accesskeys) Create(ctx context.Context, key *store.AccessKey) error {
	return a.db.Create(&key).Error
}

func (a *accesskeys) Update(ctx context.Context, key *store.AccessKey) error {
	return a.db.Save(key).Error
}

func (a *accesskeys) Delete(ctx context.Context, username, accessKeyID string) error {
	err := a.db.Where("username = ? AND access_key_id = ?", username, accessKeyID).Delete(&store.AccessKey{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (a *accesskeys) Get(ctx context.Context, username, accessKeyID string) (*store.AccessKey, error) {
	key := &store.AccessKey{}
	if err := a.db.Where("username = ? AND access_key_id = ? AND status = 1", username, accessKeyID).First(key).Error; err != nil {
		return nil, err
	}
	return key, nil
}

func (a *accesskeys) GetByKeyID(ctx context.Context, accessKeyID string) (*store.AccessKey, error) {
	key := &store.AccessKey{}
	if err := a.db.Where("access_key_id = ? AND status = 1", accessKeyID).First(key).Error; err != nil {
		return nil, err
	}
	return key, nil
}

func (a *accesskeys) List(ctx context.Context, username string) ([]*store.AccessKey, error) {
	var keys []*store.AccessKey
	if err := a.db.Where("username = ?", username).Order("id desc").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}
