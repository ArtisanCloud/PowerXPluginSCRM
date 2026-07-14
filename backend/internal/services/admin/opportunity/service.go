package opportunity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	opprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/opportunity"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/customer"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrInvalidPayload            = errors.New("invalid opportunity payload")
	ErrLeadNotFound              = errors.New("lead not found")
	ErrLeadMustBeQualified       = errors.New("线索尚未达到可创建商机状态")
	ErrActiveOpportunityConflict = errors.New("lead already has active opportunity")
	ErrOpportunityNotFound       = errors.New("opportunity not found")
	ErrInvalidStage              = errors.New("invalid opportunity stage")
	ErrInvalidStageTransition    = errors.New("invalid opportunity stage transition")
	ErrTerminalOpportunity       = errors.New("terminal opportunity cannot be changed")
	ErrInvalidCloseResult        = errors.New("invalid close result")
	ErrLostReasonRequired        = errors.New("lost reason is required")
	ErrOpportunityNotTerminal    = errors.New("opportunity is not terminal")
	ErrActorUserUUIDRequired     = errors.New("缺少有效操作人 UUID")
	ErrInvalidQuoteStatus        = errors.New("invalid quote approval status")
	ErrInvalidQuoteTransition    = errors.New("invalid quote approval transition")
)

const (
	defaultQuoteStorageRoot = "tmp/opportunity-quotes"
	maxQuoteFileSize        = 50 << 20
)

type Service struct {
	repo      opprepo.OpportunityRepository
	activity  opprepo.OpportunityActivityRepository
	lineItems opprepo.OpportunityLineItemRepository
	tasks     opprepo.OpportunityTaskRepository
	db        *gorm.DB
}

type CreateRequest struct {
	LeadUUID          string
	Title             string
	OwnerUserUUID     string
	PipelineGroupUUID string
	Amount            *float64
	Currency          string
	Probability       *int
	ExpectedCloseAt   *time.Time
	ActorUserUUID     string
}

type UpdateRequest struct {
	Title                *string
	OwnerUserUUID        *string
	Amount               *float64
	Currency             *string
	Probability          *int
	ExpectedCloseAt      *time.Time
	ClearExpectedCloseAt bool
	ActorUserUUID        string
}

type LineItemRequest struct {
	Kind            string
	Name            string
	Quantity        float64
	UnitPrice       float64
	TotalAmount     float64
	Currency        string
	StorageProvider string
	ObjectKey       string
	FileName        string
	FileSize        int64
	ContentType     string
	ActorUserUUID   string
}

type QuoteApprovalRequest struct {
	Action        string
	Comment       string
	ActorUserUUID string
}

type ContractRequest struct {
	Title         string
	ContractNo    string
	CustomerUUID  string
	QuoteItemUUID string
	Amount        *float64
	Currency      string
	Status        string
	SignedAt      *time.Time
	ActorUserUUID string
}

type ContractStatusRequest struct {
	Status        string
	SignedAt      *time.Time
	ActorUserUUID string
}

type PaymentRequest struct {
	ContractUUID  string
	Title         string
	PlannedAmount float64
	DueAt         *time.Time
	ActorUserUUID string
}

type PaymentStatusRequest struct {
	Status        string
	PaidAmount    *float64
	PaidAt        *time.Time
	Method        string
	TransactionNo string
	Note          string
	ActorUserUUID string
}

type StageConfigRequest struct {
	PipelineGroupUUID string
	StageKey          string
	Label             string
	SortOrder         int
	DefaultWinRate    int
	SLADays           int
	StageType         string
	FixedStage        string
	IsActive          *bool
	MigrationPolicy   string
	ActorUserUUID     string
}

type PipelineGroupRequest struct {
	GroupKey      string
	Name          string
	Description   string
	IsDefault     bool
	CopyFromGroup string
	TemplateKey   string
	ActorUserUUID string
}

type PipelineQueryRequest struct {
	ActorUserUUID string
}

type PipelineGroupWithStages struct {
	Group  *oppmodel.OpportunityPipelineGroup `json:"group"`
	Stages []*oppmodel.OpportunityStageConfig `json:"stages"`
}

type PipelineTemplateWithStages struct {
	Template *oppmodel.OpportunityPipelineTemplate        `json:"template"`
	Stages   []*oppmodel.OpportunityPipelineTemplateStage `json:"stages"`
}

type pipelineGroupListItem struct {
	oppmodel.OpportunityPipelineGroup
	StageCount int64 `gorm:"column:stage_count"`
}

type DuplicateOpportunityCandidate struct {
	OpportunityUUID string     `json:"opportunity_uuid"`
	Title           string     `json:"title"`
	Stage           string     `json:"stage"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	OwnerUserUUID   string     `json:"owner_user_uuid"`
	LeadUUID        string     `json:"lead_uuid"`
	SourceChannel   string     `json:"source_channel"`
	ExternalUserID  string     `json:"external_userid"`
	ExpectedCloseAt *time.Time `json:"expected_close_at,omitempty"`
	Score           int        `json:"score"`
	Reason          string     `json:"reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

type MergeRequest struct {
	SourceOpportunityUUID string
	Reason                string
	ActorUserUUID         string
}

type PaymentSummary struct {
	ContractAmount float64 `json:"contract_amount"`
	PlannedAmount  float64 `json:"planned_amount"`
	PaidAmount     float64 `json:"paid_amount"`
	Outstanding    float64 `json:"outstanding"`
	OverdueAmount  float64 `json:"overdue_amount"`
	CompletionRate float64 `json:"completion_rate"`
	Currency       string  `json:"currency"`
}

type TaskRequest struct {
	Title         string
	DueAt         *time.Time
	ActorUserUUID string
}

type TaskStatusRequest struct {
	Status        string
	ActorUserUUID string
}

type ListFilter struct {
	Stage             string
	OwnerUserUUID     string
	LeadUUID          string
	Keyword           string
	SourceChannel     string
	RiskOnly          bool
	ExpectedCloseFrom *time.Time
	ExpectedCloseTo   *time.Time
	Limit             int
}

type StageRequest struct {
	Stage         string
	ActorUserUUID string
}

type CloseRequest struct {
	Result        string
	LostReason    string
	ActorUserUUID string
}

type ReopenRequest struct {
	ActorUserUUID string
}

type RiskRequest struct {
	Flag          string
	ActorUserUUID string
	Payload       map[string]any
}

type ConflictError struct {
	OpportunityUUID string
}

func (e *ConflictError) Error() string { return ErrActiveOpportunityConflict.Error() }
func (e *ConflictError) Unwrap() error { return ErrActiveOpportunityConflict }

func NewService(repo opprepo.OpportunityRepository, activity opprepo.OpportunityActivityRepository) *Service {
	return &Service{repo: repo, activity: activity}
}

func NewServiceWithDB(db *gorm.DB, repo opprepo.OpportunityRepository, activity opprepo.OpportunityActivityRepository) *Service {
	return &Service{
		repo:      repo,
		activity:  activity,
		lineItems: opprepo.NewOpportunityLineItemRepository(db),
		tasks:     opprepo.NewOpportunityTaskRepository(db),
		db:        db,
	}
}

func (s *Service) List(ctx context.Context, tenantUUID string, filter ListFilter) ([]*oppmodel.OpportunityRecord, error) {
	if s == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	return s.repo.List(ctx, tenantUUID, opprepo.OpportunityListFilter{
		Stage:             filter.Stage,
		OwnerUserUUID:     filter.OwnerUserUUID,
		LeadUUID:          filter.LeadUUID,
		Keyword:           filter.Keyword,
		SourceChannel:     filter.SourceChannel,
		RiskOnly:          filter.RiskOnly,
		ExpectedCloseFrom: filter.ExpectedCloseFrom,
		ExpectedCloseTo:   filter.ExpectedCloseTo,
		Limit:             filter.Limit,
	})
}

func (s *Service) Dashboard(ctx context.Context, tenantUUID string, filter ListFilter) (*opprepo.OpportunityDashboard, error) {
	if s == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	return s.repo.Dashboard(ctx, tenantUUID, opprepo.OpportunityListFilter{
		Stage:             filter.Stage,
		OwnerUserUUID:     filter.OwnerUserUUID,
		LeadUUID:          filter.LeadUUID,
		Keyword:           filter.Keyword,
		SourceChannel:     filter.SourceChannel,
		RiskOnly:          filter.RiskOnly,
		ExpectedCloseFrom: filter.ExpectedCloseFrom,
		ExpectedCloseTo:   filter.ExpectedCloseTo,
		Limit:             filter.Limit,
	})
}

func (s *Service) Forecast(ctx context.Context, tenantUUID string, filter ListFilter) (*opprepo.OpportunityForecast, error) {
	if s == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	return s.repo.Forecast(ctx, tenantUUID, opprepo.OpportunityListFilter{
		Stage:             filter.Stage,
		OwnerUserUUID:     filter.OwnerUserUUID,
		LeadUUID:          filter.LeadUUID,
		Keyword:           filter.Keyword,
		SourceChannel:     filter.SourceChannel,
		RiskOnly:          filter.RiskOnly,
		ExpectedCloseFrom: filter.ExpectedCloseFrom,
		ExpectedCloseTo:   filter.ExpectedCloseTo,
		Limit:             filter.Limit,
	})
}

func (s *Service) Get(ctx context.Context, tenantUUID, opportunityUUID string) (*oppmodel.OpportunityRecord, error) {
	if s == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	item, err := s.repo.GetByUUID(ctx, tenantUUID, opportunityUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOpportunityNotFound
		}
		return nil, err
	}
	return item, nil
}

