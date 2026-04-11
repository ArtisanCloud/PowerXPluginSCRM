package lead_capture

import (
	"context"
	"errors"
	"strconv"
	"strings"

	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"gorm.io/gorm"
)

var ErrAssigneeNotBound = errors.New("assignee is not bound to source member")

// AssignmentService validates assignment pre-conditions.
type AssignmentService struct{}

func NewAssignmentService() *AssignmentService {
	return &AssignmentService{}
}

func (s *AssignmentService) EnsureMemberBound(ctx context.Context, tx *gorm.DB, tenantUUID string, memberID uint64) error {
	_ = s
	if tx == nil {
		return errors.New("database transaction is nil")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" || memberID == 0 {
		return ErrAssigneeNotBound
	}
	var count int64
	err := tx.WithContext(ctx).
		Model(&orgmodel.MemberBinding{}).
		Where(
			"tenant_uuid = ? AND main_member_id = ?",
			tenantUUID,
			strconv.FormatUint(memberID, 10),
		).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrAssigneeNotBound
	}
	return nil
}
