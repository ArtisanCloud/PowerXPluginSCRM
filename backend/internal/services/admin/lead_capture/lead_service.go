package lead_capture

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	pwtransferreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/transfer/request"
	iamentity "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	wecomauth "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/wecomauth"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrInvalidLeadPayload = errors.New("invalid lead payload")
var ErrInvalidAssignee = errors.New("invalid assignee")
var ErrAssigneeNotFound = errors.New("assignee not found")
var ErrAssigneeTransferFailed = errors.New("assignee transfer failed")
var ErrInvalidLeadStatus = errors.New("invalid lead status")
var ErrInvalidLeadStatusTransition = errors.New("invalid lead status transition")

// LeadService orchestrates lead management operations.
type LeadService struct {
	repo             *leadrepo.LeadRepository
	sourceEventRepo  *leadrepo.LeadSourceEventRepository
	normalizationSvc *NormalizationService
	dedupSvc         *DedupService
	assignmentSvc    *AssignmentService
}

func NewLeadService(repo *leadrepo.LeadRepository) *LeadService {
	svc := &LeadService{
		repo:             repo,
		normalizationSvc: NewNormalizationService(),
		dedupSvc:         NewDedupService(),
		assignmentSvc:    NewAssignmentService(),
	}
	if repo != nil && repo.DB != nil {
		svc.sourceEventRepo = leadrepo.NewLeadSourceEventRepository(repo.DB)
	}
	return svc
}

type LeadCreateRequest struct {
	DisplayName       string
	Phone             string
	Email             string
	SourceChannel     string
	SourceAppType     string
	SourceAccountUUID string
	OwnerUserUUID     string
}

type LeadAssignRequest struct {
	OwnerUserUUID string
	Reason        string
}

type LeadBatchAssignRequest struct {
	LeadUUIDs     []string
	OwnerUserUUID string
	Reason        string
}

type LeadBatchAssignItem struct {
	LeadUUID     string `json:"lead_uuid"`
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type LeadBatchAssignResult struct {
	Total        int                   `json:"total"`
	SuccessCount int                   `json:"success_count"`
	FailedCount  int                   `json:"failed_count"`
	Items        []LeadBatchAssignItem `json:"items"`
}

type LeadUpdateRequest struct {
	DisplayName       string
	Phone             string
	Email             string
	SourceChannel     string
	SourceAppType     string
	SourceAccountUUID string
}

type LeadStatusUpdateRequest struct {
	Status string
}

func (s *LeadService) Create(ctx context.Context, tenantUUID string, req LeadCreateRequest) (*model.Lead, error) {
	return s.createWithOptions(ctx, tenantUUID, req, leadCreateOptions{})
}

type leadCreateOptions struct {
	preserveSourceScope bool
}

func (s *LeadService) createWithOptions(ctx context.Context, tenantUUID string, req LeadCreateRequest, opts leadCreateOptions) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	normalized := req
	if s.normalizationSvc != nil {
		n := s.normalizationSvc.NormalizeLeadCreateRequest(req)
		normalized = LeadCreateRequest{
			DisplayName:       n.DisplayName,
			Phone:             n.Phone,
			Email:             n.Email,
			SourceChannel:     n.SourceChannel,
			SourceAppType:     n.SourceAppType,
			SourceAccountUUID: n.SourceAccountUUID,
			OwnerUserUUID:     n.OwnerUserUUID,
		}
	}
	if opts.preserveSourceScope {
		normalized.SourceChannel = strings.ToLower(strings.TrimSpace(req.SourceChannel))
		normalized.SourceAppType = strings.ToLower(strings.TrimSpace(req.SourceAppType))
		normalized.SourceAccountUUID = strings.ToLower(strings.TrimSpace(req.SourceAccountUUID))
	}

	if strings.TrimSpace(normalized.DisplayName) == "" &&
		strings.TrimSpace(normalized.Phone) == "" &&
		strings.TrimSpace(normalized.Email) == "" {
		return nil, ErrInvalidLeadPayload
	}
	status := model.LeadStatusNew
	sourceAccountUUID := strings.TrimSpace(normalized.SourceAccountUUID)
	var sourceAccountPtr *string
	if sourceAccountUUID != "" {
		sourceAccountPtr = &sourceAccountUUID
	}
	var result *model.Lead
	var mergeMeta map[string]any
	var merged bool
	err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		existing, matchOn, err := s.findExistingLead(
			ctx,
			tx,
			tenantUUID,
			normalized.Phone,
			normalized.Email,
			normalized.SourceChannel,
			normalized.SourceAppType,
			sourceAccountUUID,
		)
		if err != nil {
			return err
		}
		if existing != nil {
			updatedFields := map[string]interface{}{}
			mergedFields := []string{}
			if s.dedupSvc != nil {
				updatedFields, mergedFields = s.dedupSvc.BuildMergeUpdates(existing, NormalizedLeadInput{
					DisplayName:       normalized.DisplayName,
					Phone:             normalized.Phone,
					Email:             normalized.Email,
					SourceChannel:     normalized.SourceChannel,
					SourceAppType:     normalized.SourceAppType,
					SourceAccountUUID: sourceAccountUUID,
					OwnerUserUUID:     normalized.OwnerUserUUID,
				})
				if v, ok := updatedFields["display_name"].(string); ok {
					existing.DisplayName = v
				}
				if v, ok := updatedFields["phone"].(string); ok {
					existing.Phone = v
				}
				if v, ok := updatedFields["email"].(string); ok {
					existing.Email = v
				}
				if v, ok := updatedFields["source_channel"].(string); ok {
					existing.SourceChannel = v
				}
				if v, ok := updatedFields["source_app_type"].(string); ok {
					existing.SourceAppType = v
				}
				if _, ok := updatedFields["source_account_uuid"]; ok {
					existing.SourceAccountUUID = sourceAccountPtr
				}
				if v, ok := updatedFields["owner_user_uuid"].(string); ok {
					existing.OwnerUserUUID = v
				}
			}
			if len(updatedFields) > 0 {
				existing.UpdatedAt = time.Now().UTC()
				updatedFields["updated_at"] = existing.UpdatedAt
				if err := tx.Model(&model.Lead{}).
					Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, existing.LeadUUID).
					Updates(updatedFields).Error; err != nil {
					return err
				}
			}
			mergeMeta = map[string]any{
				"match_on":      matchOn,
				"merged_fields": mergedFields,
				"incoming":      buildIncomingPayload(normalized.DisplayName, normalized.Phone, normalized.Email, normalized.SourceChannel, normalized.SourceAppType, sourceAccountUUID, normalized.OwnerUserUUID),
			}
			if err := createLeadActivity(ctx, tx, tenantUUID, existing.LeadUUID, model.LeadActivityTypeMerge, datatypes.JSONMap(mergeMeta)); err != nil {
				return err
			}
			if err := s.createSourceTrace(ctx, tx, tenantUUID, existing.LeadUUID, normalized); err != nil {
				return err
			}
			existing.HasMerge = true
			merged = true
			result = existing
			return nil
		}
		lead := &model.Lead{
			LeadUUID:          uuid.NewString(),
			TenantUUID:        tenantUUID,
			DisplayName:       normalized.DisplayName,
			Phone:             normalized.Phone,
			Email:             normalized.Email,
			Status:            status,
			OwnerUserUUID:     strings.TrimSpace(normalized.OwnerUserUUID),
			SourceChannel:     strings.TrimSpace(normalized.SourceChannel),
			SourceAppType:     strings.TrimSpace(normalized.SourceAppType),
			SourceAccountUUID: sourceAccountPtr,
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		}
		if err := tx.Create(lead).Error; err != nil {
			return err
		}
		intakeMeta := map[string]any{
			"source_channel":      lead.SourceChannel,
			"source_app_type":     lead.SourceAppType,
			"source_account_uuid": sourceAccountUUID,
		}
		if err := createLeadActivity(ctx, tx, tenantUUID, lead.LeadUUID, model.LeadActivityTypeIntake, datatypes.JSONMap(intakeMeta)); err != nil {
			return err
		}
		if err := s.createSourceTrace(ctx, tx, tenantUUID, lead.LeadUUID, normalized); err != nil {
			return err
		}
		lead.HasMerge = false
		result = lead
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result != nil && !merged {
		if err := s.attachMergeFlag(ctx, tenantUUID, result); err != nil {
			return nil, err
		}
	}
	actor := leadobs.ResolveActorUserUUID(ctx, "")
	if merged {
		leadobs.EmitLeadMerged(ctx, tenantUUID, result.LeadUUID, actor, mergeMeta)
	} else if result != nil {
		leadobs.EmitLeadCreated(ctx, tenantUUID, result.LeadUUID, actor, map[string]any{
			"source_channel":  result.SourceChannel,
			"source_app_type": result.SourceAppType,
		})
	}
	return result, nil
}

