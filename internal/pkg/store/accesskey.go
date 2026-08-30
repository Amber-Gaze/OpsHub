package store

import "context"

// AccessKey 服务账号的访问凭证（AccessKeyID + AccessKeySecret），用于程序化鉴权。
// 归属某个用户（服务账号）；下游服务用 AccessKeySecret 自签 JWT（header kid=AccessKeyID），
// iam 按 kid 查库验签，身份以 AccessKey 归属的用户为准。
// 设计参照 marmotedu/iam 的 secret、AWS Access Key：公开标识 + 私密签名密钥，可独立创建/轮换/吊销。
type AccessKey struct {
	ID              int64  `gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Username        string `gorm:"column:username;size:128;index;not null"`           // 归属的服务账号（用户名）
	AccessKeyID     string `gorm:"column:access_key_id;size:64;uniqueIndex;not null"` // 公开标识（JWT header kid）
	AccessKeySecret string `gorm:"column:access_key_secret;size:128;not null"`        // 私密签名密钥（HMAC）
	Description     string `gorm:"column:description;size:255"`
	Status          int    `gorm:"column:status;default:1"`  // 1=启用 0=禁用
	Expires         int64  `gorm:"column:expires;default:0"` // Unix 秒；0=永不过期
	CreatedAt       int64  `gorm:"column:created_at"`
	UpdatedAt       int64  `gorm:"column:updated_at"`
}

// TableName 指定 gorm 表名。
func (a *AccessKey) TableName() string {
	return "access_key"
}

// AccessKeyStore 访问凭证存储接口。
type AccessKeyStore interface {
	Create(ctx context.Context, key *AccessKey) error
	Update(ctx context.Context, key *AccessKey) error
	Delete(ctx context.Context, username, accessKeyID string) error
	Get(ctx context.Context, username, accessKeyID string) (*AccessKey, error)
	// GetByKeyID 按 AccessKeyID 查询（JWT 验签用，跨用户）。
	GetByKeyID(ctx context.Context, accessKeyID string) (*AccessKey, error)
	List(ctx context.Context, username string) ([]*AccessKey, error)
}
