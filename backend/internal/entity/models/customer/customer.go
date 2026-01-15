package customer

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// CustomerAccount 存储 mini-app Customer 的本地登录凭证（Skeleton/local 模式）。
// Delegated 模式下该表不需要、也不应被写入。
type CustomerAccount struct {
	models.BaseModel
	CustomerUUID  string            `gorm:"column:customer_uuid;type:uuid;not null;index" json:"customer_uuid"`
	Email         string            `gorm:"column:email;type:varchar(255);index" json:"email,omitempty"`
	Phone         string            `gorm:"column:phone;type:varchar(32);index" json:"phone,omitempty"`
	PasswordHash  string            `gorm:"column:password_hash;type:text" json:"password_hash,omitempty"`
	Status        string            `gorm:"column:status;type:varchar(32);not null;default:'active'" json:"status"`
	Metadata      datatypes.JSONMap `gorm:"column:metadata;type:jsonb;default:'{}'::jsonb" json:"metadata,omitempty"`
	EmailVerified bool              `gorm:"column:email_verified;not null;default:false" json:"email_verified"`
	PhoneVerified bool              `gorm:"column:phone_verified;not null;default:false" json:"phone_verified"`
}

func (c *CustomerAccount) TableName() string {
	return models.S(models.TableCustomerAccounts)
}
