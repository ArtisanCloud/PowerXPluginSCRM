package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	"gorm.io/gorm"
)

// ApprovalStatus 审批流程中的状态枚举。
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "PENDING"
	ApprovalStatusApproved ApprovalStatus = "APPROVED"
	ApprovalStatusRejected ApprovalStatus = "REJECTED"
)

// ErrApprovalNotFound 指示审批记录不存在。
var ErrApprovalNotFound = errors.New("integration change approval not found")

// ErrApprovalInvalid 表示审批记录缺少必要字段。
var ErrApprovalInvalid = errors.New("integration change approval invalid")

// StatusUpdate 描述审批状态变更。
type StatusUpdate struct {
	Status     ApprovalStatus
	ReviewedBy string
	Reason     string
}

// ApprovalRepository 管理配置审批的持久化。
type ApprovalRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewApprovalRepository 构造审批仓储。
func NewApprovalRepository(db *gorm.DB) *ApprovalRepository {
	return &ApprovalRepository{
		db:  db,
		now: time.Now,
	}
}

// WithNow 允许在测试中覆盖时间源。
func (r *ApprovalRepository) WithNow(now func() time.Time) {
	if now != nil {
		r.now = now
	}
}

// Create 插入新的审批请求。
func (r *ApprovalRepository) Create(ctx context.Context, approval *model.ChangeApproval) (*model.ChangeApproval, error) {
	if approval == nil {
		return nil, ErrApprovalInvalid
	}
	if strings.TrimSpace(approval.TargetType) == "" || strings.TrimSpace(approval.TargetID) == "" {
		return nil, fmt.Errorf("%w: target_type and target_id are required", ErrApprovalInvalid)
	}
	if strings.TrimSpace(approval.SubmittedBy) == "" {
		return nil, fmt.Errorf("%w: submitted_by is required", ErrApprovalInvalid)
	}

	now := r.now().UTC()
	if approval.Status == "" {
		approval.Status = string(ApprovalStatusPending)
	}
	if approval.SubmittedAt.IsZero() {
		approval.SubmittedAt = now
	}
	approval.CreatedAt = now
	approval.UpdatedAt = now

	if err := r.db.WithContext(ctx).Create(approval).Error; err != nil {
		return nil, err
	}

	return r.GetByID(ctx, approval.ID)
}

// GetByID 查询指定审批记录。
func (r *ApprovalRepository) GetByID(ctx context.Context, id string) (*model.ChangeApproval, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", ErrApprovalInvalid)
	}
	var approval model.ChangeApproval
	err := r.db.WithContext(ctx).First(&approval, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApprovalNotFound
		}
		return nil, err
	}
	return &approval, nil
}

// FindPendingByTarget 列出同一目标下的待审批记录。
func (r *ApprovalRepository) FindPendingByTarget(ctx context.Context, targetType, targetID string) ([]model.ChangeApproval, error) {
	query := r.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ? AND status = ?", strings.TrimSpace(targetType), strings.TrimSpace(targetID), string(ApprovalStatusPending)).
		Order("submitted_at ASC")

	var approvals []model.ChangeApproval
	if err := query.Find(&approvals).Error; err != nil {
		return nil, err
	}
	return approvals, nil
}

// ListPending 返回待审批列表。
func (r *ApprovalRepository) ListPending(ctx context.Context, targetType string, limit int) ([]model.ChangeApproval, error) {
	query := r.db.WithContext(ctx).Where("status = ?", string(ApprovalStatusPending)).Order("submitted_at ASC")
	if strings.TrimSpace(targetType) != "" {
		query = query.Where("target_type = ?", strings.TrimSpace(targetType))
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var approvals []model.ChangeApproval
	if err := query.Find(&approvals).Error; err != nil {
		return nil, err
	}
	return approvals, nil
}

// UpdateStatus 更新审批状态及复核信息。
func (r *ApprovalRepository) UpdateStatus(ctx context.Context, id string, update StatusUpdate) (*model.ChangeApproval, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", ErrApprovalInvalid)
	}
	if update.Status == "" {
		return nil, fmt.Errorf("%w: status is required", ErrApprovalInvalid)
	}

	now := r.now().UTC()
	values := map[string]any{
		"status":     string(update.Status),
		"updated_at": now,
	}

	if update.Status != ApprovalStatusPending {
		values["reviewed_at"] = now
		values["reviewed_by"] = strings.TrimSpace(update.ReviewedBy)
		if strings.TrimSpace(update.Reason) != "" {
			values["reason"] = update.Reason
		}
	}

	res := r.db.WithContext(ctx).
		Model(&model.ChangeApproval{}).
		Where("id = ?", id).
		Updates(values)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrApprovalNotFound
	}

	return r.GetByID(ctx, id)
}
