package integration

import (
	"time"

	"gorm.io/datatypes"
)

// IdempotencyRecord 表示幂等键的持久化记录。
type IdempotencyRecord struct {
	Key         string            `gorm:"column:key;primaryKey" json:"key"`
	TenantUuid  string            `gorm:"column:tenant_uuid;not null" json:"tenant_uuid"`
	Scope       string            `gorm:"column:scope" json:"scope"`
	Operation   string            `gorm:"column:operation" json:"operation"`
	PayloadHash string            `gorm:"column:payload_hash" json:"payload_hash"`
	Response    datatypes.JSON    `gorm:"column:response_data" json:"response"`
	Metadata    datatypes.JSONMap `gorm:"column:metadata" json:"metadata"`
	ExpiresAt   *time.Time        `gorm:"column:expires_at" json:"expires_at"`
	CreatedAt   time.Time         `gorm:"column:created_at" json:"created_at"`
}

// TableName implements gorm's tabler interface.
func (IdempotencyRecord) TableName() string {
	return "integration_idempotency_records"
}
