package social_channel_governance

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

type IdempotencyService struct{}

func NewIdempotencyService() *IdempotencyService {
	return &IdempotencyService{}
}

func (s *IdempotencyService) BuildKey(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, p := range parts {
		normalized = append(normalized, strings.TrimSpace(strings.ToLower(p)))
	}
	sum := sha1.Sum([]byte(strings.Join(normalized, "|")))
	return hex.EncodeToString(sum[:])
}
