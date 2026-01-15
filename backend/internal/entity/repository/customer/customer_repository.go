package customer

import (
	"context"
	"errors"
	"strings"

	customerdomain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/customer"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/customer"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrCustomerExists   = errors.New("customer already exists")
)

type Repository struct {
	*repository.BaseRepository[customermodel.CustomerAccount]
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{BaseRepository: repository.NewBaseRepository[customermodel.CustomerAccount](db)}
}

func (r *Repository) FindByEmailOrPhone(ctx context.Context, tenantUUID string, login string) (*customermodel.CustomerAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	login = strings.TrimSpace(login)
	if tenantUUID == "" || login == "" {
		return nil, ErrCustomerNotFound
	}

	var out customermodel.CustomerAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND (email = ? OR phone = ?)", tenantUUID, login, login).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *Repository) ListTenantUUIDsByEmailOrPhone(ctx context.Context, login string) ([]string, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, ErrCustomerNotFound
	}

	var out []string
	err := r.DB.WithContext(ctx).
		Model(&customermodel.CustomerAccount{}).
		Select("DISTINCT tenant_uuid").
		Where("(email = ? OR phone = ?)", login, login).
		Where("status <> ?", string(customerdomain.CustomerStatusDeleted)).
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrCustomerNotFound
	}
	return out, nil
}

func (r *Repository) FindByCustomerUUID(ctx context.Context, tenantUUID string, customerUUID string) (*customermodel.CustomerAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	customerUUID = strings.ToLower(strings.TrimSpace(customerUUID))
	if tenantUUID == "" || customerUUID == "" {
		return nil, ErrCustomerNotFound
	}

	var out customermodel.CustomerAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND customer_uuid = ?", tenantUUID, customerUUID).
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *Repository) CreateCustomer(ctx context.Context, customer *customermodel.CustomerAccount) (*customermodel.CustomerAccount, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	if customer == nil {
		return nil, errors.New("customer is required")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(customer.TenantUuid))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}

	// Best-effort uniqueness check (email/phone).
	email := strings.TrimSpace(customer.Email)
	phone := strings.TrimSpace(customer.Phone)
	if email != "" || phone != "" {
		var count int64
		q := r.DB.WithContext(ctx).Model(&customermodel.CustomerAccount{}).Where("tenant_uuid = ?", tenantUUID)
		if email != "" && phone != "" {
			q = q.Where("(email = ? OR phone = ?)", email, phone)
		} else if email != "" {
			q = q.Where("email = ?", email)
		} else {
			q = q.Where("phone = ?", phone)
		}
		if err := q.Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrCustomerExists
		}
	}

	return r.Create(ctx, customer)
}

func (r *Repository) UpdateStatus(ctx context.Context, tenantUUID string, customerUUID string, status string) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	customerUUID = strings.ToLower(strings.TrimSpace(customerUUID))
	status = strings.TrimSpace(status)
	if tenantUUID == "" || customerUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	if status == "" {
		return errors.New("status is required")
	}

	res := r.DB.WithContext(ctx).
		Model(&customermodel.CustomerAccount{}).
		Where("tenant_uuid = ? AND customer_uuid = ?", tenantUUID, customerUUID).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCustomerNotFound
	}
	return nil
}
