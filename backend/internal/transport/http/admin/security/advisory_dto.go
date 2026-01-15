package security

import (
	"time"

	secmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
)

// AdvisoryResponse represents the API payload for a vulnerability advisory.
type AdvisoryResponse struct {
	ID               string   `json:"id"`
	Reference        string   `json:"reference"`
	Severity         string   `json:"severity"`
	Status           string   `json:"status"`
	AffectedVersions []string `json:"affected_versions"`
	PatchedInVersion string   `json:"patched_in_version,omitempty"`
	Summary          string   `json:"summary"`
	DetailsMarkdown  string   `json:"details_markdown,omitempty"`
	PublishedAt      string   `json:"published_at,omitempty"`
	PatchedAt        string   `json:"patched_at,omitempty"`
	ClosedAt         string   `json:"closed_at,omitempty"`
	SlaDeadline      string   `json:"sla_deadline,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

// AdvisoryListResponse wraps advisory records in the list contract.
type AdvisoryListResponse struct {
	Data []*AdvisoryResponse `json:"data"`
}

func NewAdvisoryResponse(advisory *secmodel.Advisory) *AdvisoryResponse {
	if advisory == nil {
		return nil
	}
	resp := &AdvisoryResponse{
		ID:               advisory.ID,
		Reference:        advisory.Reference,
		Severity:         advisory.Severity,
		Status:           advisory.Status,
		AffectedVersions: advisory.AffectedVersionList(),
		PatchedInVersion: advisory.PatchedInVersion,
		Summary:          advisory.Summary,
		DetailsMarkdown:  advisory.DetailsMarkdown,
		CreatedAt:        advisory.CreatedAt.UTC().Format(time.RFC3339),
	}
	if advisory.PublishedAt != nil {
		resp.PublishedAt = advisory.PublishedAt.UTC().Format(time.RFC3339)
	}
	if advisory.PatchedAt != nil {
		resp.PatchedAt = advisory.PatchedAt.UTC().Format(time.RFC3339)
	}
	if advisory.ClosedAt != nil {
		resp.ClosedAt = advisory.ClosedAt.UTC().Format(time.RFC3339)
	}
	if advisory.SlaDeadline != nil {
		resp.SlaDeadline = advisory.SlaDeadline.UTC().Format(time.RFC3339)
	}
	return resp
}

func NewAdvisoryListResponse(items []*secmodel.Advisory) *AdvisoryListResponse {
	out := make([]*AdvisoryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, NewAdvisoryResponse(item))
	}
	return &AdvisoryListResponse{Data: out}
}
