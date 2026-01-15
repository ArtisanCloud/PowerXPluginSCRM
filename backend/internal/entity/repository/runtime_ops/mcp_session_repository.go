package runtime_ops

import (
	"context"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"gorm.io/gorm"
)

// MCPSessionRepository manages MCP session persistence with tenant isolation.
type MCPSessionRepository struct {
	*repository.BaseRepository[model.MCPSession]
}

// NewMCPSessionRepository constructs a repository backed by BaseRepository.
func NewMCPSessionRepository(db *gorm.DB) *MCPSessionRepository {
	return &MCPSessionRepository{
		BaseRepository: repository.NewBaseRepository[model.MCPSession](db),
	}
}

// Create inserts a session record, ensuring tenant scope consistency.
func (r *MCPSessionRepository) Create(ctx context.Context, session *model.MCPSession) (*model.MCPSession, error) {
	tenantID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return nil, err
	}

	if session.TenantUuid == "" {
		session.TenantUuid = tenantID
	} else if session.TenantUuid != tenantID {
		return nil, gorm.ErrInvalidData
	}

	return r.BaseRepository.Create(ctx, session)
}

// UpdateFields updates session fields while enforcing tenant filter.
func (r *MCPSessionRepository) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) (*model.MCPSession, error) {
	tenantID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := r.BaseRepository.Patch(ctx, map[string]interface{}{
		"id":          id,
		"tenant_uuid": tenantID,
	}, fields); err != nil {
		return nil, err
	}
	var session model.MCPSession
	if err := r.DB.WithContext(ctx).
		Where("id = ? AND tenant_uuid = ?", id, tenantID).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}