func (s *LeadService) List(ctx context.Context, tenantUUID string) ([]*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	start := time.Now()
	items, err := s.repo.List(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	if err := s.attachMergeFlags(ctx, tenantUUID, items); err != nil {
		return nil, err
	}
	if err := s.attachChannelSyncState(ctx, tenantUUID, items); err != nil {
		return nil, err
	}
	elapsed := time.Since(start)
	if elapsed > 300*time.Millisecond {
		logger.WithFields(logrus.Fields{
			"component":   "lead_capture_service",
			"tenant_uuid": tenantUUID,
			"duration_ms": elapsed.Milliseconds(),
			"count":       len(items),
		}).Warn("lead list latency exceeded 300ms")
	}
	return items, nil
}

func (s *LeadService) Get(ctx context.Context, tenantUUID, leadUUID string) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	item, err := s.repo.GetByUUID(ctx, tenantUUID, leadUUID)
	if err != nil {
		return nil, err
	}
	if err := s.attachMergeFlag(ctx, tenantUUID, item); err != nil {
		return nil, err
	}
	if err := s.attachChannelSyncState(ctx, tenantUUID, []*model.Lead{item}); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *LeadService) Update(ctx context.Context, tenantUUID, leadUUID string, req LeadUpdateRequest) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	normalized := LeadCreateRequest{
		DisplayName:       strings.TrimSpace(req.DisplayName),
		Phone:             strings.TrimSpace(req.Phone),
		Email:             strings.ToLower(strings.TrimSpace(req.Email)),
		SourceChannel:     strings.ToLower(strings.TrimSpace(req.SourceChannel)),
		SourceAppType:     strings.ToLower(strings.TrimSpace(req.SourceAppType)),
		SourceAccountUUID: strings.ToLower(strings.TrimSpace(req.SourceAccountUUID)),
	}
	if normalized.DisplayName == "" &&
		normalized.Phone == "" &&
		normalized.Email == "" &&
		normalized.SourceChannel == "" &&
		normalized.SourceAppType == "" &&
		normalized.SourceAccountUUID == "" {
		return nil, ErrInvalidLeadPayload
	}

	var updated *model.Lead
	err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		lead, err := getLeadByUUIDTx(ctx, tx, tenantUUID, leadUUID)
		if err != nil {
			return err
		}
		lead.DisplayName = normalized.DisplayName
		lead.Phone = normalized.Phone
		lead.Email = normalized.Email
		lead.UpdatedAt = time.Now().UTC()
		updates := map[string]interface{}{
			"display_name": lead.DisplayName,
			"phone":        lead.Phone,
			"email":        lead.Email,
			"updated_at":   lead.UpdatedAt,
		}
		activityPayload := datatypes.JSONMap{
			"display_name": lead.DisplayName,
			"phone":        lead.Phone,
			"email":        lead.Email,
		}
		if normalized.SourceChannel != "" {
			lead.SourceChannel = normalized.SourceChannel
			updates["source_channel"] = lead.SourceChannel
			activityPayload["source_channel"] = lead.SourceChannel
		}
		if normalized.SourceAppType != "" {
			lead.SourceAppType = normalized.SourceAppType
			updates["source_app_type"] = lead.SourceAppType
			activityPayload["source_app_type"] = lead.SourceAppType
		}
		if normalized.SourceAccountUUID != "" {
			sourceAccountUUID := normalized.SourceAccountUUID
			lead.SourceAccountUUID = &sourceAccountUUID
			updates["source_account_uuid"] = lead.SourceAccountUUID
			activityPayload["source_account_uuid"] = sourceAccountUUID
		}
		if err := tx.Model(&model.Lead{}).
			Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
			Updates(updates).Error; err != nil {
			return err
		}
		if err := createLeadActivity(ctx, tx, tenantUUID, leadUUID, model.LeadActivityTypeProfileEdit, activityPayload); err != nil {
			return err
		}
		updated = lead
		return nil
	})
	if err != nil {
		return nil, err
	}
	if updated != nil {
		if err := s.attachMergeFlag(ctx, tenantUUID, updated); err != nil {
			return nil, err
		}
		if err := s.attachChannelSyncState(ctx, tenantUUID, []*model.Lead{updated}); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

func (s *LeadService) attachChannelSyncState(ctx context.Context, tenantUUID string, items []*model.Lead) error {
	if s == nil || s.repo == nil || s.repo.DB == nil || len(items) == 0 {
		return nil
	}
	leadUUIDs := make([]string, 0, len(items))
	indexByUUID := make(map[string]*model.Lead, len(items))
	latestSyncAt := make(map[string]time.Time, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		leadUUID := strings.TrimSpace(item.LeadUUID)
		if leadUUID == "" {
			continue
		}
		leadUUIDs = append(leadUUIDs, leadUUID)
		indexByUUID[leadUUID] = item
		item.ExternalUserID = ""
		item.ExternalWechatID = ""
		item.WeComFollowUserID = ""
		item.WeComAdderUserID = ""
		item.ChannelSyncStatus = "unsynced"
		setOwnerBindingStatus(item, "not_channel")
		if strings.TrimSpace(sourceAccountUUIDString(item)) != "" {
			item.LeadOriginType = "channel"
			if strings.TrimSpace(item.OwnerUserUUID) == "" {
				setOwnerBindingStatus(item, "missing_owner")
			} else {
				setOwnerBindingStatus(item, "unmapped")
			}
		} else {
			item.LeadOriginType = "local"
		}
	}
	if len(leadUUIDs) == 0 {
		return nil
	}

	var acts []model.LeadActivity
	if err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND activity_type = ? AND lead_uuid IN ?", tenantUUID, model.LeadActivityTypeSyncTrace, leadUUIDs).
		Order("updated_at DESC").
		Limit(len(leadUUIDs) * 8).
		Find(&acts).Error; err != nil {
		return err
	}

	seen := map[string]struct{}{}
	for _, act := range acts {
		leadUUID := strings.TrimSpace(act.LeadUUID)
		if leadUUID == "" {
			continue
		}
		if _, ok := seen[leadUUID]; ok {
			continue
		}
		lead := indexByUUID[leadUUID]
		if lead == nil {
			continue
		}
		payload := act.Payload
		if payload == nil {
			continue
		}
		externalUserID := strings.TrimSpace(payloadString(payload, "external_lead_id"))
		externalWechatID := strings.TrimSpace(payloadString(payload, "external_wechat_id"))
		sourceAccountUUID := strings.TrimSpace(payloadString(payload, "source_account_uuid"))
		followUserID := strings.TrimSpace(payloadString(payload, "follow_external_userid"))
		if followUserID == "" {
			followUserID = strings.TrimSpace(payloadString(payload, "owner_external_userid"))
		}
		adderUserID := strings.TrimSpace(payloadString(payload, "adder_external_userid"))
		if adderUserID == "" {
			adderUserID = strings.TrimSpace(payloadString(payload, "oper_userid"))
		}
		if externalUserID != "" {
			lead.ExternalUserID = externalUserID
			lead.ChannelSyncStatus = "synced"
			latestSyncAt[leadUUID] = act.UpdatedAt
		}
		if externalWechatID != "" {
			lead.ExternalWechatID = externalWechatID
		}
		if followUserID != "" {
			lead.WeComFollowUserID = followUserID
		}
		if adderUserID != "" {
			lead.WeComAdderUserID = adderUserID
		}
		if sourceAccountUUID != "" {
			lead.LeadOriginType = "channel"
			if lead.SourceAccountUUID == nil || strings.TrimSpace(*lead.SourceAccountUUID) == "" {
				src := sourceAccountUUID
				lead.SourceAccountUUID = &src
			}
		}
		if lead.LeadOriginType == "" {
			lead.LeadOriginType = "local"
		}
		if lead.LeadOriginType == "channel" {
			if strings.TrimSpace(lead.OwnerUserUUID) == "" {
				setOwnerBindingStatus(lead, "missing_owner")
			} else {
				setOwnerBindingStatus(lead, "unmapped")
			}
		}
		seen[leadUUID] = struct{}{}
	}

	var editActs []model.LeadActivity
	if err := s.repo.DB.WithContext(ctx).
		Select("lead_uuid, max(updated_at) AS updated_at").
		Where("tenant_uuid = ? AND activity_type = ? AND lead_uuid IN ?", tenantUUID, model.LeadActivityTypeProfileEdit, leadUUIDs).
		Group("lead_uuid").
		Find(&editActs).Error; err != nil {
		return err
	}
	for _, act := range editActs {
		leadUUID := strings.TrimSpace(act.LeadUUID)
		if leadUUID == "" {
			continue
		}
		lead := indexByUUID[leadUUID]
		if lead == nil {
			continue
		}
		syncAt, ok := latestSyncAt[leadUUID]
		if !ok {
			continue
		}
		if act.UpdatedAt.After(syncAt) {
			lead.ChannelSyncStatus = "pending_push"
		}
	}

	ownerIDs := make([]string, 0, len(items))
	accountIDs := make([]string, 0, len(items))
	ownerSeen := map[string]struct{}{}
	accountSeen := map[string]struct{}{}
	leadOwnerAccount := map[string]string{}
	for _, lead := range indexByUUID {
		if lead == nil || lead.LeadOriginType != "channel" {
			continue
		}
		ownerUserUUID := strings.TrimSpace(lead.OwnerUserUUID)
		if ownerUserUUID == "" {
			setOwnerBindingStatus(lead, "missing_owner")
			continue
		}
		sourceAccountUUID := strings.TrimSpace(sourceAccountUUIDString(lead))
		if sourceAccountUUID == "" {
			setOwnerBindingStatus(lead, "unmapped")
			continue
		}
		if _, ok := ownerSeen[ownerUserUUID]; !ok {
			ownerSeen[ownerUserUUID] = struct{}{}
			ownerIDs = append(ownerIDs, ownerUserUUID)
		}
		if _, ok := accountSeen[sourceAccountUUID]; !ok {
			accountSeen[sourceAccountUUID] = struct{}{}
			accountIDs = append(accountIDs, sourceAccountUUID)
		}
		leadOwnerAccount[lead.LeadUUID] = sourceAccountUUID + ":" + ownerUserUUID
		setOwnerBindingStatus(lead, "unmapped")
	}
	if len(ownerIDs) == 0 || len(accountIDs) == 0 {
		return nil
	}
	var rows []struct {
		MainMemberID      string `gorm:"column:main_member_id"`
		SourceAccountUUID string `gorm:"column:channel_account_uuid"`
	}
	if err := s.repo.DB.WithContext(ctx).
		Table(orgmodel.MemberBinding{}.TableName()).
		Where("tenant_uuid = ?", tenantUUID).
		Where("main_member_id IN ?", ownerIDs).
		Where("channel_account_uuid IN ?", accountIDs).
		Select("main_member_id, channel_account_uuid").
		Find(&rows).Error; err != nil {
		return err
	}
	confirmed := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.SourceAccountUUID) + ":" + strings.TrimSpace(row.MainMemberID)
		if key != ":" {
			confirmed[key] = struct{}{}
		}
	}
	for leadUUID, key := range leadOwnerAccount {
		lead := indexByUUID[leadUUID]
		if lead == nil {
			continue
		}
		if _, ok := confirmed[key]; ok {
			setOwnerBindingStatus(lead, "mapped")
		}
	}
	return nil
}

