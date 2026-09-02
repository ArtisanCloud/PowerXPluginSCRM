package lead_capture

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLeadTimeline_UnifiesEventsActorsFiltersAndPagination(t *testing.T) {
	db := openLeadAttachmentServiceDB(t, "lead_timeline_service")
	for _, statement := range []string{
		`CREATE TABLE lead_capture_assignments (assignment_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, lead_uuid TEXT NOT NULL, owner_user_uuid TEXT NOT NULL, reason TEXT, created_at DATETIME);`,
		`CREATE TABLE lead_capture_status_history (history_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, lead_uuid TEXT NOT NULL, from_status TEXT NOT NULL, to_status TEXT NOT NULL, changed_at DATETIME);`,
		`CREATE TABLE lead_capture_sources (source_uuid TEXT PRIMARY KEY, lead_uuid TEXT NOT NULL, tenant_uuid TEXT NOT NULL, channel_code TEXT, app_type TEXT, account_uuid TEXT, campaign_code TEXT, utm_source TEXT, utm_medium TEXT, utm_campaign TEXT, created_at DATETIME, updated_at DATETIME);`,
		`CREATE TABLE iam_members (id INTEGER PRIMARY KEY, tenant_uuid TEXT NOT NULL, user_id INTEGER NOT NULL, username TEXT NOT NULL, display_name TEXT, avatar_url TEXT, status TEXT NOT NULL, department_id INTEGER, meta TEXT, last_login_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME);`,
	} {
		require.NoError(t, db.Exec(statement).Error)
	}
	service := NewLeadService(leadrepo.NewLeadRepository(db))
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	otherTenantUUID := "00000000-0000-4000-8000-000000000002"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	operatorMemberUUID := "30000000-0000-4000-8000-000000000001"
	requestContext := authx.ContextWithMemberUUID(context.Background(), operatorMemberUUID)

	require.NoError(t, db.Create(&leadmodel.Lead{LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusCaptured}).Error)
	require.NoError(t, db.Create(&iamm.Member{BaseModel: basemodels.BaseModel{ID: 7, TenantUuid: tenantUUID}, UserID: 9, Username: "collector", Status: iamm.StatusActive}).Error)

	_, err := service.Assign(requestContext, tenantUUID, leadUUID, LeadAssignRequest{OwnerUserUUID: "7", Reason: "首次分配"})
	require.NoError(t, err)
	_, err = service.RecordActivity(requestContext, tenantUUID, leadUUID, LeadActivityCreateRequest{Method: "wechat", Content: "确认需求", StageKey: leadmodel.LeadStatusRouted, ActionKey: "engagement"})
	require.NoError(t, err)
	require.NoError(t, db.Create(&leadmodel.LeadSource{SourceUUID: uuid.NewString(), TenantUUID: tenantUUID, LeadUUID: leadUUID, ChannelCode: "wechat", AppType: "wecom"}).Error)

	page, err := service.ListTimeline(context.Background(), tenantUUID, leadUUID, LeadTimelineQuery{StageKey: leadmodel.LeadStatusRouted, Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Items, 2)

	assigned, err := service.ListTimeline(context.Background(), tenantUUID, leadUUID, LeadTimelineQuery{EventType: LeadTimelineEventAssigned, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, assigned.Items, 1)
	require.Equal(t, leadmodel.LeadAuditActorTypeMember, assigned.Items[0].ActorType)
	require.Equal(t, operatorMemberUUID, assigned.Items[0].ActorMemberUUID)

	statusEvents, err := service.ListTimeline(context.Background(), tenantUUID, leadUUID, LeadTimelineQuery{EventType: LeadTimelineEventStatusChanged, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, statusEvents.Items, 1)
	require.Equal(t, operatorMemberUUID, statusEvents.Items[0].ActorMemberUUID)

	legacyHistory := &leadmodel.LeadStatusHistory{HistoryUUID: uuid.NewString(), TenantUUID: tenantUUID, LeadUUID: leadUUID, FromStatus: leadmodel.LeadStatusRouted, ToStatus: leadmodel.LeadStatusEngaging, ChangedAt: time.Now().UTC().Add(time.Second)}
	require.NoError(t, db.Create(legacyHistory).Error)
	statusEvents, err = service.ListTimeline(context.Background(), tenantUUID, leadUUID, LeadTimelineQuery{EventType: LeadTimelineEventStatusChanged, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, leadmodel.LeadAuditActorTypeSystem, statusEvents.Items[0].ActorType)
	require.Empty(t, statusEvents.Items[0].ActorMemberUUID)

	_, err = service.ListTimeline(context.Background(), otherTenantUUID, leadUUID, LeadTimelineQuery{})
	require.ErrorIs(t, err, leadrepo.ErrLeadNotFound)

	_, err = service.ListTimeline(context.Background(), tenantUUID, leadUUID, LeadTimelineQuery{EventType: "unknown_event"})
	require.ErrorIs(t, err, ErrInvalidLeadTimelineQuery)

	rollbackLeadUUID := "10000000-0000-4000-8000-000000000002"
	require.NoError(t, db.Create(&leadmodel.Lead{LeadUUID: rollbackLeadUUID, TenantUUID: tenantUUID, DisplayName: "rollback", Status: leadmodel.LeadStatusCaptured}).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_assignment_audit
		BEFORE INSERT ON lead_capture_activities
		WHEN NEW.activity_type = 'assign'
		BEGIN SELECT RAISE(ABORT, 'assignment audit rejected'); END;`).Error)
	_, err = service.Assign(requestContext, tenantUUID, rollbackLeadUUID, LeadAssignRequest{OwnerUserUUID: "7"})
	require.Error(t, err)
	var assignmentCount int64
	require.NoError(t, db.Model(&leadmodel.LeadAssignment{}).Where("lead_uuid = ?", rollbackLeadUUID).Count(&assignmentCount).Error)
	require.Zero(t, assignmentCount)
	rollbackLead, err := service.Get(context.Background(), tenantUUID, rollbackLeadUUID)
	require.NoError(t, err)
	require.Equal(t, leadmodel.LeadStatusCaptured, rollbackLead.Status)
}
