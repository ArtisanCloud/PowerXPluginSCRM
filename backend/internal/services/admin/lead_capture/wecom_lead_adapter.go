package lead_capture

import (
	"context"
	"strings"
	"time"
)

// WeComLeadRecord is a normalized lead payload fetched from WeCom.
type WeComLeadRecord struct {
	ExternalLeadID string
	DisplayName    string
	Phone          string
	Email          string
	OccurredAt     time.Time
}

// WeComLeadAdapter fetches leads from WeCom channel accounts.
type WeComLeadAdapter interface {
	FetchLeads(ctx context.Context, req TriggerSyncRequest, channelAccountUUID string) ([]WeComLeadRecord, error)
}

// DefaultWeComLeadAdapter is a minimal adapter placeholder for MVP.
type DefaultWeComLeadAdapter struct{}

func NewDefaultWeComLeadAdapter() *DefaultWeComLeadAdapter {
	return &DefaultWeComLeadAdapter{}
}

func (a *DefaultWeComLeadAdapter) FetchLeads(_ context.Context, _ TriggerSyncRequest, _ string) ([]WeComLeadRecord, error) {
	_ = a
	return []WeComLeadRecord{}, nil
}

func normalizeWeComLeadRecord(in WeComLeadRecord) WeComLeadRecord {
	in.ExternalLeadID = strings.TrimSpace(in.ExternalLeadID)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now().UTC()
	}
	return in
}