func (s *Service) Create(ctx context.Context, tenantUUID string, req CreateRequest) (*oppmodel.OpportunityRecord, error) {
	if s == nil || s.repo == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	leadUUID := cleanLower(req.LeadUUID)
	title := strings.TrimSpace(req.Title)
	owner := strings.TrimSpace(req.OwnerUserUUID)
	if tenantUUID == "" || leadUUID == "" || title == "" || owner == "" {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "CNY"
	}
	probability := 0
	if req.Probability != nil {
		probability = *req.Probability
		if probability < 0 || probability > 100 {
			return nil, ErrInvalidPayload
		}
	}
	var out *oppmodel.OpportunityRecord
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		lead, err := getLeadTx(ctx, tx, tenantUUID, leadUUID)
		if err != nil {
			return err
		}
		if !isQualifiedLead(lead.Status) {
			return ErrLeadMustBeQualified
		}
		existing, err := findActiveByLeadTx(ctx, tx, tenantUUID, leadUUID)
		if err != nil {
			return err
		}
		if existing != nil {
			return &ConflictError{OpportunityUUID: existing.OpportunityUUID}
		}
		group, stages, err := ensurePipelineGroupTx(ctx, tx, tenantUUID, req.PipelineGroupUUID, actor)
		if err != nil {
			return err
		}
		initialStage := firstActiveStage(stages)
		if initialStage == nil {
			return ErrInvalidStage
		}
		var sourceAccountPtr *string
		if lead.SourceAccountUUID != nil {
			sourceAccount := strings.TrimSpace(*lead.SourceAccountUUID)
			if sourceAccount != "" {
				sourceAccountPtr = &sourceAccount
			}
		}
		now := time.Now().UTC()
		item := &oppmodel.OpportunityRecord{
			OpportunityUUID:   uuid.NewString(),
			TenantUUID:        tenantUUID,
			LeadUUID:          leadUUID,
			Title:             title,
			Stage:             firstNonEmpty(initialStage.FixedStage, initialStage.StageKey, oppmodel.StageOpen),
			PipelineGroupUUID: group.GroupUUID,
			CurrentStageUUID:  initialStage.ConfigUUID,
			Amount:            req.Amount,
			Currency:          currency,
			Probability:       firstNonZero(probability, initialStage.DefaultWinRate),
			OwnerUserUUID:     owner,
			SourceChannel:     strings.TrimSpace(lead.SourceChannel),
			SourceAppType:     strings.TrimSpace(lead.SourceAppType),
			SourceAccountUUID: sourceAccountPtr,
			ExternalUserID:    strings.TrimSpace(lead.ExternalUserID),
			ExpectedCloseAt:   req.ExpectedCloseAt,
			RiskFlags:         jsonBytes([]string{}),
			CreatedBy:         actor,
			UpdatedBy:         actor,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, item.OpportunityUUID, oppmodel.ActivityCreate, "", item.Stage, actor, map[string]any{
			"lead_uuid": leadUUID,
			"title":     title,
		}); err != nil {
			return err
		}
		out = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, tenantUUID, opportunityUUID string, req UpdateRequest) (*oppmodel.OpportunityRecord, error) {
	return s.change(ctx, tenantUUID, opportunityUUID, req.ActorUserUUID, func(item *oppmodel.OpportunityRecord, tx *gorm.DB, actor string) error {
		changed := []string{}
		if req.Title != nil {
			title := strings.TrimSpace(*req.Title)
			if title == "" {
				return ErrInvalidPayload
			}
			if item.Title != title {
				item.Title = title
				changed = append(changed, "title")
			}
		}
		if req.OwnerUserUUID != nil {
			owner := strings.TrimSpace(*req.OwnerUserUUID)
			if owner == "" {
				return ErrInvalidPayload
			}
			if item.OwnerUserUUID != owner {
				item.OwnerUserUUID = owner
				changed = append(changed, "owner_user_uuid")
			}
		}
		if req.Amount != nil {
			if item.Amount == nil || *item.Amount != *req.Amount {
				item.Amount = req.Amount
				changed = append(changed, "amount")
			}
		}
		if req.Currency != nil {
			currency := strings.ToUpper(strings.TrimSpace(*req.Currency))
			if currency == "" {
				currency = "CNY"
			}
			if item.Currency != currency {
				item.Currency = currency
				changed = append(changed, "currency")
			}
		}
		if req.Probability != nil {
			probability := *req.Probability
			if probability < 0 || probability > 100 {
				return ErrInvalidPayload
			}
			if item.Probability != probability {
				item.Probability = probability
				changed = append(changed, "probability")
			}
		}
		if req.ClearExpectedCloseAt {
			if item.ExpectedCloseAt != nil {
				item.ExpectedCloseAt = nil
				changed = append(changed, "expected_close_at")
			}
		} else if req.ExpectedCloseAt != nil {
			if item.ExpectedCloseAt == nil || !item.ExpectedCloseAt.Equal(*req.ExpectedCloseAt) {
				item.ExpectedCloseAt = req.ExpectedCloseAt
				changed = append(changed, "expected_close_at")
			}
		}
		if len(changed) == 0 {
			return nil
		}
		item.UpdatedBy = actor
		item.UpdatedAt = time.Now().UTC()
		if err := updateOpportunityTx(ctx, tx, item).Error; err != nil {
			return err
		}
		return createActivityTx(ctx, tx, tenantUUID, item.OpportunityUUID, oppmodel.ActivityNote, item.Stage, item.Stage, actor, map[string]any{
			"action":         "update",
			"changed_fields": changed,
		})
	})
}

func (s *Service) ListLineItems(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityLineItem, error) {
	if s == nil || s.lineItems == nil {
		return nil, opprepo.ErrDBNotReady
	}
	items, err := s.lineItems.ListByOpportunity(ctx, tenantUUID, opportunityUUID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		hydrateLineItemDownloadURL(item)
	}
	return items, nil
}

func (s *Service) AddLineItem(ctx context.Context, tenantUUID, opportunityUUID string, req LineItemRequest) (*oppmodel.OpportunityLineItem, error) {
	if s == nil || s.lineItems == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	name := strings.TrimSpace(req.Name)
	if tenantUUID == "" || opportunityUUID == "" || name == "" || req.Quantity <= 0 || req.UnitPrice < 0 || req.TotalAmount < 0 {
		return nil, ErrInvalidPayload
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "CNY"
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = oppmodel.LineItemKindManual
	}
	totalAmount := req.TotalAmount
	if totalAmount == 0 {
		totalAmount = req.Quantity * req.UnitPrice
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityLineItem
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if _, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID); err != nil {
			return err
		}
		now := time.Now().UTC()
		versionNo, err := nextQuoteVersionTx(ctx, tx, tenantUUID, opportunityUUID)
		if err != nil {
			return err
		}
		item := &oppmodel.OpportunityLineItem{
			ItemUUID:        uuid.NewString(),
			TenantUUID:      tenantUUID,
			OpportunityUUID: opportunityUUID,
			Kind:            kind,
			Name:            name,
			Quantity:        req.Quantity,
			UnitPrice:       req.UnitPrice,
			TotalAmount:     totalAmount,
			Currency:        currency,
			StorageProvider: strings.TrimSpace(req.StorageProvider),
			ObjectKey:       strings.TrimSpace(req.ObjectKey),
			FileName:        strings.TrimSpace(req.FileName),
			FileSize:        req.FileSize,
			ContentType:     strings.TrimSpace(req.ContentType),
			VersionNo:       versionNo,
			ApprovalStatus:  oppmodel.QuoteApprovalDraft,
			CreatedBy:       actor,
			UpdatedBy:       actor,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := tx.WithContext(ctx).Create(item).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityLineItem, "", "", actor, map[string]any{
			"action":    "create",
			"item_uuid": item.ItemUUID,
			"kind":      item.Kind,
			"name":      item.Name,
			"amount":    item.TotalAmount,
		}); err != nil {
			return err
		}
		out = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	hydrateLineItemDownloadURL(out)
	return out, nil
}

func (s *Service) AddQuoteFile(ctx context.Context, tenantUUID, opportunityUUID, actorUUID string, totalAmount float64, currency string, file multipart.File, header *multipart.FileHeader) (*oppmodel.OpportunityLineItem, error) {
	if file == nil || header == nil {
		return nil, ErrInvalidPayload
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	if tenantUUID == "" || opportunityUUID == "" || totalAmount < 0 {
		return nil, ErrInvalidPayload
	}
	if header.Size > maxQuoteFileSize {
		return nil, ErrInvalidPayload
	}
	objectKey := filepath.Join(tenantUUID, opportunityUUID, fmt.Sprintf("%d-%s", time.Now().UnixNano(), safeFileName(header.Filename)))
	targetPath := filepath.Join(quoteStorageRoot(), objectKey)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, err
	}
	dst, err := os.Create(targetPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return nil, err
	}
	return s.AddLineItem(ctx, tenantUUID, opportunityUUID, LineItemRequest{
		Kind:            oppmodel.LineItemKindQuoteFile,
		Name:            strings.TrimSpace(header.Filename),
		Quantity:        1,
		UnitPrice:       totalAmount,
		TotalAmount:     totalAmount,
		Currency:        currency,
		StorageProvider: "local",
		ObjectKey:       filepath.ToSlash(objectKey),
		FileName:        strings.TrimSpace(header.Filename),
		FileSize:        header.Size,
		ContentType:     strings.TrimSpace(header.Header.Get("Content-Type")),
		ActorUserUUID:   actorUUID,
	})
}

func (s *Service) GetLineItem(ctx context.Context, tenantUUID, opportunityUUID, itemUUID string) (*oppmodel.OpportunityLineItem, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	itemUUID = cleanLower(itemUUID)
	if tenantUUID == "" || opportunityUUID == "" || itemUUID == "" {
		return nil, ErrInvalidPayload
	}
	var item oppmodel.OpportunityLineItem
	err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ? AND item_uuid = ?", tenantUUID, opportunityUUID, itemUUID).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOpportunityNotFound
		}
		return nil, err
	}
	hydrateLineItemDownloadURL(&item)
	return &item, nil
}

func (s *Service) QuoteFilePath(item *oppmodel.OpportunityLineItem) (string, bool) {
	if item == nil || strings.TrimSpace(item.StorageProvider) != "local" || strings.TrimSpace(item.ObjectKey) == "" {
		return "", false
	}
	cleanKey := filepath.Clean(filepath.FromSlash(strings.TrimSpace(item.ObjectKey)))
	if cleanKey == "." || strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) || cleanKey == ".." || filepath.IsAbs(cleanKey) {
		return "", false
	}
	return filepath.Join(quoteStorageRoot(), cleanKey), true
}