func sourceAccountUUIDString(l *model.Lead) string {
	if l == nil || l.SourceAccountUUID == nil {
		return ""
	}
	return strings.TrimSpace(*l.SourceAccountUUID)
}

func setOwnerBindingStatus(lead *model.Lead, status string) {
	if lead == nil {
		return
	}
	clean := strings.TrimSpace(strings.ToLower(status))
	lead.OwnerBindingStatus = clean
	lead.OwnerMappingStatus = clean
}

func (s *LeadService) Assign(ctx context.Context, tenantUUID, leadUUID string, req LeadAssignRequest) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	ownerUserUUID := strings.TrimSpace(req.OwnerUserUUID)
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	memberID, err := parseMemberID(ownerUserUUID)
	if err != nil {
		return nil, ErrInvalidAssignee
	}
	var updated *model.Lead
	err = s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		lead, err := getLeadByUUIDTx(ctx, tx, tenantUUID, leadUUID)
		if err != nil {
			return err
		}
		if err := ensureMemberExists(ctx, tx, tenantUUID, memberID); err != nil {
			return err
		}
		if s.assignmentSvc != nil && shouldEnforceAssigneeBinding(lead) {
			if err := s.assignmentSvc.EnsureMemberBound(ctx, tx, tenantUUID, memberID); err != nil {
				return err
			}
		}
		if err := s.transferExternalContactOwnershipIfNeeded(ctx, tx, tenantUUID, lead, ownerUserUUID); err != nil {
			return err
		}
		fromStatus := normalizeLeadStatus(lead.Status)
		if fromStatus == "" {
			fromStatus = model.LeadStatusNew
		}
		lead.OwnerUserUUID = ownerUserUUID
		lead.Status = fromStatus
		if fromStatus == model.LeadStatusNew {
			lead.Status = model.LeadStatusAssigned
		}
		lead.UpdatedAt = time.Now().UTC()
		if err := tx.Model(&model.Lead{}).
			Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
			Updates(map[string]interface{}{
				"owner_user_uuid": lead.OwnerUserUUID,
				"status":          lead.Status,
				"updated_at":      lead.UpdatedAt,
			}).Error; err != nil {
			return err
		}
		assignment := &model.LeadAssignment{
			TenantUUID:    tenantUUID,
			LeadUUID:      leadUUID,
			OwnerUserUUID: ownerUserUUID,
			Reason:        strings.TrimSpace(req.Reason),
			CreatedAt:     time.Now().UTC(),
		}
		if err := tx.Create(assignment).Error; err != nil {
			return err
		}
		if fromStatus != lead.Status {
			history := &model.LeadStatusHistory{
				TenantUUID: tenantUUID,
				LeadUUID:   leadUUID,
				FromStatus: fromStatus,
				ToStatus:   lead.Status,
				ChangedAt:  time.Now().UTC(),
			}
			if err := tx.Create(history).Error; err != nil {
				return err
			}
		}
		updated = lead
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.attachMergeFlag(ctx, tenantUUID, updated); err != nil {
		return nil, err
	}
	actor := leadobs.ResolveActorUserUUID(ctx, "")
	leadobs.EmitLeadAssigned(ctx, tenantUUID, leadUUID, actor, map[string]any{
		"owner_user_uuid": updated.OwnerUserUUID,
		"reason":          strings.TrimSpace(req.Reason),
	})
	return updated, nil
}

