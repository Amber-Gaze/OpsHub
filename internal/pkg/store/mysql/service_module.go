package mysql

import (
	"context"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"gorm.io/gorm"
)

type serviceModules struct {
	db *gorm.DB
}

func newServiceModules(ds *datastore) *serviceModules {
	return &serviceModules{ds.db}
}

func (s *serviceModules) Create(ctx context.Context, sm *store.ServiceModule) error {
	return s.db.Create(&sm).Error
}

func (s *serviceModules) DeleteByPath(ctx context.Context, username, path string) error {
	err := s.db.Where("username = ? AND path = ?", username, path).Delete(&store.ServiceModule{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (s *serviceModules) DeleteByUsername(ctx context.Context, username string) error {
	err := s.db.Where("username = ?", username).Delete(&store.ServiceModule{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func (s *serviceModules) GetByPath(ctx context.Context, username, path string) (*store.ServiceModule, error) {
	sm := &store.ServiceModule{}
	if err := s.db.Where("username = ? AND path = ?", username, path).First(sm).Error; err != nil {
		return nil, err
	}
	return sm, nil
}

func (s *serviceModules) List(ctx context.Context, username string) ([]*store.ServiceModule, error) {
	var list []*store.ServiceModule
	if err := s.db.Where("username = ?", username).Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
