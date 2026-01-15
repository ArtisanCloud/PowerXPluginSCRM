package models

import (
	"time"
)

// PluginCredential 存储每个租户-插件的长期凭证（密文）
// 仅保存加密后的 secret，不保存明文。
// 表结构：
// - tenant_uuid + plugin_id 唯一
// - secret_ciphertext 使用 bytea 保存密文
// - iv_nonce 为 AES-GCM 的随机 nonce
// - key_version 用于将来主密钥轮换
type PluginCredential struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantUUID       string     `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_pc_tenant_plugin,unique" json:"tenant_uuid"`
	PluginID         string     `gorm:"type:varchar(128);not null;index:idx_pc_tenant_plugin,unique" json:"plugin_id"`
	ClientID         string     `gorm:"type:varchar(255);not null" json:"client_id"`
	SecretCiphertext []byte     `gorm:"type:bytea;not null" json:"-"`
	IVNonce          []byte     `gorm:"type:bytea;not null" json:"-"`
	KeyVersion       int        `gorm:"type:int;not null;default:1" json:"key_version"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (PluginCredential) TableName() string { return S(TablePluginCredentials) }