func (s *LeadService) BatchAssign(ctx context.Context, tenantUUID string, req LeadBatchAssignRequest) (*LeadBatchAssignResult, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	ownerUserUUID := strings.TrimSpace(req.OwnerUserUUID)
	memberID, err := parseMemberID(ownerUserUUID)
	if err != nil {
		return nil, ErrInvalidAssignee
	}
	if err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		return ensureMemberExists(ctx, tx, tenantUUID, memberID)
	}); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(req.LeadUUIDs))
	leadUUIDs := make([]string, 0, len(req.LeadUUIDs))
	for _, item := range req.LeadUUIDs {
		id := strings.ToLower(strings.TrimSpace(item))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		leadUUIDs = append(leadUUIDs, id)
	}
	result := &LeadBatchAssignResult{
		Total: len(leadUUIDs),
		Items: make([]LeadBatchAssignItem, 0, len(leadUUIDs)),
	}
	for _, leadUUID := range leadUUIDs {
		_, assignErr := s.Assign(ctx, tenantUUID, leadUUID, LeadAssignRequest{
			OwnerUserUUID: ownerUserUUID,
			Reason:        req.Reason,
		})
		if assignErr != nil {
			code, message := mapAssignError(assignErr)
			result.FailedCount++
			result.Items = append(result.Items, LeadBatchAssignItem{
				LeadUUID:     leadUUID,
				Success:      false,
				ErrorCode:    code,
				ErrorMessage: message,
			})
			continue
		}
		result.SuccessCount++
		result.Items = append(result.Items, LeadBatchAssignItem{
			LeadUUID: leadUUID,
			Success:  true,
		})
	}
	return result, nil
}

func mapAssignError(err error) (string, string) {
	switch {
	case errors.Is(err, ErrInvalidAssignee):
		return "INVALID_ASSIGNEE", "invalid assignee"
	case errors.Is(err, ErrAssigneeNotFound):
		return "ASSIGNEE_NOT_FOUND", "assignee not found"
	case errors.Is(err, ErrAssigneeTransferFailed):
		return "ASSIGNEE_TRANSFER_FAILED", strings.TrimSpace(err.Error())
	case errors.Is(err, ErrAssigneeNotBound):
		return "ASSIGNEE_NOT_BOUND", "assignee not bound to source member"
	case errors.Is(err, leadrepo.ErrLeadNotFound):
		return "LEAD_NOT_FOUND", "lead not found"
	case errors.Is(err, repository.ErrTenantUuidRequired):
		return "TENANT_REQUIRED", "tenant_uuid is required"
	default:
		return "INTERNAL_ERROR", strings.TrimSpace(err.Error())
	}
}

