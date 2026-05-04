package acquisition

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupTagRuleService_Match(t *testing.T) {
	svc := NewGroupTagRuleService()
	rule := GroupTagRule{
		RuleUUID:             "rule-001",
		GroupTagUUID:         "tag-001",
		SourceGroupCodeUUID:  "code-001",
		OwnerUserID:          "owner-a",
		MinMemberTaggedRatio: 0.30,
		Enabled:              true,
	}

	matched := svc.Match(rule, GroupTagRuleInput{
		ChatID:              "chat-001",
		SourceGroupCodeUUID: "code-001",
		OwnerUserID:         "owner-a",
		MemberTaggedRatio:   0.45,
	})
	require.True(t, matched)

	ownerMismatch := svc.Match(rule, GroupTagRuleInput{
		ChatID:              "chat-002",
		SourceGroupCodeUUID: "code-001",
		OwnerUserID:         "owner-b",
		MemberTaggedRatio:   0.45,
	})
	require.False(t, ownerMismatch)

	ratioMismatch := svc.Match(rule, GroupTagRuleInput{
		ChatID:              "chat-003",
		SourceGroupCodeUUID: "code-001",
		OwnerUserID:         "owner-a",
		MemberTaggedRatio:   0.20,
	})
	require.False(t, ratioMismatch)
}

func TestGroupTagRuleService_BindIdempotent(t *testing.T) {
	svc := NewGroupTagRuleService()

	set, changed := svc.BindIdempotent(nil, "chat-001")
	require.True(t, changed)
	require.Len(t, set, 1)

	set, changed = svc.BindIdempotent(set, "chat-001")
	require.False(t, changed)
	require.Len(t, set, 1)

	set, changed = svc.BindIdempotent(set, "chat-002")
	require.True(t, changed)
	require.Len(t, set, 2)
}
