package lead_capture

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	domainmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	entitymodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrAttributionServiceNotReady = errors.New("attribution service not ready")
)

type AttributionService struct {
	attributionRepo domainrepo.LeadAttributionRepository
	leadRepo        *leadrepo.LeadRepository
	leadService     *LeadService
}

func NewAttributionService(
	attributionRepo domainrepo.LeadAttributionRepository,
	leadRepo *leadrepo.LeadRepository,
	leadService *LeadService,
) *AttributionService {
	if leadService == nil && leadRepo != nil {
		leadService = NewLeadService(leadRepo)
	}
	return &AttributionService{
		attributionRepo: attributionRepo,
		leadRepo:        leadRepo,
		leadService:     leadService,
	}
}

func (s *AttributionService) AttributeByEvent(
	ctx context.Context,
	event *domainmodel.ChannelCodeEvent,
	code *domainmodel.ChannelCode,
) (leadUUID string, isPrimary bool, err error) {
	if s == nil || s.attributionRepo == nil || s.leadRepo == nil || s.leadService == nil {
		return "", false, ErrAttributionServiceNotReady
	}
	if event == nil || code == nil {
		return "", false, errors.New("event and code are required")
	}
	lead, err := s.resolveLead(ctx, event, code)
	if err != nil {
		return "", false, err
	}
	existingByEvent, err := s.attributionRepo.ListByEventUUID(ctx, event.TenantUUID, event.EventUUID)
	if err != nil {
		return "", false, err
	}
	for _, item := range existingByEvent {
		if item != nil && strings.EqualFold(strings.TrimSpace(item.LeadUUID), lead.LeadUUID) {
			return lead.LeadUUID, item.IsPrimary, nil
		}
	}
	previous, err := s.attributionRepo.ListByLeadUUID(ctx, event.TenantUUID, lead.LeadUUID)
	if err != nil {
		return "", false, err
	}
	isPrimary = true
	attributionType := domainmodel.AttributionTypeFirstTouch
	for _, item := range previous {
		if item != nil && item.IsPrimary {
			isPrimary = false
			attributionType = domainmodel.AttributionTypeFollowTouch
			break
		}
	}
	record := &domainmodel.LeadAttributionRecord{
		AttributionUUID: uuid.NewString(),
		TenantUUID:      event.TenantUUID,
		LeadUUID:        lead.LeadUUID,
		CodeUUID:        event.CodeUUID,
		EventUUID:       event.EventUUID,
		IsPrimary:       isPrimary,
		AttributionType: attributionType,
	}
	if err := s.attributionRepo.Create(ctx, record); err != nil {
		return "", false, err
	}
	return lead.LeadUUID, isPrimary, nil
}

func (s *AttributionService) ListByEventUUID(ctx context.Context, tenantUUID, eventUUID string) ([]*domainmodel.LeadAttributionRecord, error) {
	if s == nil || s.attributionRepo == nil {
		return nil, ErrAttributionServiceNotReady
	}
	return s.attributionRepo.ListByEventUUID(ctx, tenantUUID, eventUUID)
}

func (s *AttributionService) CountByCodeUUID(ctx context.Context, tenantUUID, codeUUID string) (int64, error) {
	if s == nil || s.attributionRepo == nil {
		return 0, ErrAttributionServiceNotReady
	}
	return s.attributionRepo.CountByCodeUUID(ctx, tenantUUID, codeUUID)
}

func (s *AttributionService) resolveLead(
	ctx context.Context,
	event *domainmodel.ChannelCodeEvent,
	code *domainmodel.ChannelCode,
) (*entitymodel.Lead, error) {
	if leadUUID := payloadJSONString(event.Payload, "lead_uuid"); leadUUID != "" {
		lead, err := s.leadRepo.GetByUUID(ctx, event.TenantUUID, leadUUID)
		if err == nil {
			return lead, nil
		}
	}
	phone := payloadJSONString(event.Payload, "phone")
	email := payloadJSONString(event.Payload, "email")
	if lead, err := s.findLeadByContact(ctx, event.TenantUUID, phone, email); err == nil {
		return lead, nil
	}
	created, err := s.leadService.Create(ctx, event.TenantUUID, LeadCreateRequest{
		DisplayName:       firstNonEmpty(payloadJSONString(event.Payload, "display_name"), payloadJSONString(event.Payload, "name"), "渠道线索"),
		Phone:             phone,
		Email:             email,
		SourceChannel:     code.Channel,
		SourceAppType:     code.AppType,
		SourceAccountUUID: code.ChannelAccountUUID,
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *AttributionService) findLeadByContact(ctx context.Context, tenantUUID, phone, email string) (*entitymodel.Lead, error) {
	if s == nil || s.leadRepo == nil || s.leadRepo.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	phone = strings.TrimSpace(phone)
	email = strings.TrimSpace(email)
	if phone == "" && email == "" {
		return nil, gorm.ErrRecordNotFound
	}
	q := s.leadRepo.DB.WithContext(ctx).Where("tenant_uuid = ?", strings.ToLower(strings.TrimSpace(tenantUUID)))
	if phone != "" {
		q = q.Where("phone = ?", phone)
	} else {
		q = q.Where("email = ?", strings.ToLower(email))
	}
	var out entitymodel.Lead
	if err := q.Order("created_at asc").First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func payloadJSONString(payload []byte, key string) string {
	key = strings.TrimSpace(key)
	if len(payload) == 0 || key == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return ""
	}
	raw, ok := obj[key]
	if !ok {
		return ""
	}
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