func (s *Service) DeleteLineItem(ctx context.Context, tenantUUID, opportunityUUID, itemUUID, actorUUID string) error {
	if s == nil || s.db == nil {
		return opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	itemUUID = cleanLower(itemUUID)
	if tenantUUID == "" || opportunityUUID == "" || itemUUID == "" {
		return ErrInvalidPayload
	}
	actor, err := normalizeActor(actorUUID)
	if err != nil {
		return err
	}
	return s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if _, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND opportunity_uuid = ? AND item_uuid = ?", tenantUUID, opportunityUUID, itemUUID).
			Delete(&oppmodel.OpportunityLineItem{}).Error; err != nil {
			return err
		}
		return createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityLineItem, "", "", actor, map[string]any{
			"action":    "delete",
			"item_uuid": itemUUID,
		})
	})
}

func (s *Service) UpdateQuoteApproval(ctx context.Context, tenantUUID, opportunityUUID, itemUUID string, req QuoteApprovalRequest) (*oppmodel.OpportunityLineItem, error) {
	if s == nil || s.db == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	itemUUID = cleanLower(itemUUID)
	action := cleanLower(req.Action)
	if tenantUUID == "" || opportunityUUID == "" || itemUUID == "" || action == "" {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityLineItem
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		opportunityItem, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID)
		if err != nil {
			return err
		}
		item, err := getLineItemTx(ctx, tx, tenantUUID, opportunityUUID, itemUUID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		fromStatus := strings.TrimSpace(item.ApprovalStatus)
		if fromStatus == "" {
			fromStatus = oppmodel.QuoteApprovalDraft
		}
		toStatus := fromStatus
		payload := map[string]any{
			"item_uuid":   item.ItemUUID,
			"version_no":  item.VersionNo,
			"from_status": fromStatus,
			"action":      action,
		}
		comment := strings.TrimSpace(req.Comment)
		if comment != "" {
			payload["comment"] = comment
		}
		updates := map[string]any{
			"updated_by": actor,
			"updated_at": now,
		}
		switch action {
		case "submit":
			if fromStatus != oppmodel.QuoteApprovalDraft && fromStatus != oppmodel.QuoteApprovalWithdrawn && fromStatus != oppmodel.QuoteApprovalRejected {
				return ErrInvalidQuoteTransition
			}
			toStatus = oppmodel.QuoteApprovalSubmitted
			updates["approval_status"] = toStatus
			updates["submitted_at"] = now
			updates["approval_comment"] = comment
		case "withdraw":
			if fromStatus != oppmodel.QuoteApprovalSubmitted {
				return ErrInvalidQuoteTransition
			}
			toStatus = oppmodel.QuoteApprovalWithdrawn
			updates["approval_status"] = toStatus
			updates["approval_comment"] = comment
		case "approve":
			if fromStatus != oppmodel.QuoteApprovalSubmitted {
				return ErrInvalidQuoteTransition
			}
			toStatus = oppmodel.QuoteApprovalApproved
			updates["approval_status"] = toStatus
			updates["approved_at"] = now
			updates["approved_by"] = actor
			updates["approval_comment"] = comment
		case "reject":
			if fromStatus != oppmodel.QuoteApprovalSubmitted {
				return ErrInvalidQuoteTransition
			}
			toStatus = oppmodel.QuoteApprovalRejected
			updates["approval_status"] = toStatus
			updates["rejected_at"] = now
			updates["approved_by"] = actor
			updates["approval_comment"] = comment
		case "effective":
			if fromStatus != oppmodel.QuoteApprovalApproved && fromStatus != oppmodel.QuoteApprovalEffective {
				return ErrInvalidQuoteTransition
			}
			toStatus = oppmodel.QuoteApprovalEffective
			if err := tx.WithContext(ctx).Model(&oppmodel.OpportunityLineItem{}).
				Where("tenant_uuid = ? AND opportunity_uuid = ? AND item_uuid <> ?", tenantUUID, opportunityUUID, itemUUID).
				Updates(map[string]any{"is_effective": false}).Error; err != nil {
				return err
			}
			updates["approval_status"] = toStatus
			updates["is_effective"] = true
			updates["effective_at"] = now
			amount := item.TotalAmount
			opportunityItem.Amount = &amount
			opportunityItem.Currency = item.Currency
			opportunityItem.UpdatedBy = actor
			opportunityItem.UpdatedAt = now
			if err := updateOpportunityTx(ctx, tx, opportunityItem).Error; err != nil {
				return err
			}
			payload["amount"] = item.TotalAmount
			payload["currency"] = item.Currency
		default:
			return ErrInvalidQuoteStatus
		}
		payload["to_status"] = toStatus
		if err := tx.WithContext(ctx).Model(&oppmodel.OpportunityLineItem{}).
			Where("tenant_uuid = ? AND opportunity_uuid = ? AND item_uuid = ?", tenantUUID, opportunityUUID, itemUUID).
			Updates(updates).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityQuote, "", "", actor, payload); err != nil {
			return err
		}
		refreshed, err := getLineItemTx(ctx, tx, tenantUUID, opportunityUUID, itemUUID)
		if err != nil {
			return err
		}
		out = refreshed
		return nil
	})
	if err != nil {
		return nil, err
	}
	hydrateLineItemDownloadURL(out)
	return out, nil
}

func (s *Service) ListContracts(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityContract, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	var out []*oppmodel.OpportunityContract
	err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		Order("created_at DESC").
		Find(&out).Error
	return out, err
}

func (s *Service) AddContract(ctx context.Context, tenantUUID, opportunityUUID string, req ContractRequest) (*oppmodel.OpportunityContract, error) {
	if s == nil || s.db == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityContract
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		opp, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID)
		if err != nil {
			return err
		}
		amount := float64(0)
		currency := strings.ToUpper(strings.TrimSpace(req.Currency))
		if req.QuoteItemUUID != "" {
			quote, err := getLineItemTx(ctx, tx, tenantUUID, opportunityUUID, cleanLower(req.QuoteItemUUID))
			if err != nil {
				return err
			}
			amount = quote.TotalAmount
			currency = quote.Currency
		}
		if req.Amount != nil {
			amount = *req.Amount
		}
		if amount == 0 && opp.Amount != nil {
			amount = *opp.Amount
		}
		if currency == "" {
			currency = strings.ToUpper(strings.TrimSpace(opp.Currency))
		}
		if currency == "" {
			currency = "CNY"
		}
		status := cleanLower(req.Status)
		if status == "" {
			status = oppmodel.ContractStatusDraft
		}
		if !isValidContractStatus(status) {
			return ErrInvalidPayload
		}
		now := time.Now().UTC()
		contract := &oppmodel.OpportunityContract{
			ContractUUID:    uuid.NewString(),
			TenantUUID:      tenantUUID,
			OpportunityUUID: opportunityUUID,
			CustomerUUID:    cleanLower(req.CustomerUUID),
			QuoteItemUUID:   cleanLower(req.QuoteItemUUID),
			ContractNo:      firstNonEmpty(strings.TrimSpace(req.ContractNo), fmt.Sprintf("HT-%s", strings.ToUpper(uuid.NewString()[:8]))),
			Title:           title,
			Amount:          amount,
			Currency:        currency,
			Status:          status,
			SignedAt:        req.SignedAt,
			CreatedBy:       actor,
			UpdatedBy:       actor,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := tx.WithContext(ctx).Create(contract).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityContract, "", "", actor, map[string]any{
			"action":        "create",
			"contract_uuid": contract.ContractUUID,
			"contract_no":   contract.ContractNo,
			"amount":        contract.Amount,
			"currency":      contract.Currency,
		}); err != nil {
			return err
		}
		out = contract
		return nil
	})
	return out, err
}

func (s *Service) UpdateContractStatus(ctx context.Context, tenantUUID, opportunityUUID, contractUUID string, req ContractStatusRequest) (*oppmodel.OpportunityContract, error) {
	if s == nil || s.db == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	contractUUID = cleanLower(contractUUID)
	status := cleanLower(req.Status)
	if !isValidContractStatus(status) {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityContract
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if _, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID); err != nil {
			return err
		}
		var contract oppmodel.OpportunityContract
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ? AND contract_uuid = ?", tenantUUID, opportunityUUID, contractUUID).First(&contract).Error; err != nil {
			return err
		}
		fromStatus := contract.Status
		now := time.Now().UTC()
		signedAt := req.SignedAt
		if status == oppmodel.ContractStatusSigned && signedAt == nil {
			signedAt = &now
		}
		if err := tx.WithContext(ctx).Model(&oppmodel.OpportunityContract{}).
			Where("tenant_uuid = ? AND opportunity_uuid = ? AND contract_uuid = ?", tenantUUID, opportunityUUID, contractUUID).
			Updates(map[string]any{"status": status, "signed_at": signedAt, "updated_by": actor, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityContract, "", "", actor, map[string]any{
			"action":        "status",
			"contract_uuid": contractUUID,
			"from_status":   fromStatus,
			"to_status":     status,
		}); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ? AND contract_uuid = ?", tenantUUID, opportunityUUID, contractUUID).First(&contract).Error; err != nil {
			return err
		}
		out = &contract
		return nil
	})
	return out, err
}

func (s *Service) ListPayments(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityPayment, *PaymentSummary, error) {
	if s == nil || s.db == nil {
		return nil, nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	var items []*oppmodel.OpportunityPayment
	if err := s.db.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).Order("due_at ASC, created_at ASC").Find(&items).Error; err != nil {
		return nil, nil, err
	}
	summary := &PaymentSummary{}
	var contracts []oppmodel.OpportunityContract
	if err := s.db.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).Find(&contracts).Error; err == nil {
		for _, contract := range contracts {
			summary.ContractAmount += contract.Amount
			if summary.Currency == "" {
				summary.Currency = contract.Currency
			}
		}
	}
	now := time.Now().UTC()
	for _, item := range items {
		summary.PlannedAmount += item.PlannedAmount
		summary.PaidAmount += item.PaidAmount
		if item.Status == oppmodel.PaymentStatusOverdue || (item.Status == oppmodel.PaymentStatusPlanned && item.DueAt != nil && item.DueAt.Before(now)) {
			summary.OverdueAmount += item.PlannedAmount - item.PaidAmount
		}
		if summary.Currency == "" {
			summary.Currency = item.Currency
		}
	}
	summary.Outstanding = summary.PlannedAmount - summary.PaidAmount
	if summary.PlannedAmount > 0 {
		summary.CompletionRate = summary.PaidAmount / summary.PlannedAmount * 100
	}
	return items, summary, nil
}

