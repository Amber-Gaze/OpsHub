package store

import "context"

// ServiceModule 服务账号的「模块订阅」记录：服务订阅哪些业务/模块即可拉取对应配置（只读）。
// 订阅由管理员在 /services/{name}/modules 注册，注册即授予该模块的 read 权限（scope 联动）。
type ServiceModule struct {
	ID        int64  `gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Username  string `gorm:"column:username;size:128;index;not null"` // 服务账号
	Business  string `gorm:"column:business;size:64;not null"`        // 业务
	Module    string `gorm:"column:module;size:64;not null"`          // 模块（可为空=整业务）
	Path      string `gorm:"column:path;size:255;not null"`           // 归一化路径 business/module 或 business
	CreatedAt int64  `gorm:"column:created_at"`
}

// TableName 指定 gorm 表名。
func (s *ServiceModule) TableName() string {
	return "service_module"
}

// ServiceModuleStore 服务模块订阅存储接口。
type ServiceModuleStore interface {
	Create(ctx context.Context, sm *ServiceModule) error
	DeleteByPath(ctx context.Context, username, path string) error
	DeleteByUsername(ctx context.Context, username string) error
	GetByPath(ctx context.Context, username, path string) (*ServiceModule, error)
	List(ctx context.Context, username string) ([]*ServiceModule, error)
}