func (s *LeadService) transferExternalContactOwnershipIfNeeded(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID string,
	lead *model.Lead,
	newOwnerUserUUID string,
) error {
	if s == nil || tx == nil || lead == nil {
		return nil
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil
	}
	channelAccountUUID := sourceAccountUUIDString(lead)
	if channelAccountUUID == "" {
		return nil
	}
	sourceChannel := strings.ToLower(strings.TrimSpace(lead.SourceChannel))
	sourceAppType := strings.ToLower(strings.TrimSpace(lead.SourceAppType))
	if _, err := wecomauth.ResolveKind(sourceChannel, sourceAppType); err != nil {
		return nil
	}
	if sourceChannel != "wechat" {
		return nil
	}
	oldOwnerUserUUID := strings.TrimSpace(lead.OwnerUserUUID)
	newOwnerUserUUID = strings.TrimSpace(newOwnerUserUUID)
	if oldOwnerUserUUID == "" || newOwnerUserUUID == "" || oldOwnerUserUUID == newOwnerUserUUID {
		return nil
	}

	externalUserID, err := s.resolveLeadExternalUserIDForTransferTx(ctx, tx, tenantUUID, lead.LeadUUID, channelAccountUUID)
	if err != nil {
		return fmt.Errorf("%w: resolve external_userid failed: %v", ErrAssigneeTransferFailed, err)
	}
	if externalUserID == "" {
		return fmt.Errorf("%w: lead missing external_userid", ErrAssigneeTransferFailed)
	}

	handoverUserID, err := s.resolveExternalMemberIDByMainMemberTx(ctx, tx, tenantUUID, channelAccountUUID, oldOwnerUserUUID)
	if err != nil {
		return fmt.Errorf("%w: resolve handover userid failed: %v", ErrAssigneeTransferFailed, err)
	}
	if handoverUserID == "" {
		handoverUserID, err = s.resolveOwnerExternalUserIDFromSyncTraceTx(ctx, tx, tenantUUID, lead.LeadUUID, channelAccountUUID, externalUserID)
		if err != nil {
			return fmt.Errorf("%w: fallback handover userid from sync_trace failed: %v", ErrAssigneeTransferFailed, err)
		}
	}
	if handoverUserID == "" {
		return fmt.Errorf("%w: handover userid is empty", ErrAssigneeTransferFailed)
	}

	takeoverUserID, err := s.resolveExternalMemberIDByMainMemberTx(ctx, tx, tenantUUID, channelAccountUUID, newOwnerUserUUID)
	if err != nil {
		return fmt.Errorf("%w: resolve takeover userid failed: %v", ErrAssigneeTransferFailed, err)
	}
	if takeoverUserID == "" {
		return ErrAssigneeNotBound
	}
	if handoverUserID == takeoverUserID {
		return nil
	}

	accountRepo := socialrepo.NewAccountRepository(tx)
	account, err := accountRepo.GetByAccountUUID(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return fmt.Errorf("%w: resolve channel account failed: %v", ErrAssigneeTransferFailed, err)
	}
	if account == nil {
		return fmt.Errorf("%w: channel account not found", ErrAssigneeTransferFailed)
	}

	credentials := credentialsToStringMap(account.Credentials)
	openworkRepo := socialrepo.NewOpenWorkFoundationRepository(tx)
	platformRepo := socialrepo.NewChannelPlatformSettingRepository(tx)
	resolver := NewDefaultWeComLeadAdapterWithResolvers(accountRepo, openworkRepo, platformRepo)
	credentials = resolver.mergeDelegatedCredentials(ctx, tenantUUID, channelAccountUUID, credentials)

	app, err := newWeComLeadSyncApp(strings.TrimSpace(account.ChannelCode), strings.TrimSpace(account.AppType), credentials)
	if err != nil {
		return fmt.Errorf("%w: init wecom app failed: %v", ErrAssigneeTransferFailed, err)
	}
	if app == nil || app.ExternalContactTransfer == nil {
		return fmt.Errorf("%w: external contact transfer client unavailable", ErrAssigneeTransferFailed)
	}
	if app.ExternalContact == nil {
		return fmt.Errorf("%w: external contact client unavailable", ErrAssigneeTransferFailed)
	}

	// Always resolve real current follower from WeCom before transfer.
	// This avoids stale local owner mapping causing 84061.
	detailResp, err := app.ExternalContact.Get(ctx, externalUserID, "")
	if err != nil {
		return fmt.Errorf("%w: externalcontact.get failed: %v", ErrAssigneeTransferFailed, err)
	}
	if detailResp == nil {
		return fmt.Errorf("%w: externalcontact.get empty response", ErrAssigneeTransferFailed)
	}
	if detailResp.ErrCode != 0 {
		return fmt.Errorf("%w: wecom externalcontact.get failed: %d %s", ErrAssigneeTransferFailed, detailResp.ErrCode, strings.TrimSpace(detailResp.ErrMSG))
	}
	followUserIDs := make([]string, 0, len(detailResp.FollowUsers))
	for _, fu := range detailResp.FollowUsers {
		if fu == nil {
			continue
		}
		id := strings.TrimSpace(fu.UserID)
		if id == "" {
			id = strings.TrimSpace(fu.OperUserID)
		}
		if id != "" {
			followUserIDs = append(followUserIDs, id)
		}
	}
	if len(followUserIDs) == 0 {
		return fmt.Errorf("%w: externalcontact has no follow_user", ErrAssigneeTransferFailed)
	}
	resolvedHandover := ""
	for _, uid := range followUserIDs {
		if uid == handoverUserID {
			resolvedHandover = uid
			break
		}
	}
	if resolvedHandover == "" {
		resolvedHandover = followUserIDs[0]
	}
	handoverUserID = resolvedHandover

	resp, err := app.ExternalContactTransfer.TransferCustomer(ctx, &pwtransferreq.RequestTransferCustomer{
		HandoverUserID: handoverUserID,
		TakeoverUserID: takeoverUserID,
		ExternalUserID: []string{externalUserID},
	})
	if err != nil {
		return fmt.Errorf("%w: transfer_customer request failed: %v", ErrAssigneeTransferFailed, err)
	}
	if resp == nil {
		return fmt.Errorf("%w: transfer_customer empty response", ErrAssigneeTransferFailed)
	}
	if err := validateWeComResponseCode("externalcontact.transfer_customer", resp.ResponseWork); err != nil {
		return fmt.Errorf("%w: %v", ErrAssigneeTransferFailed, err)
	}
	return nil
}