func (s *Service) AddPayment(ctx context.Context, tenantUUID, opportunityUUID string, req PaymentRequest) (*oppmodel.OpportunityPayment, error) {
	if s == nil || s.db == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	contractUUID := cleanLower(req.ContractUUID)
	title := strings.TrimSpace(req.Title)
	if title == "" || contractUUID == "" || req.PlannedAmount < 0 {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityPayment
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if _, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID); err != nil {
			return err
		}
		var contract oppmodel.OpportunityContract
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ? AND contract_uuid = ?", tenantUUID, opportunityUUID, contractUUID).First(&contract).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		item := &oppmodel.OpportunityPayment{
			PaymentUUID:     uuid.NewString(),
			TenantUUID:      tenantUUID,
			OpportunityUUID: opportunityUUID,
			ContractUUID:    contractUUID,
			Title:           title,
			PlannedAmount:   req.PlannedAmount,
			Currency:        firstNonEmpty(contract.Currency, "CNY"),
			Status:          oppmodel.PaymentStatusPlanned,
			DueAt:           req.DueAt,
			CreatedBy:       actor,
			UpdatedBy:       actor,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := tx.WithContext(ctx).Create(item).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityPayment, "", "", actor, map[string]any{
			"action":        "plan",
			"payment_uuid":  item.PaymentUUID,
			"contract_uuid": contractUUID,
			"amount":        item.PlannedAmount,
		}); err != nil {
			return err
		}
		out = item
		return nil
	})
	return out, err
}

func (s *Service) UpdatePaymentStatus(ctx context.Context, tenantUUID, opportunityUUID, paymentUUID string, req PaymentStatusRequest) (*oppmodel.OpportunityPayment, error) {
	if s == nil || s.db == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	paymentUUID = cleanLower(paymentUUID)
	status := cleanLower(req.Status)
	if !isValidPaymentStatus(status) {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityPayment
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if _, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID); err != nil {
			return err
		}
		var item oppmodel.OpportunityPayment
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ? AND payment_uuid = ?", tenantUUID, opportunityUUID, paymentUUID).First(&item).Error; err != nil {
			return err
		}
		fromStatus := item.Status
		now := time.Now().UTC()
		paidAmount := item.PaidAmount
		if req.PaidAmount != nil {
			paidAmount = *req.PaidAmount
		}
		paidAt := req.PaidAt
		if status == oppmodel.PaymentStatusPaid && paidAt == nil {
			paidAt = &now
		}
		if err := tx.WithContext(ctx).Model(&oppmodel.OpportunityPayment{}).
			Where("tenant_uuid = ? AND opportunity_uuid = ? AND payment_uuid = ?", tenantUUID, opportunityUUID, paymentUUID).
			Updates(map[string]any{
				"status":         status,
				"paid_amount":    paidAmount,
				"paid_at":        paidAt,
				"method":         strings.TrimSpace(req.Method),
				"transaction_no": strings.TrimSpace(req.TransactionNo),
				"note":           strings.TrimSpace(req.Note),
				"updated_by":     actor,
				"updated_at":     now,
			}).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityPayment, "", "", actor, map[string]any{
			"action":       "status",
			"payment_uuid": paymentUUID,
			"from_status":  fromStatus,
			"to_status":    status,
			"paid_amount":  paidAmount,
		}); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ? AND payment_uuid = ?", tenantUUID, opportunityUUID, paymentUUID).First(&item).Error; err != nil {
			return err
		}
		out = &item
		return nil
	})
	return out, err
}

func (s *Service) ListStageConfigs(ctx context.Context, tenantUUID string) ([]*oppmodel.OpportunityStageConfig, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	group, err := s.defaultPipelineGroup(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return defaultStageConfigTemplates(tenantUUID, ""), nil
	}
	var out []*oppmodel.OpportunityStageConfig
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND pipeline_group_uuid = ?", tenantUUID, group.GroupUUID).
		Order("sort_order ASC, created_at ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return defaultStageConfigTemplates(tenantUUID, group.GroupUUID), nil
	}
	return out, nil
}

func (s *Service) DefaultPipeline(ctx context.Context, tenantUUID string, opts ...PipelineQueryRequest) (*PipelineGroupWithStages, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	group, err := s.defaultPipelineGroup(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	actor := actorFromPipelineOptions(opts...)
	if group == nil && actor != "" {
		var stages []*oppmodel.OpportunityStageConfig
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var txErr error
			group, stages, txErr = ensurePipelineGroupTx(ctx, tx, tenantUUID, "", actor)
			return txErr
		})
		if err != nil {
			return nil, err
		}
		return &PipelineGroupWithStages{Group: group, Stages: stages}, nil
	}
	if group == nil {
		return &PipelineGroupWithStages{Group: defaultPipelineGroupTemplate(tenantUUID), Stages: defaultStageConfigTemplates(tenantUUID, "")}, nil
	}
	var stages []*oppmodel.OpportunityStageConfig
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND pipeline_group_uuid = ? AND is_active = ?", tenantUUID, group.GroupUUID, true).
		Order("sort_order ASC, created_at ASC").
		Find(&stages).Error; err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		stages = defaultStageConfigTemplates(tenantUUID, group.GroupUUID)
	}
	return &PipelineGroupWithStages{Group: group, Stages: stages}, nil
}

func (s *Service) ListPipelineGroups(ctx context.Context, tenantUUID string, opts ...PipelineQueryRequest) ([]*oppmodel.OpportunityPipelineGroup, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	var rows []pipelineGroupListItem
	if err := s.db.WithContext(ctx).
		Table(oppmodel.OpportunityPipelineGroup{}.TableName()+" AS g").
		Select("g.*, COUNT(s.config_uuid) AS stage_count").
		Joins("LEFT JOIN "+oppmodel.OpportunityStageConfig{}.TableName()+" AS s ON s.tenant_uuid = g.tenant_uuid AND s.pipeline_group_uuid = g.group_uuid AND s.is_active = ?", true).
		Where("g.tenant_uuid = ? AND g.is_active = ?", tenantUUID, true).
		Group("g.group_uuid").
		Order("is_default DESC, sort_order ASC, created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		if actor := actorFromPipelineOptions(opts...); actor != "" {
			err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				_, _, txErr := ensurePipelineGroupTx(ctx, tx, tenantUUID, "", actor)
				return txErr
			})
			if err != nil {
				return nil, err
			}
			return s.ListPipelineGroups(ctx, tenantUUID)
		}
		return []*oppmodel.OpportunityPipelineGroup{defaultPipelineGroupTemplate(tenantUUID)}, nil
	}
	return dedupePipelineGroupListItems(rows), nil
}

func (s *Service) ListPipelineTemplates(ctx context.Context) ([]*PipelineTemplateWithStages, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	var templates []*oppmodel.OpportunityPipelineTemplate
	if err := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC, created_at ASC").
		Find(&templates).Error; err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(templates))
	for _, template := range templates {
		if template != nil {
			keys = append(keys, cleanLower(template.TemplateKey))
		}
	}
	var stages []*oppmodel.OpportunityPipelineTemplateStage
	if err := s.db.WithContext(ctx).
		Where("template_key IN ? AND is_active = ?", keys, true).
		Order("template_key ASC, sort_order ASC, created_at ASC").
		Find(&stages).Error; err != nil {
		return nil, err
	}
	stageMap := make(map[string][]*oppmodel.OpportunityPipelineTemplateStage, len(templates))
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		key := cleanLower(stage.TemplateKey)
		stageMap[key] = append(stageMap[key], stage)
	}
	out := make([]*PipelineTemplateWithStages, 0, len(templates))
	for _, template := range templates {
		if template == nil {
			continue
		}
		key := cleanLower(template.TemplateKey)
		out = append(out, &PipelineTemplateWithStages{
			Template: template,
			Stages:   stageMap[key],
		})
	}
	return out, nil
}

func (s *Service) GetPipelineGroup(ctx context.Context, tenantUUID, groupUUID string) (*PipelineGroupWithStages, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	groupUUID = cleanLower(groupUUID)
	if tenantUUID == "" || groupUUID == "" {
		return nil, ErrInvalidPayload
	}
	var group oppmodel.OpportunityPipelineGroup
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND group_uuid = ? AND is_active = ?", tenantUUID, groupUUID, true).
		First(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidPayload
		}
		return nil, err
	}
	var stages []*oppmodel.OpportunityStageConfig
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND pipeline_group_uuid = ?", tenantUUID, group.GroupUUID).
		Order("sort_order ASC, created_at ASC").
		Find(&stages).Error; err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		stages = defaultStageConfigTemplates(tenantUUID, group.GroupUUID)
	}
	return &PipelineGroupWithStages{Group: &group, Stages: stages}, nil
}

