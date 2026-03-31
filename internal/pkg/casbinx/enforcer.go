package casbinx

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// NewSyncedEnforcer 使用当前 MySQL 连接持久化 casbin_rule 表。
func NewSyncedEnforcer(db *gorm.DB) (*casbin.SyncedEnforcer, error) {
	a, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}
	m, err := model.NewModelFromString(ModelText)
	if err != nil {
		return nil, err
	}
	e, err := casbin.NewSyncedEnforcer(m, a)
	if err != nil {
		return nil, err
	}
	e.EnableAutoSave(true)
	if err := e.LoadPolicy(); err != nil {
		return nil, err
	}
	return e, nil
}