func (s *LeadService) resolveExternalMemberIDByMainMemberTx(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID, channelAccountUUID, mainMemberID string,
) (string, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	mainMemberID = strings.TrimSpace(mainMemberID)
	if tenantUUID == "" || channelAccountUUID == "" || mainMemberID == "" {
		return "", nil
	}
	var row struct {
		ExternalMemberID string `gorm:"column:external_member_id"`
	}
	err := tx.WithContext(ctx).
		Table(orgmodel.MemberBinding{}.TableName()).
		Where("tenant_uuid = ? AND channel_account_uuid = ? AND main_member_id = ?", tenantUUID, channelAccountUUID, mainMemberID).
		Order("updated_at DESC").
		Limit(1).
		Select("external_member_id").
		Scan(&row).Error
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(row.ExternalMemberID), nil
}

func (s *LeadService) resolveLeadExternalUserIDForTransferTx(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID, leadUUID, channelAccountUUID string,
) (string, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.TrimSpace(leadUUID)
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || leadUUID == "" {
		return "", nil
	}
	activities := make([]model.LeadActivity, 0, 16)
	if err := tx.WithContext(ctx).
		Where(
			"tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?",
			tenantUUID,
			leadUUID,
			model.LeadActivityTypeSyncTrace,
		).
		Order("updated_at DESC").
		Limit(30).
		Find(&activities).Error; err != nil {
		return "", err
	}
	for _, activity := range activities {
		payload := activity.Payload
		if payload == nil {
			continue
		}
		if channelAccountUUID != "" && payloadString(payload, "source_account_uuid") != channelAccountUUID {
			continue
		}
		externalUserID := strings.TrimSpace(payloadString(payload, "external_lead_id"))
		if externalUserID == "" {
			externalUserID = strings.TrimSpace(payloadString(payload, "external_wechat_id"))
		}
		if externalUserID != "" {
			return externalUserID, nil
		}
	}
	return "", nil
}

func (s *LeadService) resolveOwnerExternalUserIDFromSyncTraceTx(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID, leadUUID, channelAccountUUID, externalUserID string,
) (string, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.TrimSpace(leadUUID)
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	externalUserID = strings.TrimSpace(externalUserID)
	if tenantUUID == "" || leadUUID == "" {
		return "", nil
	}
	activities := make([]model.LeadActivity, 0, 16)
	if err := tx.WithContext(ctx).
		Where(
			"tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?",
			tenantUUID,
			leadUUID,
			model.LeadActivityTypeSyncTrace,
		).
		Order("updated_at DESC").
		Limit(30).
		Find(&activities).Error; err != nil {
		return "", err
	}
	for _, activity := range activities {
		payload := activity.Payload
		if payload == nil {
			continue
		}
		if channelAccountUUID != "" && payloadString(payload, "source_account_uuid") != channelAccountUUID {
			continue
		}
		if externalUserID != "" {
			payloadExternal := strings.TrimSpace(payloadString(payload, "external_lead_id"))
			if payloadExternal == "" {
				payloadExternal = strings.TrimSpace(payloadString(payload, "external_wechat_id"))
			}
			if payloadExternal != externalUserID {
				continue
			}
		}
		operatorUserID := strings.TrimSpace(payloadString(payload, "owner_external_userid"))
		if operatorUserID != "" {
			return operatorUserID, nil
		}
	}
	return "", nil
}

func shouldEnforceAssigneeBinding(lead *model.Lead) bool {
	if lead == nil {
		return false
	}
	channel := strings.ToLower(strings.TrimSpace(lead.SourceChannel))
	appType := strings.ToLower(strings.TrimSpace(lead.SourceAppType))
	if channel != "wechat" || appType != "wecom" {
		return false
	}
	if lead.SourceAccountUUID == nil {
		return false
	}
	return strings.TrimSpace(*lead.SourceAccountUUID) != ""
}

func (s *LeadService) UpdateStatus(ctx context.Context, tenantUUID, leadUUID string, req LeadStatusUpdateRequest) (*model.Lead, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	toStatus := normalizeLeadStatus(req.Status)
	if !isValidLeadStatus(toStatus) {
		return nil, ErrInvalidLeadStatus
	}
	var updated *model.Lead
	var fromStatus string
	err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		lead, err := getLeadByUUIDTx(ctx, tx, tenantUUID, leadUUID)
		if err != nil {
			return err
		}
		fromStatus = normalizeLeadStatus(lead.Status)
		if fromStatus == "" {
			fromStatus = model.LeadStatusNew
		}
		if !isAllowedLeadStatusTransition(fromStatus, toStatus) {
			return ErrInvalidLeadStatusTransition
		}
		lead.Status = toStatus
		lead.UpdatedAt = time.Now().UTC()
		if err := tx.Model(&model.Lead{}).
			Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
			Updates(map[string]interface{}{
				"status":     lead.Status,
				"updated_at": lead.UpdatedAt,
			}).Error; err != nil {
			return err
		}
		history := &model.LeadStatusHistory{
			TenantUUID: tenantUUID,
			LeadUUID:   leadUUID,
			FromStatus: fromStatus,
			ToStatus:   toStatus,
			ChangedAt:  time.Now().UTC(),
		}
		if err := tx.Create(history).Error; err != nil {
			return err
		}
		updated = lead
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.attachMergeFlag(ctx, tenantUUID, updated); err != nil {
		return nil, err
	}
	actor := leadobs.ResolveActorUserUUID(ctx, "")
	leadobs.EmitLeadStatusChanged(ctx, tenantUUID, leadUUID, actor, map[string]any{
		"from_status": fromStatus,
		"to_status":   toStatus,
	})
	return updated, nil
}

func (s *LeadService) UpdateQualification(ctx context.Context, tenantUUID, leadUUID string, req LeadStatusUpdateRequest) (*model.Lead, error) {
	target := normalizeLeadStatus(req.Status)
	if target == "rollback" {
		if s == nil || s.repo == nil {
			return nil, errors.New("lead repository not configured")
		}
		tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
		leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
		if tenantUUID == "" || leadUUID == "" {
			return nil, repository.ErrTenantUuidRequired
		}
		lead, err := s.Get(ctx, tenantUUID, leadUUID)
		if err != nil {
			return nil, err
		}
		switch normalizeLeadStatus(lead.Status) {
		case model.LeadStatusSQL:
			target = model.LeadStatusMQL
		case model.LeadStatusMQL:
			target = model.LeadStatusInProgress
		default:
			return nil, ErrInvalidLeadStatusTransition
		}
	}
	switch target {
	case model.LeadStatusMQL, model.LeadStatusSQL:
	default:
		return nil, ErrInvalidLeadStatus
	}
	return s.UpdateStatus(ctx, tenantUUID, leadUUID, LeadStatusUpdateRequest{Status: target})
}