func (s *Service) CreatePipelineGroup(ctx context.Context, tenantUUID string, req PipelineGroupRequest) (*PipelineGroupWithStages, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrInvalidPayload
	}
	groupKey := cleanLower(req.GroupKey)
	if groupKey == "" {
		groupKey = cleanLower(name)
	}
	groupKey = strings.NewReplacer(" ", "-", "_", "-").Replace(groupKey)
	now := time.Now().UTC()
	var out PipelineGroupWithStages
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingCount int64
		if err := tx.Model(&oppmodel.OpportunityPipelineGroup{}).
			Where("tenant_uuid = ? AND group_key = ? AND is_active = ?", tenantUUID, groupKey, true).
			Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			return ErrInvalidPayload
		}
		if req.IsDefault {
			if err := tx.Model(&oppmodel.OpportunityPipelineGroup{}).
				Where("tenant_uuid = ?", tenantUUID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		group := &oppmodel.OpportunityPipelineGroup{
			GroupUUID:   uuid.NewString(),
			TenantUUID:  tenantUUID,
			GroupKey:    groupKey,
			Name:        name,
			Description: strings.TrimSpace(req.Description),
			IsDefault:   req.IsDefault,
			IsActive:    true,
			SortOrder:   100,
			CreatedBy:   actor,
			UpdatedBy:   actor,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		stages, err := s.pipelineTemplateStages(ctx, tx, tenantUUID, group.GroupUUID, req.TemplateKey)
		if err != nil {
			return err
		}
		if req.CopyFromGroup != "" {
			var source []*oppmodel.OpportunityStageConfig
			if err := tx.Where("tenant_uuid = ? AND pipeline_group_uuid = ?", tenantUUID, cleanLower(req.CopyFromGroup)).
				Order("sort_order ASC, created_at ASC").
				Find(&source).Error; err != nil {
				return err
			}
			if len(source) > 0 {
				stages = make([]*oppmodel.OpportunityStageConfig, 0, len(source))
				for _, item := range source {
					copyItem := *item
					copyItem.ConfigUUID = ""
					copyItem.PipelineGroupUUID = group.GroupUUID
					stages = append(stages, &copyItem)
				}
			}
		}
		for _, stage := range stages {
			stage.ConfigUUID = uuid.NewString()
			stage.TenantUUID = tenantUUID
			stage.PipelineGroupUUID = group.GroupUUID
			stage.CreatedBy = actor
			stage.UpdatedBy = actor
			stage.CreatedAt = now
			stage.UpdatedAt = now
			if err := tx.Create(stage).Error; err != nil {
				return err
			}
		}
		out.Group = group
		out.Stages = stages
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) defaultPipelineGroup(ctx context.Context, tenantUUID string) (*oppmodel.OpportunityPipelineGroup, error) {
	var group oppmodel.OpportunityPipelineGroup
	err := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND is_default = ? AND is_active = ?", tenantUUID, true, true).
		Order("sort_order ASC, created_at ASC").
		First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

func (s *Service) SaveStageConfig(ctx context.Context, tenantUUID string, req StageConfigRequest) (*oppmodel.OpportunityStageConfig, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	stageKey := cleanLower(req.StageKey)
	label := strings.TrimSpace(req.Label)
	fixedStage := cleanLower(req.FixedStage)
	stageType := cleanLower(req.StageType)
	if stageType == "" {
		stageType = oppmodel.StageTypeActive
	}
	if stageKey == "" || label == "" || !isValidStageType(stageType) || !isValidStage(fixedStage) {
		return nil, ErrInvalidPayload
	}
	if stageType == oppmodel.StageTypeActive && !isActiveStage(fixedStage) {
		return nil, ErrInvalidPayload
	}
	if stageType == oppmodel.StageTypeWon && fixedStage != oppmodel.StageWon {
		return nil, ErrInvalidPayload
	}
	if stageType == oppmodel.StageTypeLost && fixedStage != oppmodel.StageLost {
		return nil, ErrInvalidPayload
	}
	if req.DefaultWinRate < 0 || req.DefaultWinRate > 100 || req.SLADays < 0 {
		return nil, ErrInvalidPayload
	}
	policy := firstNonEmpty(cleanLower(req.MigrationPolicy), "map_to_fixed")
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	now := time.Now().UTC()
	var out oppmodel.OpportunityStageConfig
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, _, err := ensurePipelineGroupTx(ctx, tx, tenantUUID, req.PipelineGroupUUID, actor)
		if err != nil {
			return err
		}
		var existing oppmodel.OpportunityStageConfig
		err = tx.Where("tenant_uuid = ? AND pipeline_group_uuid = ? AND stage_key = ?", tenantUUID, group.GroupUUID, stageKey).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			existing = oppmodel.OpportunityStageConfig{
				ConfigUUID:        uuid.NewString(),
				TenantUUID:        tenantUUID,
				PipelineGroupUUID: group.GroupUUID,
				StageKey:          stageKey,
				CreatedBy:         actor,
				CreatedAt:         now,
			}
		}
		existing.Label = label
		existing.SortOrder = req.SortOrder
		existing.DefaultWinRate = req.DefaultWinRate
		existing.SLADays = req.SLADays
		existing.StageType = stageType
		existing.FixedStage = fixedStage
		existing.IsActive = isActive
		existing.MigrationPolicy = policy
		existing.UpdatedBy = actor
		existing.UpdatedAt = now
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
		out = existing
		return nil
	})
	return &out, err
}

func (s *Service) DetectDuplicates(ctx context.Context, tenantUUID, opportunityUUID string) ([]DuplicateOpportunityCandidate, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	target, err := s.Get(ctx, tenantUUID, opportunityUUID)
	if err != nil {
		return nil, err
	}
	var rows []oppmodel.OpportunityRecord
	query := s.db.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid <> ?", tenantUUID, opportunityUUID).
		Where("stage IN ?", []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation, oppmodel.StageWon, oppmodel.StageLost})
	if err := query.Order("updated_at DESC").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DuplicateOpportunityCandidate, 0)
	for _, item := range rows {
		score, reason := duplicateScore(target, &item)
		if score < 60 {
			continue
		}
		amount := 0.0
		if item.Amount != nil {
			amount = *item.Amount
		}
		out = append(out, DuplicateOpportunityCandidate{
			OpportunityUUID: item.OpportunityUUID,
			Title:           item.Title,
			Stage:           item.Stage,
			Amount:          amount,
			Currency:        item.Currency,
			OwnerUserUUID:   item.OwnerUserUUID,
			LeadUUID:        item.LeadUUID,
			SourceChannel:   item.SourceChannel,
			ExternalUserID:  item.ExternalUserID,
			ExpectedCloseAt: item.ExpectedCloseAt,
			Score:           score,
			Reason:          reason,
			CreatedAt:       item.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) MergeOpportunity(ctx context.Context, tenantUUID, targetOpportunityUUID string, req MergeRequest) (*oppmodel.OpportunityRecord, error) {
	if s == nil || s.db == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	targetOpportunityUUID = cleanLower(targetOpportunityUUID)
	sourceOpportunityUUID := cleanLower(req.SourceOpportunityUUID)
	if targetOpportunityUUID == "" || sourceOpportunityUUID == "" || targetOpportunityUUID == sourceOpportunityUUID {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityRecord
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		target, err := getOpportunityTx(ctx, tx, tenantUUID, targetOpportunityUUID)
		if err != nil {
			return err
		}
		source, err := getOpportunityTx(ctx, tx, tenantUUID, sourceOpportunityUUID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, table := range []string{
			oppmodel.OpportunityActivity{}.TableName(),
			oppmodel.OpportunityLineItem{}.TableName(),
			oppmodel.OpportunityTask{}.TableName(),
			oppmodel.OpportunityContract{}.TableName(),
			oppmodel.OpportunityPayment{}.TableName(),
		} {
			if err := tx.WithContext(ctx).
				Table(table).
				Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, sourceOpportunityUUID).
				Update("opportunity_uuid", targetOpportunityUUID).Error; err != nil {
				return err
			}
		}
		if err := tx.WithContext(ctx).
			Model(&oppmodel.OpportunityRecord{}).
			Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, sourceOpportunityUUID).
			Updates(map[string]any{
				"stage":       oppmodel.StageLost,
				"lost_at":     now,
				"lost_reason": firstNonEmpty(strings.TrimSpace(req.Reason), "merged into "+targetOpportunityUUID),
				"updated_by":  actor,
				"updated_at":  now,
			}).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, targetOpportunityUUID, oppmodel.ActivityMerge, "", "", actor, map[string]any{
			"action":                  "merge",
			"target_opportunity_uuid": targetOpportunityUUID,
			"source_opportunity_uuid": sourceOpportunityUUID,
			"source_title":            source.Title,
			"reason":                  strings.TrimSpace(req.Reason),
		}); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, targetOpportunityUUID).First(target).Error; err != nil {
			return err
		}
		out = target
		return nil
	})
	return out, err
}

func (s *Service) ListTasks(ctx context.Context, tenantUUID, opportunityUUID string) ([]*oppmodel.OpportunityTask, error) {
	if s == nil || s.tasks == nil {
		return nil, opprepo.ErrDBNotReady
	}
	return s.tasks.ListByOpportunity(ctx, tenantUUID, opportunityUUID)
}

func (s *Service) AddTask(ctx context.Context, tenantUUID, opportunityUUID string, req TaskRequest) (*oppmodel.OpportunityTask, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	title := strings.TrimSpace(req.Title)
	if tenantUUID == "" || opportunityUUID == "" || title == "" {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityTask
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if _, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID); err != nil {
			return err
		}
		now := time.Now().UTC()
		item := &oppmodel.OpportunityTask{
			TaskUUID:        uuid.NewString(),
			TenantUUID:      tenantUUID,
			OpportunityUUID: opportunityUUID,
			Title:           title,
			DueAt:           req.DueAt,
			Status:          oppmodel.TaskStatusOpen,
			CreatedBy:       actor,
			UpdatedBy:       actor,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := tx.WithContext(ctx).Create(item).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityTask, "", "", actor, map[string]any{
			"action":    "create",
			"task_uuid": item.TaskUUID,
			"title":     item.Title,
		}); err != nil {
			return err
		}
		out = item
		return nil
	})
	return out, err
}

func (s *Service) UpdateTaskStatus(ctx context.Context, tenantUUID, opportunityUUID, taskUUID string, req TaskStatusRequest) (*oppmodel.OpportunityTask, error) {
	if s == nil || s.db == nil {
		return nil, opprepo.ErrDBNotReady
	}
	status := cleanLower(req.Status)
	if status != oppmodel.TaskStatusOpen && status != oppmodel.TaskStatusDone {
		return nil, ErrInvalidPayload
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	taskUUID = cleanLower(taskUUID)
	actor, err := normalizeActor(req.ActorUserUUID)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityTask
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if _, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID); err != nil {
			return err
		}
		var item oppmodel.OpportunityTask
		if err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND opportunity_uuid = ? AND task_uuid = ?", tenantUUID, opportunityUUID, taskUUID).
			First(&item).Error; err != nil {
			return err
		}
		if item.Status == status {
			out = &item
			return nil
		}
		item.Status = status
		item.UpdatedBy = actor
		item.UpdatedAt = time.Now().UTC()
		if err := tx.WithContext(ctx).
			Model(&oppmodel.OpportunityTask{}).
			Where("tenant_uuid = ? AND opportunity_uuid = ? AND task_uuid = ?", tenantUUID, opportunityUUID, taskUUID).
			Updates(map[string]any{"status": item.Status, "updated_by": item.UpdatedBy, "updated_at": item.UpdatedAt}).Error; err != nil {
			return err
		}
		if err := createActivityTx(ctx, tx, tenantUUID, opportunityUUID, oppmodel.ActivityTask, "", "", actor, map[string]any{
			"action":    "status",
			"task_uuid": taskUUID,
			"status":    status,
		}); err != nil {
			return err
		}
		out = &item
		return nil
	})
	return out, err
}

