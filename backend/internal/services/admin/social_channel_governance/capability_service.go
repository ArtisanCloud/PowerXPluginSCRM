package social_channel_governance

import (
	"context"
	"strings"
)

type CapabilityService struct {
	factory *ChannelFactory
}

func NewCapabilityService(factory *ChannelFactory) *CapabilityService {
	return &CapabilityService{factory: factory}
}

func (s *CapabilityService) Matrix(ctx context.Context, tenantUUID, channel, appType string) map[string]string {
	if s == nil {
		return map[string]string{}
	}
	if s.factory != nil {
		if adapter, ok := s.factory.Resolve(channel, appType); ok && adapter != nil {
			return adapter.CapabilityMatrix(ctx, tenantUUID)
		}
	}
	// Default degraded matrix when adapter is absent.
	return map[string]string{
		"auth":              "not_supported",
		"tags":              "not_supported",
		"org":               "not_supported",
		"external_contacts": "not_supported",
		"leads":             "not_supported",
	}
}

func (s *CapabilityService) StatusForDomain(matrix map[string]string, domain string) string {
	state := strings.TrimSpace(matrix[strings.TrimSpace(domain)])
	if state == "" {
		return "not_supported"
	}
	return state
}
