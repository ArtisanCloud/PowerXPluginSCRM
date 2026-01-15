package security

import (
	"time"

	privmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/privacy"
)

type ConsentTokenResponse struct {
	ID            string   `json:"id"`
	TenantUuid    string   `json:"tenant_uuid"`
	Token         string   `json:"token"`
	Scope         []string `json:"scope"`
	Status        string   `json:"status"`
	ExpiresAt     string   `json:"expires_at,omitempty"`
	IssuedAt      string   `json:"issued_at"`
	IssuedBy      string   `json:"issued_by"`
	RevokedAt     string   `json:"revoked_at,omitempty"`
	RevokedReason string   `json:"revoked_reason,omitempty"`
}

func NewConsentTokenResponse(token *privmodel.ConsentToken) *ConsentTokenResponse {
	if token == nil {
		return nil
	}
	scope, _ := token.ScopeValues()
	resp := &ConsentTokenResponse{
		ID:         token.ID,
		TenantUuid: token.TenantUuid,
		Token:      token.Token,
		Scope:      scope,
		Status:     token.Status,
		IssuedAt:   token.IssuedAt.UTC().Format(time.RFC3339),
		IssuedBy:   token.IssuedBy,
	}
	if token.ExpiresAt != nil {
		resp.ExpiresAt = token.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if token.RevokedAt != nil {
		resp.RevokedAt = token.RevokedAt.UTC().Format(time.RFC3339)
	}
	if token.RevokedReason != "" {
		resp.RevokedReason = token.RevokedReason
	}
	return resp
}

type ConsentTokenListResponse struct {
	Data []*ConsentTokenResponse `json:"data"`
}

func NewConsentTokenListResponse(tokens []*privmodel.ConsentToken) *ConsentTokenListResponse {
	out := make([]*ConsentTokenResponse, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, NewConsentTokenResponse(t))
	}
	return &ConsentTokenListResponse{Data: out}
}

type RevokeConsentRequest struct {
	Reason      string `json:"reason"`
	RequestedBy string `json:"requested_by"`
}

type LifecycleEventResponse struct {
	ID         string      `json:"id"`
	TenantUuid string      `json:"tenant_uuid"`
	EventType  string      `json:"event_type"`
	AssetKey   string      `json:"asset_key"`
	Status     string      `json:"status"`
	OccurredAt string      `json:"occurred_at"`
	RecordedBy string      `json:"recorded_by"`
	Payload    interface{} `json:"payload,omitempty"`
}

func NewLifecycleEventResponse(evt *privmodel.LifecycleEvent) *LifecycleEventResponse {
	if evt == nil {
		return nil
	}
	resp := &LifecycleEventResponse{
		ID:         evt.ID,
		TenantUuid: evt.TenantUuid,
		EventType:  evt.EventType,
		AssetKey:   evt.AssetKey,
		Status:     evt.Status,
		OccurredAt: evt.OccurredAt.UTC().Format(time.RFC3339),
		RecordedBy: evt.RecordedBy,
	}
	if len(evt.Payload) > 0 {
		resp.Payload = evt.Payload
	}
	return resp
}

type LifecycleEventListResponse struct {
	Data []*LifecycleEventResponse `json:"data"`
}

func NewLifecycleEventListResponse(events []*privmodel.LifecycleEvent) *LifecycleEventListResponse {
	out := make([]*LifecycleEventResponse, 0, len(events))
	for _, evt := range events {
		out = append(out, NewLifecycleEventResponse(evt))
	}
	return &LifecycleEventListResponse{Data: out}
}