func (s *Service) AdvanceStage(ctx context.Context, tenantUUID, opportunityUUID string, req StageRequest) (*oppmodel.OpportunityRecord, error) {
	toStage := cleanLower(req.Stage)
	if !isActiveStage(toStage) {
		return nil, ErrInvalidStage
	}
	return s.change(ctx, tenantUUID, opportunityUUID, req.ActorUserUUID, func(item *oppmodel.OpportunityRecord, tx *gorm.DB, actor string) error {
		fromStage := cleanLower(item.Stage)
		if isTerminalStage(fromStage) {
			return ErrTerminalOpportunity
		}
		if fromStage == toStage {
			return nil
		}
		group, stages, err := ensurePipelineGroupTx(ctx, tx, tenantUUID, item.PipelineGroupUUID, actor)
		if err != nil {
			return err
		}
		fromConfig := stageByFixed(stages, fromStage)
		toConfig := stageByFixed(stages, toStage)
		if fromConfig == nil || toConfig == nil {
			return ErrInvalidStage
		}
		if item.CurrentStageUUID == "" {
			item.CurrentStageUUID = fromConfig.ConfigUUID
		}
		if !canAdvanceInPipeline(stages, item.CurrentStageUUID, toConfig.ConfigUUID) && !canAdvance(fromStage, toStage) {
			return ErrInvalidStageTransition
		}
		item.Stage = toStage
		item.PipelineGroupUUID = group.GroupUUID
		item.CurrentStageUUID = toConfig.ConfigUUID
		item.Probability = firstNonZero(item.Probability, toConfig.DefaultWinRate)
		item.UpdatedBy = actor
		item.UpdatedAt = time.Now().UTC()
		if err := updateOpportunityTx(ctx, tx, item).Error; err != nil {
			return err
		}
		return createActivityTx(ctx, tx, tenantUUID, item.OpportunityUUID, oppmodel.ActivityStageChange, fromStage, toStage, actor, map[string]any{})
	})
}

func (s *Service) Close(ctx context.Context, tenantUUID, opportunityUUID string, req CloseRequest) (*oppmodel.OpportunityRecord, error) {
	result := cleanLower(req.Result)
	if result != oppmodel.StageWon && result != oppmodel.StageLost {
		return nil, ErrInvalidCloseResult
	}
	lostReason := strings.TrimSpace(req.LostReason)
	if result == oppmodel.StageLost && lostReason == "" {
		return nil, ErrLostReasonRequired
	}
	return s.change(ctx, tenantUUID, opportunityUUID, req.ActorUserUUID, func(item *oppmodel.OpportunityRecord, tx *gorm.DB, actor string) error {
		fromStage := cleanLower(item.Stage)
		if fromStage == result {
			return nil
		}
		if isTerminalStage(fromStage) {
			return ErrTerminalOpportunity
		}
		now := time.Now().UTC()
		group, stages, err := ensurePipelineGroupTx(ctx, tx, tenantUUID, item.PipelineGroupUUID, actor)
		if err != nil {
			return err
		}
		resultConfig := stageByFixed(stages, result)
		if resultConfig == nil {
			return ErrInvalidStage
		}
		item.Stage = result
		item.PipelineGroupUUID = group.GroupUUID
		item.CurrentStageUUID = resultConfig.ConfigUUID
		item.Probability = resultConfig.DefaultWinRate
		item.UpdatedBy = actor
		item.UpdatedAt = now
		if result == oppmodel.StageWon {
			item.WonAt = &now
			if err := ensureCustomerForOpportunityTx(ctx, tx, tenantUUID, item); err != nil {
				return err
			}
		} else {
			item.LostAt = &now
			item.LostReason = lostReason
			if err := closeLeadTx(ctx, tx, tenantUUID, item.LeadUUID); err != nil {
				return err
			}
		}
		if err := updateOpportunityTx(ctx, tx, item).Error; err != nil {
			return err
		}
		return createActivityTx(ctx, tx, tenantUUID, item.OpportunityUUID, oppmodel.ActivityClose, fromStage, result, actor, map[string]any{
			"result":      result,
			"lost_reason": lostReason,
		})
	})
}

func (s *Service) Reopen(ctx context.Context, tenantUUID, opportunityUUID string, req ReopenRequest) (*oppmodel.OpportunityRecord, error) {
	return s.change(ctx, tenantUUID, opportunityUUID, req.ActorUserUUID, func(item *oppmodel.OpportunityRecord, tx *gorm.DB, actor string) error {
		fromStage := cleanLower(item.Stage)
		if fromStage == oppmodel.StageOpen {
			return nil
		}
		if !isTerminalStage(fromStage) {
			return ErrOpportunityNotTerminal
		}
		group, stages, err := ensurePipelineGroupTx(ctx, tx, tenantUUID, item.PipelineGroupUUID, actor)
		if err != nil {
			return err
		}
		initialStage := firstActiveStage(stages)
		if initialStage == nil {
			return ErrInvalidStage
		}
		item.Stage = firstNonEmpty(initialStage.FixedStage, initialStage.StageKey, oppmodel.StageOpen)
		item.PipelineGroupUUID = group.GroupUUID
		item.CurrentStageUUID = initialStage.ConfigUUID
		item.Probability = firstNonZero(item.Probability, initialStage.DefaultWinRate)
		item.WonAt = nil
		item.LostAt = nil
		item.LostReason = ""
		item.UpdatedBy = actor
		item.UpdatedAt = time.Now().UTC()
		if err := updateOpportunityTx(ctx, tx, item).Error; err != nil {
			return err
		}
		return createActivityTx(ctx, tx, tenantUUID, item.OpportunityUUID, oppmodel.ActivityReopen, fromStage, item.Stage, actor, map[string]any{})
	})
}

func (s *Service) MarkRisk(ctx context.Context, tenantUUID, opportunityUUID string, req RiskRequest) (*oppmodel.OpportunityRecord, error) {
	flag := cleanLower(req.Flag)
	if flag == "" {
		flag = "disconnected"
	}
	return s.change(ctx, tenantUUID, opportunityUUID, req.ActorUserUUID, func(item *oppmodel.OpportunityRecord, tx *gorm.DB, actor string) error {
		flags := parseRiskFlags(item.RiskFlags)
		exists := false
		for _, current := range flags {
			if current == flag {
				exists = true
				break
			}
		}
		if !exists {
			flags = append(flags, flag)
			item.RiskFlags = jsonBytes(flags)
			item.UpdatedBy = actor
			item.UpdatedAt = time.Now().UTC()
			if err := updateOpportunityTx(ctx, tx, item).Error; err != nil {
				return err
			}
		}
		payload := req.Payload
		if payload == nil {
			payload = map[string]any{}
		}
		payload["flag"] = flag
		return createActivityTx(ctx, tx, tenantUUID, item.OpportunityUUID, oppmodel.ActivityRiskFlag, "", item.Stage, actor, payload)
	})
}

func (s *Service) ListActivities(ctx context.Context, tenantUUID, opportunityUUID string, limit int) ([]*oppmodel.OpportunityActivity, error) {
	if s == nil || s.activity == nil {
		return nil, opprepo.ErrDBNotReady
	}
	return s.activity.ListByOpportunity(ctx, tenantUUID, opportunityUUID, limit)
}

func (s *Service) change(ctx context.Context, tenantUUID, opportunityUUID, actor string, fn func(*oppmodel.OpportunityRecord, *gorm.DB, string) error) (*oppmodel.OpportunityRecord, error) {
	if s == nil || s.repo == nil {
		return nil, opprepo.ErrDBNotReady
	}
	tenantUUID = cleanLower(tenantUUID)
	opportunityUUID = cleanLower(opportunityUUID)
	if tenantUUID == "" || opportunityUUID == "" {
		return nil, ErrInvalidPayload
	}
	actor, err := normalizeActor(actor)
	if err != nil {
		return nil, err
	}
	var out *oppmodel.OpportunityRecord
	err = s.repo.BeginTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		item, err := getOpportunityTx(ctx, tx, tenantUUID, opportunityUUID)
		if err != nil {
			return err
		}
		if err := fn(item, tx, actor); err != nil {
			return err
		}
		out = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func getLeadTx(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string) (*leadmodel.Lead, error) {
	var lead leadmodel.Lead
	err := tx.WithContext(ctx).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).First(&lead).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLeadNotFound
		}
		return nil, err
	}
	return &lead, nil
}

func getOpportunityTx(ctx context.Context, tx *gorm.DB, tenantUUID, opportunityUUID string) (*oppmodel.OpportunityRecord, error) {
	var item oppmodel.OpportunityRecord
	err := tx.WithContext(ctx).Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOpportunityNotFound
		}
		return nil, err
	}
	return &item, nil
}

