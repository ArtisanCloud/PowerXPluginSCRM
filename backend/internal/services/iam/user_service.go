package iam

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserService struct {
	db    *gorm.DB
	audit *AuditService
}

func NewUserService(db *gorm.DB, audit *AuditService) *UserService {
	return &UserService{db: db, audit: audit}
}

type UserFilter struct {
	TenantUUID string
	Status     string
	Query      string
}

type UserView struct {
	ID           uint64     `json:"id"`
	UserID       uint64     `json:"user_id"`
	TenantUUID   string     `json:"tenant_uuid"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	DisplayName  string     `json:"display_name"`
	Username     string     `json:"username"`
	Status       string     `json:"status"`
	DepartmentID *uint64    `json:"department_id"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	Roles        []string   `json:"roles"`
}

type UserBulkImportResult struct {
	Created []*UserView           `json:"created"`
	Failed  []UserBulkImportError `json:"failed"`
}

type UserBulkImportError struct {
	Index   int    `json:"index"`
	Email   string `json:"email"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (s *UserService) List(ctx context.Context, filter UserFilter) ([]UserView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	tenantUUID := strings.TrimSpace(filter.TenantUUID)
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid required")
	}
	query := s.db.WithContext(ctx).
		Table(iamm.Member{}.TableName()+" u").
		Select(`u.id AS id, u.user_id AS user_id, u.tenant_uuid, u.username, u.status, u.department_id,
            u.last_login_at, u.created_at, COALESCE(u.display_name, a.display_name) AS display_name,
            a.email, a.phone`).
		Joins("JOIN "+iamm.User{}.TableName()+" a ON a.id = u.user_id").
		Where("u.tenant_uuid = ?", tenantUUID)
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("u.status = ?", status)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("(lower(a.email) LIKE ? OR lower(u.username) LIKE ? OR lower(a.display_name) LIKE ?)", like, like, like)
	}
	query = query.Order("u.created_at DESC")
	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []UserView{}
	for rows.Next() {
		var view UserView
		if err := s.db.ScanRows(rows, &view); err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	if len(result) == 0 {
		return result, nil
	}
	ids := make([]uint64, 0, len(result))
	for _, view := range result {
		ids = append(ids, view.ID)
	}
	roleMap, err := s.userRolesMap(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range result {
		if roles, ok := roleMap[result[i].ID]; ok {
			result[i].Roles = roles
		}
	}
	return result, nil
}

func (s *UserService) userRolesMap(ctx context.Context, userIDs []uint64) (map[uint64][]string, error) {
	roleMap := make(map[uint64][]string)
	if s == nil || s.db == nil || len(userIDs) == 0 {
		return roleMap, nil
	}
	dedup := make([]uint64, 0, len(userIDs))
	seen := make(map[uint64]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		dedup = append(dedup, id)
	}
	if len(dedup) == 0 {
		return roleMap, nil
	}
	rows := []struct {
		UserID uint64
		Code   string
	}{}
	if err := s.db.WithContext(ctx).
		Table(iamm.MemberRole{}.TableName()+" ur").
		Select("ur.member_id AS user_id, r.code").
		Joins("JOIN "+iamm.Role{}.TableName()+" r ON r.id = ur.role_id").
		Where("ur.member_id IN ?", dedup).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		roleMap[row.UserID] = append(roleMap[row.UserID], row.Code)
	}
	return roleMap, nil
}

func (s *UserService) assignRoles(ctx context.Context, tx *gorm.DB, tenantUUID string, userID uint64, roleIDs []uint64) error {
	if tx == nil {
		return errors.New("iam: db is nil")
	}
	cleaned := uniqueUint64(roleIDs)
	if len(cleaned) == 0 {
		return tx.WithContext(ctx).Where("member_id = ?", userID).Delete(&iamm.MemberRole{}).Error
	}
	var count int64
	if err := tx.WithContext(ctx).Model(&iamm.Role{}).
		Where("tenant_uuid = ?", tenantUUID).
		Where("id IN ?", cleaned).
		Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(cleaned)) {
		return fmt.Errorf("role ids do not belong to tenant")
	}
	if err := tx.WithContext(ctx).Where("member_id = ?", userID).Delete(&iamm.MemberRole{}).Error; err != nil {
		return err
	}
	for _, roleID := range cleaned {
		rel := &iamm.MemberRole{UserID: userID, RoleID: roleID}
		if err := tx.WithContext(ctx).Create(rel).Error; err != nil {
			return err
		}
	}
	return nil
}

type CreateUserInput struct {
	TenantUUID   string
	Email        string
	DisplayName  string
	Username     string
	Phone        string
	DepartmentID *uint64
	Status       string
	ActorID      *uint64
	Roles        []uint64
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*UserView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(input.TenantUUID))
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid required")
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, errors.New("email required")
	}
	username := strings.TrimSpace(input.Username)
	if username == "" {
		username = slugify(strings.Split(email, "@")[0])
	}
	status := normalizeUserStatus(input.Status)
	if status == "" {
		status = iamm.StatusActive
	}

	var dept *iamm.Department
	if input.DepartmentID != nil {
		dept = &iamm.Department{}
		if err := s.db.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", *input.DepartmentID, tenantUUID).First(dept).Error; err != nil {
			return nil, err
		}
	}

	var created *UserView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := ensureAccount(ctx, tx, email, input.DisplayName, input.Phone)
		if err != nil {
			return err
		}
		var exists int64
		if err := tx.Model(&iamm.Member{}).Where("tenant_uuid = ? AND user_id = ?", tenantUUID, account.ID).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return fmt.Errorf("user already exists for tenant")
		}
		var usernameExists int64
		if err := tx.Model(&iamm.Member{}).
			Where("tenant_uuid = ? AND lower(username) = ?", tenantUUID, strings.ToLower(username)).
			Count(&usernameExists).Error; err != nil {
			return err
		}
		if usernameExists > 0 {
			return fmt.Errorf("username already exists")
		}
		record := &iamm.Member{
			BaseModel:   basemodels.BaseModel{TenantUuid: tenantUUID},
			UserID:      account.ID,
			Username:    username,
			DisplayName: strings.TrimSpace(input.DisplayName),
			Status:      status,
			Meta:        datatypes.JSONMap{},
		}
		if dept != nil {
			record.DepartmentID = &dept.ID
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		if len(input.Roles) > 0 {
			if err := s.assignRoles(ctx, tx, tenantUUID, record.ID, input.Roles); err != nil {
				return err
			}
		}
		created = &UserView{
			ID:           record.ID,
			UserID:       account.ID,
			TenantUUID:   tenantUUID,
			Email:        account.Email,
			Phone:        account.Phone,
			DisplayName:  firstNonEmpty(record.DisplayName, account.DisplayName),
			Username:     record.Username,
			Status:       record.Status,
			DepartmentID: record.DepartmentID,
			CreatedAt:    record.CreatedAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if created != nil {
		roleMap, err := s.userRolesMap(ctx, []uint64{created.ID})
		if err == nil {
			created.Roles = roleMap[created.ID]
		}
	}
	if s.audit != nil && created != nil {
		diff := map[string]any{
			"member_id": created.ID,
			"user_id":   created.UserID,
			"username":  created.Username,
			"status":    created.Status,
		}
		if len(created.Roles) > 0 {
			diff["roles"] = created.Roles
		}
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    tenantUUID,
			ActorMemberID: input.ActorID,
			Action:        "create",
			Resource:      "iam.user",
			Diff:          diff,
		})
	}
	return created, nil
}

type UpdateUserInput struct {
	DisplayName  string
	Status       string
	DepartmentID *uint64
	ActorID      *uint64
	Roles        []uint64
	ReplaceRoles bool
}

func (s *UserService) BulkImport(ctx context.Context, inputs []CreateUserInput) (*UserBulkImportResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	result := &UserBulkImportResult{Created: []*UserView{}, Failed: []UserBulkImportError{}}
	var firstErr error
	for idx, payload := range inputs {
		view, err := s.Create(ctx, payload)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			result.Failed = append(result.Failed, UserBulkImportError{
				Index:   idx,
				Email:   payload.Email,
				Message: err.Error(),
				Err:     err,
			})
			continue
		}
		result.Created = append(result.Created, view)
	}
	return result, firstErr
}

func (s *UserService) Update(ctx context.Context, id uint64, input UpdateUserInput) (*UserView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	var user iamm.Member
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	var dept *iamm.Department
	if input.DepartmentID != nil && *input.DepartmentID != 0 {
		dept = &iamm.Department{}
		if err := s.db.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", *input.DepartmentID, user.TenantUuid).First(dept).Error; err != nil {
			return nil, err
		}
	}

	updates := map[string]any{}
	if name := strings.TrimSpace(input.DisplayName); name != "" && name != user.DisplayName {
		updates["display_name"] = name
	}
	var statusChanged bool
	if status := normalizeUserStatus(input.Status); status != "" && status != user.Status {
		updates["status"] = status
		statusChanged = true
	}
	if input.DepartmentID != nil {
		if *input.DepartmentID == 0 {
			updates["department_id"] = nil
		} else if user.DepartmentID == nil || *user.DepartmentID != *input.DepartmentID {
			updates["department_id"] = dept.ID
		}
	}

	var rolesChanged bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return err
			}
		}
		if input.ReplaceRoles {
			if err := s.assignRoles(ctx, tx, user.TenantUuid, user.ID, input.Roles); err != nil {
				return err
			}
			rolesChanged = true
		}
		return tx.Where("id = ?", user.ID).First(&user).Error
	})
	if err != nil {
		return nil, err
	}

	var account iamm.User
	if err := s.db.WithContext(ctx).Where("id = ?", user.UserID).First(&account).Error; err != nil {
		return nil, err
	}
	roleMap, err := s.userRolesMap(ctx, []uint64{user.ID})
	if err != nil {
		return nil, err
	}
	view := &UserView{
		ID:           user.ID,
		UserID:       account.ID,
		TenantUUID:   user.TenantUuid,
		Email:        account.Email,
		Phone:        account.Phone,
		DisplayName:  firstNonEmpty(user.DisplayName, account.DisplayName),
		Username:     user.Username,
		Status:       user.Status,
		DepartmentID: user.DepartmentID,
		LastLoginAt:  user.LastLoginAt,
		CreatedAt:    user.CreatedAt,
		Roles:        roleMap[user.ID],
	}
	if s.audit != nil && (len(updates) > 0 || rolesChanged) {
		diff := make(map[string]any, len(updates))
		for k, v := range updates {
			diff[k] = v
		}
		if rolesChanged {
			diff["roles"] = view.Roles
		}
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    user.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.user",
			Diff:          diff,
		})
	}
	if (statusChanged && user.Status != iamm.StatusActive) || rolesChanged {
		_ = revokeMemberSession(ctx, s.db, user.ID)
	}
	return view, nil
}

func ensureAccount(ctx context.Context, tx *gorm.DB, email, displayName, phone string) (*iamm.User, error) {
	var account iamm.User
	if err := tx.WithContext(ctx).Where("lower(email) = ?", strings.ToLower(email)).First(&account).Error; err == nil {
		return &account, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	passwordHash, err := randomPasswordHash()
	if err != nil {
		return nil, err
	}
	account = iamm.User{
		Email:        strings.ToLower(email),
		DisplayName:  strings.TrimSpace(displayName),
		Status:       iamm.StatusActive,
		PasswordHash: passwordHash,
		Phone:        strings.TrimSpace(phone),
		Meta:         datatypes.JSONMap{},
	}
	if account.DisplayName == "" {
		account.DisplayName = account.Email
	}
	if err := tx.WithContext(ctx).Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func randomPasswordHash() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(buf)
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func normalizeUserStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", iamm.StatusActive:
		return iamm.StatusActive
	case iamm.StatusDisabled:
		return iamm.StatusDisabled
	case iamm.StatusLocked:
		return iamm.StatusLocked
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	return value
}

func uniqueUint64(values []uint64) []uint64 {
	if len(values) == 0 {
		return values
	}
	result := make([]uint64, 0, len(values))
	seen := make(map[uint64]struct{}, len(values))
	for _, v := range values {
		if v == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}
