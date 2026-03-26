package acquisition

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	"github.com/google/uuid"
)

var (
	ErrStaffLiveCodeServiceNotReady   = errors.New("staff live code service not ready")
	ErrInvalidStaffLiveCodePayload    = errors.New("invalid staff live code payload")
	ErrStaffLiveCodeStatusInvalid     = errors.New("invalid staff live code status")
	ErrStaffLiveCodeAlreadyExists     = errors.New("staff live code already exists")
	ErrStaffMemberMappingNotConfirmed = errors.New("member mapping is not confirmed")
	ErrDefaultChannelAccountNotFound  = errors.New("default channel account not found")
)

var codeKeySanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type StaffDefaultAccountResolver interface {
	ResolveDefaultChannelAccount(ctx context.Context, tenantUUID, channel, appType string) (string, error)
}

type StaffLiveCodeCreateRequest struct {
	TenantUUID              string
	Channel                 string
	AppType                 string
	ChannelAccountUUID      string
	ActivityName            string
	CodeKey                 string
	MemberUUIDs             []string
	CorpTagIDs              []string
	NewCustomerRemarkEnable bool
	ActorUserUUID           string
}

type StaffLiveCodeListRequest struct {
	TenantUUID   string
	ActivityName string
	Status       string
	Limit        int
}

type StaffLiveCodeStatusUpdateRequest struct {
	TenantUUID    string
	StaffCodeUUID string
	Status        string
	ActorUserUUID string
}

type StaffLiveCodeService struct {
	repo            acqrepo.StaffLiveCodeRepository
	accountResolver StaffDefaultAccountResolver
}

func NewStaffLiveCodeService(repo acqrepo.StaffLiveCodeRepository, resolver ...StaffDefaultAccountResolver) *StaffLiveCodeService {
	svc := &StaffLiveCodeService{repo: repo}
	if len(resolver) > 0 {
		svc.accountResolver = resolver[0]
	}
	return svc
}

func (s *StaffLiveCodeService) Create(ctx context.Context, req StaffLiveCodeCreateRequest) (*acqmodel.StaffLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrStaffLiveCodeServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.AppType = strings.ToLower(strings.TrimSpace(req.AppType))
	req.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(req.ChannelAccountUUID))
	req.ActivityName = strings.TrimSpace(req.ActivityName)
	req.CodeKey = strings.TrimSpace(req.CodeKey)
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.Channel == "" || req.AppType == "" || req.ActivityName == "" || len(req.MemberUUIDs) == 0 {
		return nil, ErrInvalidStaffLiveCodePayload
	}
	if req.ChannelAccountUUID == "" {
		if s.accountResolver == nil {
			return nil, ErrDefaultChannelAccountNotFound
		}
		accountUUID, err := s.accountResolver.ResolveDefaultChannelAccount(ctx, req.TenantUUID, req.Channel, req.AppType)
		if err != nil || strings.TrimSpace(accountUUID) == "" {
			return nil, ErrDefaultChannelAccountNotFound
		}
		req.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	}
	autoCodeKey := false
	if req.CodeKey == "" {
		autoCodeKey = true
		req.CodeKey = generateCodeKey(req.ActivityName)
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = "system"
	}

	totalConfirmed, err := s.repo.CountConfirmedMappings(ctx, req.TenantUUID, req.MemberUUIDs)
	if err != nil {
		return nil, err
	}
	if totalConfirmed != int64(len(uniqueStrings(req.MemberUUIDs))) {
		return nil, ErrStaffMemberMappingNotConfirmed
	}

	buildItem := func(codeKey string) *acqmodel.StaffLiveCode {
		return &acqmodel.StaffLiveCode{
			StaffCodeUUID:      uuid.NewString(),
			TenantUUID:         req.TenantUUID,
			Channel:            req.Channel,
			AppType:            req.AppType,
			ChannelAccountUUID: req.ChannelAccountUUID,
			ActivityName:       req.ActivityName,
			CodeKey:            codeKey,
			MemberUUIDs:        uniqueStrings(req.MemberUUIDs),
			CorpTagIDs:         uniqueStrings(req.CorpTagIDs),
			RemarkEnabled:      req.NewCustomerRemarkEnable,
			Status:             acqmodel.LiveCodeStatusDraft,
			CreatedBy:          req.ActorUserUUID,
			UpdatedBy:          req.ActorUserUUID,
		}
	}
	item := buildItem(req.CodeKey)
	createErr := s.repo.Create(ctx, item)
	if createErr != nil && autoCodeKey && isUniqueConflict(createErr) {
		// Auto-generated key may collide under high concurrency; regenerate and retry.
		item = buildItem(generateCodeKey(req.ActivityName))
		createErr = s.repo.Create(ctx, item)
	}
	if createErr != nil {
		if isUniqueConflict(createErr) {
			return nil, ErrStaffLiveCodeAlreadyExists
		}
		return nil, createErr
	}
	return item, nil
}

func generateCodeKey(activityName string) string {
	base := strings.ToLower(strings.TrimSpace(activityName))
	base = strings.ReplaceAll(base, " ", "-")
	base = codeKeySanitizer.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "staff-code"
	}
	if len(base) > 48 {
		base = base[:48]
		base = strings.Trim(base, "-")
		if base == "" {
			base = "staff-code"
		}
	}
	return fmt.Sprintf("%s-%d-%s", base, time.Now().Unix()%1000000, uuid.NewString()[:6])
}

func isUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "uq_acq_staff_codes_tenant_code_key") || strings.Contains(lower, "unique")
}

func (s *StaffLiveCodeService) List(ctx context.Context, req StaffLiveCodeListRequest) ([]*acqmodel.StaffLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrStaffLiveCodeServiceNotReady
	}
	return s.repo.List(ctx, req.TenantUUID, acqrepo.StaffLiveCodeListFilter{
		ActivityName: req.ActivityName,
		Status:       req.Status,
		Limit:        req.Limit,
	})
}

func (s *StaffLiveCodeService) UpdateStatus(ctx context.Context, req StaffLiveCodeStatusUpdateRequest) (*acqmodel.StaffLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrStaffLiveCodeServiceNotReady
	}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if req.Status != acqmodel.LiveCodeStatusActive && req.Status != acqmodel.LiveCodeStatusDisabled {
		return nil, ErrStaffLiveCodeStatusInvalid
	}
	actor := strings.TrimSpace(req.ActorUserUUID)
	if actor == "" {
		actor = "system"
	}
	return s.repo.UpdateStatus(ctx, req.TenantUUID, req.StaffCodeUUID, req.Status, actor)
}

func (s *StaffLiveCodeService) IsCodeKeyAvailable(ctx context.Context, tenantUUID, codeKey string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrStaffLiveCodeServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	codeKey = strings.TrimSpace(codeKey)
	if tenantUUID == "" || codeKey == "" {
		return false, ErrInvalidStaffLiveCodePayload
	}
	exists, err := s.repo.ExistsByCodeKey(ctx, tenantUUID, codeKey)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		clean := strings.ToLower(strings.TrimSpace(value))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}
