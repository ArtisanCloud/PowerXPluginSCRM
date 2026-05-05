package opportunity

import (
	opprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/opportunity"
)

// Service is a minimal foundational skeleton for Phase 2 wiring.
type Service struct {
	repo     opprepo.OpportunityRepository
	activity opprepo.OpportunityActivityRepository
}

func NewService(repo opprepo.OpportunityRepository, activity opprepo.OpportunityActivityRepository) *Service {
	return &Service{repo: repo, activity: activity}
}
