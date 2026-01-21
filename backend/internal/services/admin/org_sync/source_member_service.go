package org_sync

import (
	"context"
	"errors"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
)

// SourceMemberService handles source member queries.
type SourceMemberService struct {
	repo *orgrepo.SourceMemberRepository
}

func NewSourceMemberService(repo *orgrepo.SourceMemberRepository) *SourceMemberService {
	return &SourceMemberService{repo: repo}
}

func (s *SourceMemberService) List(ctx context.Context, tenantUUID, sourceAccountUUID string, status *string, q *string) ([]*model.SourceMember, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source member repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.repo.ListByAccount(ctx, tenantUUID, sourceAccountUUID, status, q)
}
