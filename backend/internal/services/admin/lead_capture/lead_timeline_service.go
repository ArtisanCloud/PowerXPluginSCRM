package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
)

var ErrInvalidLeadTimelineQuery = errors.New("invalid lead timeline query")

const (
	LeadTimelineEventStatusChanged      = "status_changed"
	LeadTimelineEventAssigned           = "assigned"
	LeadTimelineEventSourceCaptured     = "source_captured"
	LeadTimelineEventManualActivity     = "manual_activity"
	LeadTimelineEventAttachmentUploaded = "attachment_uploaded"
	LeadTimelineEventAttachmentDeleted  = "attachment_deleted"
)

type LeadTimelineQuery struct {
	StageKey  string
	EventType string
	Page      int
	PageSize  int
}

type LeadTimelineEvent struct {
	EventUUID       string         `json:"event_uuid"`
	EventType       string         `json:"event_type"`
	ActorType       string         `json:"actor_type"`
	ActorMemberUUID string         `json:"actor_member_uuid,omitempty"`
	StageKey        string         `json:"stage_key,omitempty"`
	ActionKey       string         `json:"action_key,omitempty"`
	OccurredAt      time.Time      `json:"occurred_at"`
	Data            map[string]any `json:"data"`
}

type LeadTimelinePage struct {
	Items    []LeadTimelineEvent `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

type leadTimelineActor struct {
	actorType  string
	memberUUID string
}

func (s *LeadService) ListTimeline(ctx context.Context, tenantUUID, leadUUID string, query LeadTimelineQuery) (*LeadTimelinePage, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("lead repository not configured")
	}
	snapshot, err := s.repo.ListTimelineSnapshot(ctx, tenantUUID, leadUUID)
	if err != nil {
		return nil, err
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	stageKey := normalizeLeadStatus(query.StageKey)
	eventType := strings.ToLower(strings.TrimSpace(query.EventType))
	if strings.TrimSpace(query.StageKey) != "" && !isValidLeadStatus(stageKey) {
		return nil, ErrInvalidLeadTimelineQuery
	}
	if eventType != "" && !isLeadTimelineEventType(eventType) {
		return nil, ErrInvalidLeadTimelineQuery
	}

	actors := make(map[string]leadTimelineActor)
	for _, activity := range snapshot.Activities {
		if activity == nil {
			continue
		}
		actor := timelineActor(activity.Payload)
		for _, key := range []string{"assignment_uuid", "history_uuid"} {
			if subjectUUID := timelinePayloadText(activity.Payload, key); subjectUUID != "" {
				actors[strings.ToLower(subjectUUID)] = actor
			}
		}
	}

	items := make([]LeadTimelineEvent, 0, len(snapshot.Activities)+len(snapshot.Assignments)+len(snapshot.StatusHistory)+len(snapshot.Sources))
	for _, activity := range snapshot.Activities {
		if activity == nil {
			continue
		}
		typeKey := strings.TrimSpace(activity.ActivityType)
		if typeKey == model.LeadActivityTypeAssign || typeKey == model.LeadActivityTypeStatusChange {
			continue
		}
		if typeKey != model.LeadActivityTypeManual && typeKey != model.LeadActivityTypeAttachmentUploaded && typeKey != model.LeadActivityTypeAttachmentDeleted {
			continue
		}
		actor := timelineActor(activity.Payload)
		data := timelinePayloadCopy(activity.Payload)
		data["activity_uuid"] = activity.ActivityUUID
		data["activity_type"] = typeKey
		items = append(items, LeadTimelineEvent{
			EventUUID: activity.ActivityUUID, EventType: typeKey, ActorType: actor.actorType,
			ActorMemberUUID: actor.memberUUID, StageKey: timelinePayloadText(activity.Payload, "stage_key"),
			ActionKey: timelinePayloadText(activity.Payload, "action_key"), OccurredAt: activity.CreatedAt, Data: data,
		})
	}
	for _, assignment := range snapshot.Assignments {
		if assignment == nil {
			continue
		}
		actor := actors[strings.ToLower(assignment.AssignmentUUID)]
		actor = normalizedTimelineActor(actor)
		data := map[string]any{"assignment_uuid": assignment.AssignmentUUID, "owner_user_uuid": assignment.OwnerUserUUID, "reason": assignment.Reason}
		stage := model.LeadStatusRouted
		if audit := timelineAuditActivity(snapshot.Activities, "assignment_uuid", assignment.AssignmentUUID); audit != nil {
			stage = timelinePayloadText(audit.Payload, "stage_key")
		}
		items = append(items, LeadTimelineEvent{EventUUID: assignment.AssignmentUUID, EventType: LeadTimelineEventAssigned, ActorType: actor.actorType, ActorMemberUUID: actor.memberUUID, StageKey: stage, ActionKey: "assign", OccurredAt: assignment.CreatedAt, Data: data})
	}
	for _, history := range snapshot.StatusHistory {
		if history == nil {
			continue
		}
		actor := normalizedTimelineActor(actors[strings.ToLower(history.HistoryUUID)])
		items = append(items, LeadTimelineEvent{
			EventUUID: history.HistoryUUID, EventType: LeadTimelineEventStatusChanged, ActorType: actor.actorType,
			ActorMemberUUID: actor.memberUUID, StageKey: history.ToStatus, ActionKey: "status", OccurredAt: history.ChangedAt,
			Data: map[string]any{"history_uuid": history.HistoryUUID, "from_status": history.FromStatus, "to_status": history.ToStatus},
		})
	}
	for _, source := range snapshot.Sources {
		if source == nil {
			continue
		}
		data := map[string]any{"source_uuid": source.SourceUUID, "channel_code": source.ChannelCode, "app_type": source.AppType, "campaign_code": source.CampaignCode, "utm_source": source.UTMSource, "utm_medium": source.UTMMedium, "utm_campaign": source.UTMCampaign}
		if source.AccountUUID != nil {
			data["account_uuid"] = *source.AccountUUID
		}
		items = append(items, LeadTimelineEvent{EventUUID: source.SourceUUID, EventType: LeadTimelineEventSourceCaptured, ActorType: model.LeadAuditActorTypeSystem, StageKey: model.LeadStatusCaptured, ActionKey: "source", OccurredAt: source.CreatedAt, Data: data})
	}

	filtered := items[:0]
	for _, item := range items {
		if eventType != "" && item.EventType != eventType {
			continue
		}
		if stageKey != "" && !timelineEventMatchesStage(item, stageKey) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].OccurredAt.After(filtered[j].OccurredAt) })
	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return &LeadTimelinePage{Items: filtered[start:end], Total: total, Page: page, PageSize: pageSize}, nil
}

func timelineActor(payload map[string]any) leadTimelineActor {
	memberUUID := strings.ToLower(timelinePayloadText(payload, "operator_member_uuid"))
	actorType := strings.ToLower(timelinePayloadText(payload, "actor_type"))
	if memberUUID != "" {
		return leadTimelineActor{actorType: model.LeadAuditActorTypeMember, memberUUID: memberUUID}
	}
	if actorType != model.LeadAuditActorTypeMember {
		actorType = model.LeadAuditActorTypeSystem
	}
	return leadTimelineActor{actorType: actorType}
}

func normalizedTimelineActor(actor leadTimelineActor) leadTimelineActor {
	if actor.memberUUID != "" {
		actor.actorType = model.LeadAuditActorTypeMember
		return actor
	}
	actor.actorType = model.LeadAuditActorTypeSystem
	return actor
}

func timelinePayloadText(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func isLeadTimelineEventType(value string) bool {
	switch value {
	case LeadTimelineEventStatusChanged, LeadTimelineEventAssigned, LeadTimelineEventSourceCaptured,
		LeadTimelineEventManualActivity, LeadTimelineEventAttachmentUploaded, LeadTimelineEventAttachmentDeleted:
		return true
	default:
		return false
	}
}

func timelinePayloadCopy(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload)+2)
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func timelineAuditActivity(activities []*model.LeadActivity, key, value string) *model.LeadActivity {
	for _, activity := range activities {
		if activity != nil && strings.EqualFold(timelinePayloadText(activity.Payload, key), value) {
			return activity
		}
	}
	return nil
}

func timelineEventMatchesStage(event LeadTimelineEvent, stageKey string) bool {
	if strings.EqualFold(event.StageKey, stageKey) {
		return true
	}
	if event.EventType == LeadTimelineEventStatusChanged {
		return strings.EqualFold(fmt.Sprint(event.Data["from_status"]), stageKey) || strings.EqualFold(fmt.Sprint(event.Data["to_status"]), stageKey)
	}
	return false
}