func findActiveByLeadTx(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string) (*oppmodel.OpportunityRecord, error) {
	var item oppmodel.OpportunityRecord
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ? AND stage IN ?", tenantUUID, leadUUID, []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}).
		Order("created_at DESC").
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func ensurePipelineGroupTx(ctx context.Context, tx *gorm.DB, tenantUUID, requestedGroupUUID, actor string) (*oppmodel.OpportunityPipelineGroup, []*oppmodel.OpportunityStageConfig, error) {
	tenantUUID = cleanLower(tenantUUID)
	requestedGroupUUID = cleanLower(requestedGroupUUID)
	actor = strings.TrimSpace(actor)
	if tenantUUID == "" || actor == "" {
		return nil, nil, ErrInvalidPayload
	}
	var group oppmodel.OpportunityPipelineGroup
	query := tx.WithContext(ctx).Where("tenant_uuid = ? AND is_active = ?", tenantUUID, true)
	var err error
	if requestedGroupUUID != "" {
		err = query.Where("group_uuid = ?", requestedGroupUUID).First(&group).Error
	} else {
		err = query.Where("is_default = ?", true).Order("sort_order ASC, created_at ASC").First(&group).Error
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}
	now := time.Now().UTC()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if requestedGroupUUID != "" {
			return nil, nil, ErrInvalidPayload
		}
		group = *defaultPipelineGroupTemplate(tenantUUID)
		group.GroupUUID = uuid.NewString()
		group.CreatedBy = actor
		group.UpdatedBy = actor
		group.CreatedAt = now
		group.UpdatedAt = now
		if err := tx.WithContext(ctx).Create(&group).Error; err != nil {
			return nil, nil, err
		}
		for _, stage := range defaultStageConfigTemplates(tenantUUID, group.GroupUUID) {
			stage.ConfigUUID = uuid.NewString()
			stage.CreatedBy = actor
			stage.UpdatedBy = actor
			stage.CreatedAt = now
			stage.UpdatedAt = now
			if err := tx.WithContext(ctx).Create(stage).Error; err != nil {
				return nil, nil, err
			}
		}
	}
	var stages []*oppmodel.OpportunityStageConfig
	if err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND pipeline_group_uuid = ? AND is_active = ?", tenantUUID, group.GroupUUID, true).
		Order("sort_order ASC, created_at ASC").
		Find(&stages).Error; err != nil {
		return nil, nil, err
	}
	if len(stages) == 0 {
		for _, stage := range defaultStageConfigTemplates(tenantUUID, group.GroupUUID) {
			stage.ConfigUUID = uuid.NewString()
			stage.CreatedBy = actor
			stage.UpdatedBy = actor
			stage.CreatedAt = now
			stage.UpdatedAt = now
			if err := tx.WithContext(ctx).Create(stage).Error; err != nil {
				return nil, nil, err
			}
			stages = append(stages, stage)
		}
	}
	return &group, stages, nil
}

func getLineItemTx(ctx context.Context, tx *gorm.DB, tenantUUID, opportunityUUID, itemUUID string) (*oppmodel.OpportunityLineItem, error) {
	var item oppmodel.OpportunityLineItem
	err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND opportunity_uuid = ? AND item_uuid = ?", tenantUUID, opportunityUUID, itemUUID).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOpportunityNotFound
		}
		return nil, err
	}
	return &item, nil
}

func nextQuoteVersionTx(ctx context.Context, tx *gorm.DB, tenantUUID, opportunityUUID string) (int, error) {
	var maxVersion int
	err := tx.WithContext(ctx).Model(&oppmodel.OpportunityLineItem{}).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", tenantUUID, opportunityUUID).
		Select("COALESCE(MAX(version_no), 0)").
		Scan(&maxVersion).Error
	if err != nil {
		return 0, err
	}
	return maxVersion + 1, nil
}

func updateOpportunityTx(ctx context.Context, tx *gorm.DB, item *oppmodel.OpportunityRecord) *gorm.DB {
	return tx.WithContext(ctx).Model(&oppmodel.OpportunityRecord{}).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", item.TenantUUID, item.OpportunityUUID).
		Updates(map[string]any{
			"title":               item.Title,
			"stage":               item.Stage,
			"pipeline_group_uuid": item.PipelineGroupUUID,
			"current_stage_uuid":  item.CurrentStageUUID,
			"amount":              item.Amount,
			"currency":            item.Currency,
			"probability":         item.Probability,
			"owner_user_uuid":     item.OwnerUserUUID,
			"expected_close_at":   item.ExpectedCloseAt,
			"won_at":              item.WonAt,
			"lost_at":             item.LostAt,
			"lost_reason":         item.LostReason,
			"risk_flags":          item.RiskFlags,
			"updated_by":          item.UpdatedBy,
			"updated_at":          item.UpdatedAt,
		})
}

func createActivityTx(ctx context.Context, tx *gorm.DB, tenantUUID, opportunityUUID, activityType, fromStage, toStage, actor string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	item := &oppmodel.OpportunityActivity{
		ActivityUUID:     uuid.NewString(),
		TenantUUID:       tenantUUID,
		OpportunityUUID:  opportunityUUID,
		ActivityType:     activityType,
		FromStage:        fromStage,
		ToStage:          toStage,
		Payload:          jsonBytes(payload),
		OperatorUserUUID: actor,
		CreatedAt:        time.Now().UTC(),
	}
	return tx.WithContext(ctx).Create(item).Error
}

func closeLeadTx(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string) error {
	return tx.WithContext(ctx).Model(&leadmodel.Lead{}).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Updates(map[string]any{
			"status":     leadmodel.LeadStatusClosed,
			"updated_at": time.Now().UTC(),
		}).Error
}

