package acquisition

import "strings"

// GroupTagRule defines a minimal rule payload used by V2.1 tests.
type GroupTagRule struct {
	RuleUUID             string
	GroupTagUUID         string
	SourceGroupCodeUUID  string
	OwnerUserID          string
	MinMemberTaggedRatio float64
	Enabled              bool
}

// GroupTagRuleInput is the materialized group snapshot for rule evaluation.
type GroupTagRuleInput struct {
	ChatID              string
	SourceGroupCodeUUID string
	OwnerUserID         string
	MemberTaggedRatio   float64
}

// GroupTagRuleService provides deterministic matching and idempotent binding helpers.
type GroupTagRuleService struct{}

func NewGroupTagRuleService() *GroupTagRuleService {
	return &GroupTagRuleService{}
}

func (s *GroupTagRuleService) Match(rule GroupTagRule, input GroupTagRuleInput) bool {
	if !rule.Enabled {
		return false
	}
	if strings.TrimSpace(input.ChatID) == "" {
		return false
	}
	if rule.SourceGroupCodeUUID != "" && rule.SourceGroupCodeUUID != input.SourceGroupCodeUUID {
		return false
	}
	if rule.OwnerUserID != "" && rule.OwnerUserID != input.OwnerUserID {
		return false
	}
	if rule.MinMemberTaggedRatio > 0 && input.MemberTaggedRatio < rule.MinMemberTaggedRatio {
		return false
	}
	return true
}

// BindIdempotent adds chatID into an existing set and returns whether the set changed.
func (s *GroupTagRuleService) BindIdempotent(existing map[string]struct{}, chatID string) (map[string]struct{}, bool) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		if existing == nil {
			existing = map[string]struct{}{}
		}
		return existing, false
	}
	if existing == nil {
		existing = map[string]struct{}{}
	}
	if _, ok := existing[chatID]; ok {
		return existing, false
	}
	existing[chatID] = struct{}{}
	return existing, true
}