func (s *LeadService) ListAssignments(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadAssignment, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out []*model.LeadAssignment
	err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *LeadService) ListStatusHistory(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadStatusHistory, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out []*model.LeadStatusHistory
	err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("changed_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *LeadService) ListActivities(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadActivity, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out []*model.LeadActivity
	err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("created_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *LeadService) ListSourceEvents(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadSource, error) {
	if s == nil || s.sourceEventRepo == nil {
		return []*model.LeadSource{}, nil
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	return s.sourceEventRepo.ListByLead(ctx, tenantUUID, leadUUID)
}

type LeadImportError struct {
	Row    int
	Reason string
}

type LeadImportResult struct {
	Total   int
	Success int
	Failed  int
	Errors  []LeadImportError
}

type LeadImportPreview struct {
	Headers           []string
	SampleRows        [][]string
	SuggestedMappings map[string]int
	RequiredFields    []string
	AllFields         []string
}

func (s *LeadService) ImportCSV(ctx context.Context, tenantUUID string, reader io.Reader) (*LeadImportResult, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if reader == nil {
		return nil, errors.New("reader is required")
	}

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	header, err := csvReader.Read()
	if err != nil {
		return nil, err
	}
	headerMap := buildImportHeaderMap(header)
	if headerMap.displayName == -1 && headerMap.phone == -1 && headerMap.email == -1 {
		return nil, errors.New("missing required columns: display_name/phone/email")
	}

	result := &LeadImportResult{}
	rowIndex := 1
	for {
		rowIndex++
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, LeadImportError{Row: rowIndex, Reason: err.Error()})
			continue
		}
		payload := LeadCreateRequest{
			DisplayName:       pickField(record, headerMap.displayName),
			Phone:             pickField(record, headerMap.phone),
			Email:             pickField(record, headerMap.email),
			SourceChannel:     pickField(record, headerMap.sourceChannel),
			SourceAppType:     pickField(record, headerMap.sourceAppType),
			SourceAccountUUID: pickField(record, headerMap.sourceAccountUUID),
			OwnerUserUUID:     pickField(record, headerMap.ownerUserUUID),
		}
		if strings.TrimSpace(payload.DisplayName) == "" &&
			strings.TrimSpace(payload.Phone) == "" &&
			strings.TrimSpace(payload.Email) == "" {
			result.Failed++
			result.Errors = append(result.Errors, LeadImportError{Row: rowIndex, Reason: "姓名/手机号/邮箱至少一项"})
			continue
		}
		result.Total++
		if _, err := s.createWithOptions(ctx, tenantUUID, payload, leadCreateOptions{preserveSourceScope: true}); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, LeadImportError{Row: rowIndex, Reason: err.Error()})
			continue
		}
		result.Success++
	}
	return result, nil
}

func (s *LeadService) PreviewImportCSV(ctx context.Context, tenantUUID string, reader io.Reader) (*LeadImportPreview, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if reader == nil {
		return nil, errors.New("reader is required")
	}

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	header, err := csvReader.Read()
	if err != nil {
		return nil, err
	}
	headerMap := buildImportHeaderMap(header)
	suggested := buildSuggestedMappings(headerMap)
	samples := make([][]string, 0, 5)
	for len(samples) < 5 {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		samples = append(samples, record)
	}
	return &LeadImportPreview{
		Headers:           header,
		SampleRows:        samples,
		SuggestedMappings: suggested,
		RequiredFields:    []string{"display_name", "phone", "email"},
		AllFields: []string{
			"display_name",
			"phone",
			"email",
			"source_channel",
			"source_app_type",
			"source_account_uuid",
			"owner_user_uuid",
		},
	}, nil
}

func (s *LeadService) ImportCSVWithMapping(ctx context.Context, tenantUUID string, reader io.Reader, mapping map[string]int) (*LeadImportResult, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("lead repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if reader == nil {
		return nil, errors.New("reader is required")
	}
	if len(mapping) == 0 {
		return nil, errors.New("mapping is required")
	}
	if mapping["display_name"] < 0 && mapping["phone"] < 0 && mapping["email"] < 0 {
		return nil, errors.New("missing required mappings: display_name/phone/email")
	}

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	_, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	result := &LeadImportResult{}
	rowIndex := 1
	for {
		rowIndex++
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, LeadImportError{Row: rowIndex, Reason: err.Error()})
			continue
		}
		payload := LeadCreateRequest{
			DisplayName:       pickField(record, mapping["display_name"]),
			Phone:             pickField(record, mapping["phone"]),
			Email:             pickField(record, mapping["email"]),
			SourceChannel:     pickField(record, mapping["source_channel"]),
			SourceAppType:     pickField(record, mapping["source_app_type"]),
			SourceAccountUUID: pickField(record, mapping["source_account_uuid"]),
			OwnerUserUUID:     pickField(record, mapping["owner_user_uuid"]),
		}
		if strings.TrimSpace(payload.DisplayName) == "" &&
			strings.TrimSpace(payload.Phone) == "" &&
			strings.TrimSpace(payload.Email) == "" {
			result.Failed++
			result.Errors = append(result.Errors, LeadImportError{Row: rowIndex, Reason: "姓名/手机号/邮箱至少一项"})
			continue
		}
		result.Total++
		if _, err := s.createWithOptions(ctx, tenantUUID, payload, leadCreateOptions{preserveSourceScope: true}); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, LeadImportError{Row: rowIndex, Reason: err.Error()})
			continue
		}
		result.Success++
	}
	return result, nil
}

func parseMemberID(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, ErrInvalidAssignee
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, ErrInvalidAssignee
	}
	return id, nil
}

func ensureMemberExists(ctx context.Context, tx *gorm.DB, tenantUUID string, memberID uint64) error {
	if tx == nil {
		return errors.New("database transaction is nil")
	}
	var member iamentity.Member
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND id = ?", tenantUUID, memberID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAssigneeNotFound
		}
		return err
	}
	return nil
}

func getLeadByUUIDTx(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string) (*model.Lead, error) {
	var lead model.Lead
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		First(&lead).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, leadrepo.ErrLeadNotFound
		}
		return nil, err
	}
	return &lead, nil
}

func normalizeLeadStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func isValidLeadStatus(status string) bool {
	switch status {
	case model.LeadStatusNew,
		model.LeadStatusAssigned,
		model.LeadStatusInProgress,
		model.LeadStatusMQL,
		model.LeadStatusSQL,
		model.LeadStatusConverted,
		model.LeadStatusClosed,
		model.LeadStatusDisconnected:
		return true
	default:
		return false
	}
}