func ensureCustomerForOpportunityTx(ctx context.Context, tx *gorm.DB, tenantUUID string, item *oppmodel.OpportunityRecord) error {
	lead, err := getLeadTx(ctx, tx, tenantUUID, item.LeadUUID)
	if err != nil {
		return err
	}
	phone := strings.TrimSpace(lead.Phone)
	email := strings.TrimSpace(lead.Email)
	externalUserID := strings.TrimSpace(item.ExternalUserID)
	query := tx.WithContext(ctx).Model(&customermodel.CustomerAccount{}).Where("tenant_uuid = ?", tenantUUID)
	switch {
	case externalUserID != "":
		query = query.Where("metadata ->> 'external_userid' = ?", externalUserID)
	case phone != "":
		query = query.Where("phone = ?", phone)
	case email != "":
		query = query.Where("email = ?", email)
	default:
		return nil
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	customer := &customermodel.CustomerAccount{
		CustomerUUID: uuid.NewString(),
		Email:        email,
		Phone:        phone,
		Status:       "active",
		Metadata: datatypes.JSONMap{
			"source":              "opportunity",
			"opportunity_uuid":    item.OpportunityUUID,
			"lead_uuid":           item.LeadUUID,
			"source_channel":      item.SourceChannel,
			"source_app_type":     item.SourceAppType,
			"source_account_uuid": derefString(item.SourceAccountUUID),
			"external_userid":     externalUserID,
		},
	}
	customer.TenantUuid = tenantUUID
	return tx.WithContext(ctx).Create(customer).Error
}

func isQualifiedLead(status string) bool {
	status = cleanLower(status)
	switch status {
	case "sql", leadmodel.LeadStatusConverted:
		return true
	default:
		return false
	}
}

func isActiveStage(stage string) bool {
	switch cleanLower(stage) {
	case oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation:
		return true
	default:
		return false
	}
}

func isTerminalStage(stage string) bool {
	stage = cleanLower(stage)
	return stage == oppmodel.StageWon || stage == oppmodel.StageLost
}

func isValidStage(stage string) bool {
	return isActiveStage(stage) || isTerminalStage(stage)
}

func isValidStageType(stageType string) bool {
	switch cleanLower(stageType) {
	case oppmodel.StageTypeActive, oppmodel.StageTypeWon, oppmodel.StageTypeLost:
		return true
	default:
		return false
	}
}

func isValidContractStatus(status string) bool {
	switch cleanLower(status) {
	case oppmodel.ContractStatusDraft, oppmodel.ContractStatusPending, oppmodel.ContractStatusSigned, oppmodel.ContractStatusCancelled:
		return true
	default:
		return false
	}
}

func isValidPaymentStatus(status string) bool {
	switch cleanLower(status) {
	case oppmodel.PaymentStatusPlanned, oppmodel.PaymentStatusPaid, oppmodel.PaymentStatusOverdue, oppmodel.PaymentStatusVoided:
		return true
	default:
		return false
	}
}

func canAdvance(fromStage, toStage string) bool {
	order := map[string]int{
		oppmodel.StageOpen:        0,
		oppmodel.StageQualified:   1,
		oppmodel.StageProposal:    2,
		oppmodel.StageNegotiation: 3,
	}
	from, okFrom := order[cleanLower(fromStage)]
	to, okTo := order[cleanLower(toStage)]
	return okFrom && okTo && to >= from
}

func canAdvanceInPipeline(stages []*oppmodel.OpportunityStageConfig, fromUUID, toUUID string) bool {
	fromUUID = cleanLower(fromUUID)
	toUUID = cleanLower(toUUID)
	if fromUUID == "" || toUUID == "" {
		return false
	}
	fromIndex := -1
	toIndex := -1
	for index, stage := range stages {
		if stage == nil || cleanLower(stage.StageType) != oppmodel.StageTypeActive {
			continue
		}
		if cleanLower(stage.ConfigUUID) == fromUUID {
			fromIndex = index
		}
		if cleanLower(stage.ConfigUUID) == toUUID {
			toIndex = index
		}
	}
	return fromIndex >= 0 && toIndex >= 0 && toIndex >= fromIndex
}

func parseRiskFlags(raw datatypes.JSON) []string {
	var flags []string
	if len(raw) == 0 {
		return flags
	}
	_ = json.Unmarshal(raw, &flags)
	return flags
}

func jsonBytes(value any) datatypes.JSON {
	raw, _ := json.Marshal(value)
	return datatypes.JSON(raw)
}

func cleanLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func normalizeActor(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := uuid.Parse(value)
	if value == "" || err != nil || parsed == uuid.Nil {
		return "", ErrActorUserUUIDRequired
	}
	return strings.ToLower(parsed.String()), nil
}

func actorFromPipelineOptions(opts ...PipelineQueryRequest) string {
	for _, opt := range opts {
		actor, err := normalizeActor(opt.ActorUserUUID)
		if err == nil {
			return actor
		}
	}
	return ""
}

func dedupePipelineGroups(items []*oppmodel.OpportunityPipelineGroup) []*oppmodel.OpportunityPipelineGroup {
	if len(items) <= 1 {
		return items
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]*oppmodel.OpportunityPipelineGroup, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		key := pipelineGroupDedupeKey(item)
		if key == "" {
			out = append(out, item)
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func dedupePipelineGroupListItems(items []pipelineGroupListItem) []*oppmodel.OpportunityPipelineGroup {
	if len(items) == 0 {
		return nil
	}
	best := make(map[string]int, len(items))
	order := make([]string, 0, len(items))
	fallback := make([]*oppmodel.OpportunityPipelineGroup, 0)
	for i := range items {
		item := &items[i]
		key := pipelineGroupDedupeKey(&item.OpportunityPipelineGroup)
		if key == "" {
			group := item.OpportunityPipelineGroup
			fallback = append(fallback, &group)
			continue
		}
		if _, ok := best[key]; !ok {
			best[key] = i
			order = append(order, key)
			continue
		}
		current := &items[best[key]]
		if shouldPreferPipelineGroup(item, current) {
			best[key] = i
		}
	}
	out := make([]*oppmodel.OpportunityPipelineGroup, 0, len(order)+len(fallback))
	for _, key := range order {
		group := items[best[key]].OpportunityPipelineGroup
		out = append(out, &group)
	}
	out = append(out, fallback...)
	return out
}

func pipelineGroupDedupeKey(item *oppmodel.OpportunityPipelineGroup) string {
	if item == nil {
		return ""
	}
	groupKey := cleanLower(item.GroupKey)
	if groupKey != "" {
		return "key:" + groupKey
	}
	return "uuid:" + cleanLower(item.GroupUUID)
}

func shouldPreferPipelineGroup(candidate, current *pipelineGroupListItem) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	if candidate.IsDefault != current.IsDefault {
		return candidate.IsDefault
	}
	if candidate.IsActive != current.IsActive {
		return candidate.IsActive
	}
	if candidate.StageCount != current.StageCount {
		return candidate.StageCount > current.StageCount
	}
	if candidate.SortOrder != current.SortOrder {
		return candidate.SortOrder < current.SortOrder
	}
	return candidate.CreatedAt.Before(current.CreatedAt)
}

func defaultPipelineGroupTemplate(tenantUUID string) *oppmodel.OpportunityPipelineGroup {
	now := time.Now().UTC()
	return &oppmodel.OpportunityPipelineGroup{
		TenantUUID:  tenantUUID,
		GroupKey:    oppmodel.DefaultPipelineGroupKey,
		Name:        "默认销售流程",
		Description: "系统默认商机销售管道",
		IsDefault:   true,
		IsActive:    true,
		SortOrder:   10,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func defaultStageConfigTemplates(tenantUUID, groupUUID string) []*oppmodel.OpportunityStageConfig {
	return []*oppmodel.OpportunityStageConfig{
		{TenantUUID: tenantUUID, PipelineGroupUUID: groupUUID, StageKey: oppmodel.StageOpen, Label: "打开", SortOrder: 10, DefaultWinRate: 20, SLADays: 3, StageType: oppmodel.StageTypeActive, FixedStage: oppmodel.StageOpen, IsActive: true, MigrationPolicy: "map_to_fixed"},
		{TenantUUID: tenantUUID, PipelineGroupUUID: groupUUID, StageKey: oppmodel.StageQualified, Label: "已确认", SortOrder: 20, DefaultWinRate: 40, SLADays: 5, StageType: oppmodel.StageTypeActive, FixedStage: oppmodel.StageQualified, IsActive: true, MigrationPolicy: "map_to_fixed"},
		{TenantUUID: tenantUUID, PipelineGroupUUID: groupUUID, StageKey: oppmodel.StageProposal, Label: "方案", SortOrder: 30, DefaultWinRate: 60, SLADays: 7, StageType: oppmodel.StageTypeActive, FixedStage: oppmodel.StageProposal, IsActive: true, MigrationPolicy: "map_to_fixed"},
		{TenantUUID: tenantUUID, PipelineGroupUUID: groupUUID, StageKey: oppmodel.StageNegotiation, Label: "谈判", SortOrder: 40, DefaultWinRate: 80, SLADays: 10, StageType: oppmodel.StageTypeActive, FixedStage: oppmodel.StageNegotiation, IsActive: true, MigrationPolicy: "map_to_fixed"},
		{TenantUUID: tenantUUID, PipelineGroupUUID: groupUUID, StageKey: oppmodel.StageWon, Label: "赢单", SortOrder: 90, DefaultWinRate: 100, SLADays: 0, StageType: oppmodel.StageTypeWon, FixedStage: oppmodel.StageWon, IsActive: true, MigrationPolicy: "map_to_fixed"},
		{TenantUUID: tenantUUID, PipelineGroupUUID: groupUUID, StageKey: oppmodel.StageLost, Label: "输单", SortOrder: 100, DefaultWinRate: 0, SLADays: 0, StageType: oppmodel.StageTypeLost, FixedStage: oppmodel.StageLost, IsActive: true, MigrationPolicy: "map_to_fixed"},
	}
}

func (s *Service) pipelineTemplateStages(ctx context.Context, db *gorm.DB, tenantUUID, groupUUID, templateKey string) ([]*oppmodel.OpportunityStageConfig, error) {
	templateKey = cleanLower(templateKey)
	if templateKey == "" || templateKey == "default" || templateKey == "b2b_standard" {
		if stages, err := s.pipelineTemplateStagesFromDB(ctx, db, tenantUUID, groupUUID, "b2b_standard"); err == nil && len(stages) > 0 {
			return stages, nil
		}
		return defaultStageConfigTemplates(tenantUUID, groupUUID), nil
	}
	stages, err := s.pipelineTemplateStagesFromDB(ctx, db, tenantUUID, groupUUID, templateKey)
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return nil, ErrInvalidPayload
	}
	return stages, nil
}

func (s *Service) pipelineTemplateStagesFromDB(ctx context.Context, db *gorm.DB, tenantUUID, groupUUID, templateKey string) ([]*oppmodel.OpportunityStageConfig, error) {
	templateKey = cleanLower(templateKey)
	if db == nil || templateKey == "" {
		return nil, nil
	}
	var template oppmodel.OpportunityPipelineTemplate
	if err := db.WithContext(ctx).
		Where("template_key = ? AND is_active = ?", templateKey, true).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var rows []*oppmodel.OpportunityPipelineTemplateStage
	if err := db.WithContext(ctx).
		Where("template_key = ? AND is_active = ?", templateKey, true).
		Order("sort_order ASC, created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*oppmodel.OpportunityStageConfig, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, &oppmodel.OpportunityStageConfig{
			TenantUUID:        tenantUUID,
			PipelineGroupUUID: groupUUID,
			StageKey:          cleanLower(row.StageKey),
			Label:             strings.TrimSpace(row.Label),
			SortOrder:         row.SortOrder,
			DefaultWinRate:    row.DefaultWinRate,
			SLADays:           row.SLADays,
			StageType:         cleanLower(row.StageType),
			FixedStage:        cleanLower(row.FixedStage),
			IsActive:          row.IsActive,
			MigrationPolicy:   "map_to_fixed",
		})
	}
	return out, nil
}

func firstActiveStage(stages []*oppmodel.OpportunityStageConfig) *oppmodel.OpportunityStageConfig {
	for _, stage := range stages {
		if stage != nil && stage.IsActive && cleanLower(stage.StageType) == oppmodel.StageTypeActive {
			return stage
		}
	}
	return nil
}

func stageByFixed(stages []*oppmodel.OpportunityStageConfig, fixedStage string) *oppmodel.OpportunityStageConfig {
	fixedStage = cleanLower(fixedStage)
	for _, stage := range stages {
		if stage != nil && stage.IsActive && cleanLower(stage.FixedStage) == fixedStage {
			return stage
		}
	}
	return nil
}

func duplicateScore(a, b *oppmodel.OpportunityRecord) (int, string) {
	if a == nil || b == nil {
		return 0, ""
	}
	score := 0
	reasons := make([]string, 0, 4)
	if a.LeadUUID != "" && a.LeadUUID == b.LeadUUID {
		score += 80
		reasons = append(reasons, "同一线索")
	}
	if a.ExternalUserID != "" && a.ExternalUserID == b.ExternalUserID {
		score += 70
		reasons = append(reasons, "同一外部联系人")
	}
	if a.SourceChannel != "" && a.SourceChannel == b.SourceChannel && a.SourceAccountUUID != nil && b.SourceAccountUUID != nil && *a.SourceAccountUUID == *b.SourceAccountUUID {
		score += 50
		reasons = append(reasons, "同一来源账号")
	}
	if a.OwnerUserUUID != "" && a.OwnerUserUUID == b.OwnerUserUUID {
		score += 15
		reasons = append(reasons, "同一负责人")
	}
	if titleSimilarity(a.Title, b.Title) >= 0.6 {
		score += 30
		reasons = append(reasons, "标题相似")
	}
	if score > 100 {
		score = 100
	}
	return score, strings.Join(reasons, "、")
}

func titleSimilarity(a, b string) float64 {
	left := []rune(strings.ToLower(strings.TrimSpace(a)))
	right := []rune(strings.ToLower(strings.TrimSpace(b)))
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	set := map[rune]struct{}{}
	for _, r := range left {
		if r != ' ' {
			set[r] = struct{}{}
		}
	}
	matches := 0
	for _, r := range right {
		if _, ok := set[r]; ok {
			matches++
		}
	}
	shorter := len(left)
	if len(right) < shorter {
		shorter = len(right)
	}
	return float64(matches) / float64(shorter)
}

var _ = isValidStage

func quoteStorageRoot() string {
	if value := strings.TrimSpace(os.Getenv("SCRM_QUOTE_STORAGE_ROOT")); value != "" {
		return value
	}
	return defaultQuoteStorageRoot
}

func safeFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == string(os.PathSeparator) {
		return "quote-file"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	if len(name) > 160 {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(ext) > 24 {
			ext = ext[:24]
		}
		if len(base) > 136 {
			base = base[:136]
		}
		name = base + ext
	}
	return name
}

func hydrateLineItemDownloadURL(item *oppmodel.OpportunityLineItem) {
	if item == nil || strings.TrimSpace(item.StorageProvider) == "" || strings.TrimSpace(item.ObjectKey) == "" {
		return
	}
	if strings.TrimSpace(item.StorageProvider) != "local" {
		return
	}
	item.DownloadURL = fmt.Sprintf("/api/v1/admin/opportunity/records/%s/line-items/%s/download", item.OpportunityUUID, item.ItemUUID)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
