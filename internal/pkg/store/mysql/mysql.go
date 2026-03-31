package mysql

import (
	"context"
	"fmt"
	"sync"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
	"github.com/Amber-Gaze/OpsHub/internal/pkg/store"
	"github.com/Amber-Gaze/OpsHub/pkg/db"
	"github.com/Amber-Gaze/OpsHub/pkg/logger"
	"gorm.io/gorm"
)

type datastore struct {
	db *gorm.DB
}

func (ds *datastore) Users() store.UserStore {
	return newUsers(ds)
}

func (ds *datastore) Close() error {
	db, err := ds.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}

func (ds *datastore) Ping(ctx context.Context) error {
	sqlDB, err := ds.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

var (
	mysqlFactory store.Factory
	once         sync.Once
)

// GetMySQLFactoryOr create mysql factory with the given config.
func GetMySQLFactoryOr(opts *options.MySQLOptions) (store.Factory, error) {
	if opts == nil && mysqlFactory == nil {
		return nil, fmt.Errorf("failed to get mysql store fatory")
	}

	var err error
	var dbIns *gorm.DB
	once.Do(func() {
		options := &db.Options{
			Host:                  opts.Host,
			Username:              opts.Username,
			Password:              opts.Password,
			Database:              opts.Database,
			MaxIdleConnections:    opts.MaxIdleConnections,
			MaxOpenConnections:    opts.MaxOpenConnections,
			MaxConnectionLifeTime: opts.MaxConnectionLifeTime,
			LogLevel:              opts.LogLevel,
			Logger:                logger.NewZapGormLogger(),
		}
		dbIns, err = db.New(options)

		mysqlFactory = &datastore{dbIns}
	})

	if mysqlFactory == nil || err != nil {
		return nil, fmt.Errorf("failed to get mysql store fatory, mysqlFactory: %+v, error: %w", mysqlFactory, err)
	}

	return mysqlFactory, nil
}

// GetGORM 返回已初始化的 *gorm.DB（供 Casbin 等组件复用连接）。
func GetGORM() (*gorm.DB, error) {
	if mysqlFactory == nil {
		return nil, fmt.Errorf("mysql store not initialized")
	}
	ds, ok := mysqlFactory.(*datastore)
	if !ok {
		return nil, fmt.Errorf("unexpected mysql factory type")
	}
	return ds.db, nil
}