func isAllowedLeadStatusTransition(fromStatus, toStatus string) bool {
	if fromStatus == toStatus {
		return false
	}
	switch fromStatus {
	case model.LeadStatusNew:
		return toStatus == model.LeadStatusAssigned || toStatus == model.LeadStatusMQL
	case model.LeadStatusAssigned:
		return toStatus == model.LeadStatusInProgress || toStatus == model.LeadStatusMQL
	case model.LeadStatusInProgress:
		return toStatus == model.LeadStatusMQL || toStatus == model.LeadStatusConverted || toStatus == model.LeadStatusClosed
	case model.LeadStatusMQL:
		return toStatus == model.LeadStatusSQL || toStatus == model.LeadStatusInProgress || toStatus == model.LeadStatusClosed
	case model.LeadStatusSQL:
		return toStatus == model.LeadStatusMQL || toStatus == model.LeadStatusConverted || toStatus == model.LeadStatusClosed
	case model.LeadStatusDisconnected:
		return toStatus == model.LeadStatusAssigned
	default:
		return false
	}
}

func (s *LeadService) findExistingLead(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID, phone, email, sourceChannel, sourceAppType, sourceAccountUUID string,
) (*model.Lead, string, error) {
	if s != nil && s.dedupSvc != nil {
		return s.dedupSvc.FindExistingLead(ctx, tx, tenantUUID, phone, email, sourceChannel, sourceAppType, sourceAccountUUID)
	}
	if tx == nil {
		return nil, "", errors.New("database transaction is nil")
	}
	return NewDedupService().FindExistingLead(ctx, tx, tenantUUID, phone, email, sourceChannel, sourceAppType, sourceAccountUUID)
}

func (s *LeadService) createSourceTrace(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string, req LeadCreateRequest) error {
	if s == nil || s.sourceEventRepo == nil {
		return nil
	}
	sourceAccountUUID := strings.TrimSpace(req.SourceAccountUUID)
	var sourceAccountPtr *string
	if sourceAccountUUID != "" {
		sourceAccountPtr = &sourceAccountUUID
	}
	returned := &model.LeadSource{
		TenantUUID:  tenantUUID,
		LeadUUID:    leadUUID,
		ChannelCode: strings.TrimSpace(req.SourceChannel),
		AppType:     strings.TrimSpace(req.SourceAppType),
		AccountUUID: sourceAccountPtr,
	}
	if strings.EqualFold(strings.TrimSpace(req.SourceChannel), "wechat") &&
		strings.EqualFold(strings.TrimSpace(req.SourceAppType), "wecom") {
		returned.UTMSource = "wecom_sync"
	}
	_, err := s.sourceEventRepo.CreateEventTx(ctx, tx, returned)
	return err
}

func createLeadActivity(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID, activityType string, payload datatypes.JSONMap) error {
	if tx == nil {
		return errors.New("database transaction is nil")
	}
	activity := &model.LeadActivity{
		TenantUUID:   tenantUUID,
		LeadUUID:     leadUUID,
		ActivityType: activityType,
		Payload:      payload,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	return tx.Create(activity).Error
}

func buildIncomingPayload(displayName, phone, email, sourceChannel, sourceAppType, sourceAccountUUID, ownerUserUUID string) datatypes.JSONMap {
	payload := datatypes.JSONMap{}
	if strings.TrimSpace(displayName) != "" {
		payload["display_name"] = displayName
	}
	if strings.TrimSpace(phone) != "" {
		payload["phone"] = phone
	}
	if strings.TrimSpace(email) != "" {
		payload["email"] = email
	}
	if strings.TrimSpace(sourceChannel) != "" {
		payload["source_channel"] = strings.TrimSpace(sourceChannel)
	}
	if strings.TrimSpace(sourceAppType) != "" {
		payload["source_app_type"] = strings.TrimSpace(sourceAppType)
	}
	if strings.TrimSpace(sourceAccountUUID) != "" {
		payload["source_account_uuid"] = strings.TrimSpace(sourceAccountUUID)
	}
	if strings.TrimSpace(ownerUserUUID) != "" {
		payload["owner_user_uuid"] = strings.TrimSpace(ownerUserUUID)
	}
	return payload
}

type importHeaderIndex struct {
	displayName       int
	phone             int
	email             int
	sourceChannel     int
	sourceAppType     int
	sourceAccountUUID int
	ownerUserUUID     int
}

func buildImportHeaderMap(headers []string) importHeaderIndex {
	mapped := importHeaderIndex{
		displayName:       -1,
		phone:             -1,
		email:             -1,
		sourceChannel:     -1,
		sourceAppType:     -1,
		sourceAccountUUID: -1,
		ownerUserUUID:     -1,
	}
	for idx, raw := range headers {
		key := strings.ToLower(strings.TrimSpace(raw))
		switch key {
		case "display_name", "name", "姓名":
			mapped.displayName = idx
		case "phone", "mobile", "手机号":
			mapped.phone = idx
		case "email", "邮箱":
			mapped.email = idx
		case "source_channel", "channel", "渠道":
			mapped.sourceChannel = idx
		case "source_app_type", "app_type", "应用类型":
			mapped.sourceAppType = idx
		case "source_account_uuid", "account_uuid", "渠道账号uuid":
			mapped.sourceAccountUUID = idx
		case "owner_user_uuid", "owner", "负责人":
			mapped.ownerUserUUID = idx
		}
	}
	return mapped
}

func pickField(record []string, idx int) string {
	if idx < 0 || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func buildSuggestedMappings(headerMap importHeaderIndex) map[string]int {
	mapping := map[string]int{
		"display_name":        headerMap.displayName,
		"phone":               headerMap.phone,
		"email":               headerMap.email,
		"source_channel":      headerMap.sourceChannel,
		"source_app_type":     headerMap.sourceAppType,
		"source_account_uuid": headerMap.sourceAccountUUID,
		"owner_user_uuid":     headerMap.ownerUserUUID,
	}
	return mapping
}

func (s *LeadService) attachMergeFlag(ctx context.Context, tenantUUID string, lead *model.Lead) error {
	if lead == nil {
		return nil
	}
	hasMerge, err := s.repo.HasMergeActivity(ctx, tenantUUID, lead.LeadUUID)
	if err != nil {
		return err
	}
	lead.HasMerge = hasMerge
	return nil
}

func (s *LeadService) attachMergeFlags(ctx context.Context, tenantUUID string, leads []*model.Lead) error {
	if len(leads) == 0 {
		return nil
	}
	uuids := make([]string, 0, len(leads))
	for _, lead := range leads {
		if lead != nil {
			uuids = append(uuids, lead.LeadUUID)
		}
	}
	flags, err := s.repo.MergeActivityFlags(ctx, tenantUUID, uuids)
	if err != nil {
		return err
	}
	for _, lead := range leads {
		if lead == nil {
			continue
		}
		lead.HasMerge = flags[lead.LeadUUID]
	}
	return nil
}
