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
	LeadUUID        string
	Title           string
	OwnerUserUUID   string
	Amount          *float64
	Currency        string
	Probability     *int
	ExpectedCloseAt *time.Time
	ActorUserUUID   string
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
			Stage:             oppmodel.StageOpen,
			Amount:            req.Amount,
			Currency:          currency,
			Probability:       probability,
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
		if !canAdvance(fromStage, toStage) {
			return ErrInvalidStageTransition
		}
		item.Stage = toStage
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
		item.Stage = result
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
		item.Stage = oppmodel.StageOpen
		item.WonAt = nil
		item.LostAt = nil
		item.LostReason = ""
		item.UpdatedBy = actor
		item.UpdatedAt = time.Now().UTC()
		if err := updateOpportunityTx(ctx, tx, item).Error; err != nil {
			return err
		}
		return createActivityTx(ctx, tx, tenantUUID, item.OpportunityUUID, oppmodel.ActivityReopen, fromStage, oppmodel.StageOpen, actor, map[string]any{})
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

func updateOpportunityTx(ctx context.Context, tx *gorm.DB, item *oppmodel.OpportunityRecord) *gorm.DB {
	return tx.WithContext(ctx).Model(&oppmodel.OpportunityRecord{}).
		Where("tenant_uuid = ? AND opportunity_uuid = ?", item.TenantUUID, item.OpportunityUUID).
		Updates(map[string]any{
			"title":             item.Title,
			"stage":             item.Stage,
			"amount":            item.Amount,
			"currency":          item.Currency,
			"probability":       item.Probability,
			"owner_user_uuid":   item.OwnerUserUUID,
			"expected_close_at": item.ExpectedCloseAt,
			"won_at":            item.WonAt,
			"lost_at":           item.LostAt,
			"lost_reason":       item.LostReason,
			"risk_flags":        item.RiskFlags,
			"updated_by":        item.UpdatedBy,
			"updated_at":        item.UpdatedAt,
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
	case leadmodel.LeadStatusClosed, leadmodel.LeadStatusDisconnected:
		return false
	default:
		return status != ""
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

func normalizeActor(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := uuid.Parse(value)
	if value == "" || err != nil || parsed == uuid.Nil {
		return "", ErrActorUserUUIDRequired
	}
	return strings.ToLower(parsed.String()), nil
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
