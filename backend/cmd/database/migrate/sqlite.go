package migrate

// sqliteTables 定义 SQLite 模式下需要自动迁移的表结构（使用原生类型）。
type sqliteTemplate struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	TenantUuid    string `gorm:"column:tenant_uuid;type:uuid;index;not null"`
	Name        string `gorm:"column:name;type:text;not null"`
	Description string `gorm:"column:description;type:text"`
	Content     string `gorm:"column:content;type:text"`
	CreatedAt   int64  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   int64  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   *int64 `gorm:"column:deleted_at"`
}

func (sqliteTemplate) TableName() string { return "template" }

type sqlitePluginCredential struct {
	ID               uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	TenantUuid         string `gorm:"column:tenant_uuid;type:uuid;index;not null"`
	PluginID         string `gorm:"column:plugin_id;type:text;not null"`
	ClientID         string `gorm:"column:client_id;type:text;not null"`
	SecretCiphertext []byte `gorm:"column:secret_ciphertext;type:blob;not null"`
	IVNonce          []byte `gorm:"column:iv_nonce;type:blob;not null"`
	KeyVersion       int    `gorm:"column:key_version;type:integer;not null;default:1"`
	CreatedAt        int64  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        int64  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt        *int64 `gorm:"column:deleted_at"`
}

func (sqlitePluginCredential) TableName() string { return "plugin_credentials" }
