package infrastructure

import (
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"

	mysqladapter "github.com/casbin/mysql-adapter"
)

func InitCasbin(dbDSN string) *casbin.Enforcer {
	// load Casbin model
	m, err := model.NewModelFromFile(constants.CasbinModelPath)
	if err != nil {
		log.Fatalf("casbin model load error: %v", err)
	}

	// use MySQL Adapter
	adapter, err := mysqladapter.NewAdapter("mysql", dbDSN)
	if err != nil {
		log.Fatalf("casbin db adapter error:%v", err)
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		log.Fatalf("casbin new enforcer error:%v", err)
	}

	enforcer.LoadPolicy()
	return enforcer
}
