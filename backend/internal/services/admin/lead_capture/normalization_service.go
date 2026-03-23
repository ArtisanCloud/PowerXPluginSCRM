package lead_capture

import "strings"

// NormalizedLeadInput is a normalized view of lead intake payload.
type NormalizedLeadInput struct {
	DisplayName       string
	Phone             string
	Email             string
	SourceChannel     string
	SourceAppType     string
	SourceAccountUUID string
	OwnerUserUUID     string
}

// NormalizationService centralizes lead-field normalization for multi-entry reuse.
type NormalizationService struct{}

func NewNormalizationService() *NormalizationService {
	return &NormalizationService{}
}

func (s *NormalizationService) NormalizeLeadCreateRequest(req LeadCreateRequest) NormalizedLeadInput {
	_ = s
	out := NormalizedLeadInput{
		DisplayName:       strings.TrimSpace(req.DisplayName),
		Phone:             strings.TrimSpace(req.Phone),
		Email:             strings.ToLower(strings.TrimSpace(req.Email)),
		SourceChannel:     strings.ToLower(strings.TrimSpace(req.SourceChannel)),
		SourceAppType:     strings.ToLower(strings.TrimSpace(req.SourceAppType)),
		SourceAccountUUID: strings.ToLower(strings.TrimSpace(req.SourceAccountUUID)),
		OwnerUserUUID:     strings.TrimSpace(req.OwnerUserUUID),
	}
	if out.SourceChannel == "" {
		out.SourceChannel = "wechat"
	}
	if out.SourceAppType == "" {
		out.SourceAppType = "wecom"
	}
	return out
}
