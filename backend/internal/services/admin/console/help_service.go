package console

import (
	"context"
)

// HelpService serves contextual runbook guidance for the console.
type HelpService struct {
	entries []GuidanceItem
}

// NewHelpService constructs a help service with default entries.
func NewHelpService() *HelpService {
	items := make([]GuidanceItem, len(defaultTroubleshootingHelp))
	copy(items, defaultTroubleshootingHelp)
	return &HelpService{entries: items}
}

// TroubleshootingGuidance returns guidance items for troubleshooting sections.
func (s *HelpService) FetchGuidance(_ context.Context, _ *string) ([]GuidanceItem, error) {
	items := make([]GuidanceItem, len(s.entries))
	copy(items, s.entries)
	return items, nil
}
