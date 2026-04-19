package lead_capture

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

// LeadDedupService implements the canonical dedup key:
// external_userid + phone + corp + channel.
type LeadDedupService struct{}

func NewLeadDedupService() *LeadDedupService {
	return &LeadDedupService{}
}

func (s *LeadDedupService) BuildExternalContactKey(externalUserID, phone, corpID, channel string) string {
	raw := strings.ToLower(strings.TrimSpace(externalUserID)) + "|" +
		strings.TrimSpace(phone) + "|" +
		strings.ToLower(strings.TrimSpace(corpID)) + "|" +
		strings.ToLower(strings.TrimSpace(channel))
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}
