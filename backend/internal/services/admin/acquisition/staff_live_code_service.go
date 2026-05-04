package acquisition

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	work "github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	pwcontactwayreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/contactWay/request"
	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	"github.com/google/uuid"
)

var (
	ErrStaffLiveCodeServiceNotReady   = errors.New("staff live code service not ready")
	ErrInvalidStaffLiveCodePayload    = errors.New("invalid staff live code payload")
	ErrStaffLiveCodeStatusInvalid     = errors.New("invalid staff live code status")
	ErrStaffLiveCodeAlreadyExists     = errors.New("staff live code already exists")
	ErrStaffMemberBindingNotConfirmed = errors.New("member binding is not confirmed")
	// 保留旧错误名兼容既有测试/调用方，后续统一迁移到 ErrStaffMemberBindingNotConfirmed。
	ErrStaffMemberMappingNotConfirmed = ErrStaffMemberBindingNotConfirmed
	ErrDefaultChannelAccountNotFound  = errors.New("default channel account not found")
)

var codeKeySanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type StaffDefaultAccountResolver interface {
	ResolveDefaultChannelAccount(ctx context.Context, tenantUUID, channel, appType string) (string, error)
	GetChannelAccount(ctx context.Context, tenantUUID, accountUUID string) (*GroupChatAccountProfile, error)
	GetChannelAccountCredentials(ctx context.Context, tenantUUID, accountUUID string) (map[string]string, error)
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

type StaffLiveCodeUpdateRequest struct {
	TenantUUID              string
	StaffCodeUUID           string
	ActivityName            string
	MemberUUIDs             []string
	CorpTagIDs              []string
	NewCustomerRemarkEnable *bool
	ActorUserUUID           string
}

type StaffLiveCodeService struct {
	repo               acqrepo.StaffLiveCodeRepository
	welcomeConfigRepo  acqrepo.StaffWelcomeConfigRepository
	welcomeAttemptRepo acqrepo.StaffWelcomeSyncAttemptRepository
	accountResolver    StaffDefaultAccountResolver
	providerFactory    *GroupChatProviderFactory
}

func NewStaffLiveCodeService(repo acqrepo.StaffLiveCodeRepository, resolver ...StaffDefaultAccountResolver) *StaffLiveCodeService {
	factory := NewGroupChatProviderFactory()
	_ = factory.Register("wechat", "wecom", buildWeComSelfBuiltWorkApp)
	_ = factory.Register("wechat", "openwork", buildWeComOpenWorkApp)
	svc := &StaffLiveCodeService{repo: repo, providerFactory: factory}
	if len(resolver) > 0 {
		svc.accountResolver = resolver[0]
	}
	return svc
}

func (s *StaffLiveCodeService) WithWelcomeRepos(configRepo acqrepo.StaffWelcomeConfigRepository, attemptRepo acqrepo.StaffWelcomeSyncAttemptRepository) *StaffLiveCodeService {
	if s == nil {
		return s
	}
	s.welcomeConfigRepo = configRepo
	s.welcomeAttemptRepo = attemptRepo
	return s
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
	if totalConfirmed != int64(len(uniqueLowerStrings(req.MemberUUIDs))) {
		return nil, ErrStaffMemberBindingNotConfirmed
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
			MemberUUIDs:        uniqueLowerStrings(req.MemberUUIDs),
			CorpTagIDs:         uniqueTrimmedStrings(req.CorpTagIDs),
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
	if err := s.syncRemoteContactWay(ctx, item); err != nil {
		return nil, err
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

func (s *StaffLiveCodeService) Update(ctx context.Context, req StaffLiveCodeUpdateRequest) (*acqmodel.StaffLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrStaffLiveCodeServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.StaffCodeUUID = strings.ToLower(strings.TrimSpace(req.StaffCodeUUID))
	req.ActivityName = strings.TrimSpace(req.ActivityName)
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.StaffCodeUUID == "" {
		return nil, ErrInvalidStaffLiveCodePayload
	}
	item, err := s.repo.GetByUUID(ctx, req.TenantUUID, req.StaffCodeUUID)
	if err != nil {
		return nil, err
	}
	if req.ActivityName != "" {
		item.ActivityName = req.ActivityName
	}
	if req.MemberUUIDs != nil && len(req.MemberUUIDs) > 0 {
		totalConfirmed, confirmErr := s.repo.CountConfirmedMappings(ctx, req.TenantUUID, req.MemberUUIDs)
		if confirmErr != nil {
			return nil, confirmErr
		}
		if totalConfirmed != int64(len(uniqueLowerStrings(req.MemberUUIDs))) {
			return nil, ErrStaffMemberBindingNotConfirmed
		}
		item.MemberUUIDs = uniqueLowerStrings(req.MemberUUIDs)
	}
	if req.CorpTagIDs != nil {
		item.CorpTagIDs = uniqueTrimmedStrings(req.CorpTagIDs)
	}
	if req.NewCustomerRemarkEnable != nil {
		item.RemarkEnabled = *req.NewCustomerRemarkEnable
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = "system"
	}
	item.UpdatedBy = req.ActorUserUUID
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	if err := s.syncRemoteContactWay(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByUUID(ctx, req.TenantUUID, req.StaffCodeUUID)
}

type staffLiveCodeRuntime struct {
	tenantUUID  string
	accountUUID string
	channelCode string
	appType     string
}

func (s *StaffLiveCodeService) resolveRuntime(ctx context.Context, item *acqmodel.StaffLiveCode) (*staffLiveCodeRuntime, error) {
	if s == nil || s.accountResolver == nil {
		return nil, ErrStaffLiveCodeServiceNotReady
	}
	if item == nil {
		return nil, ErrInvalidStaffLiveCodePayload
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(item.TenantUUID))
	accountUUID := strings.ToLower(strings.TrimSpace(item.ChannelAccountUUID))
	if tenantUUID == "" {
		return nil, ErrInvalidStaffLiveCodePayload
	}
	if accountUUID == "" {
		resolved, err := s.accountResolver.ResolveDefaultChannelAccount(ctx, tenantUUID, strings.TrimSpace(item.Channel), strings.TrimSpace(item.AppType))
		if err != nil || strings.TrimSpace(resolved) == "" {
			return nil, ErrDefaultChannelAccountNotFound
		}
		accountUUID = strings.ToLower(strings.TrimSpace(resolved))
		item.ChannelAccountUUID = accountUUID
	}
	profile, err := s.accountResolver.GetChannelAccount(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrDefaultChannelAccountNotFound
	}
	channelCode := strings.ToLower(strings.TrimSpace(profile.ChannelCode))
	appType := strings.ToLower(strings.TrimSpace(profile.AppType))
	if channelCode != "wechat" {
		return nil, fmt.Errorf("%w: channel=%s app_type=%s", ErrUnsupportedChannelAccountType, channelCode, appType)
	}
	if appType != "wecom" && appType != "openwork" {
		return nil, fmt.Errorf("%w: channel=%s app_type=%s", ErrUnsupportedChannelAccountType, channelCode, appType)
	}
	return &staffLiveCodeRuntime{
		tenantUUID:  tenantUUID,
		accountUUID: accountUUID,
		channelCode: channelCode,
		appType:     appType,
	}, nil
}

func (s *StaffLiveCodeService) buildProvider(ctx context.Context, rt *staffLiveCodeRuntime) (*work.Work, error) {
	if s == nil || s.accountResolver == nil || s.providerFactory == nil || rt == nil {
		return nil, ErrStaffLiveCodeServiceNotReady
	}
	credentials, err := s.accountResolver.GetChannelAccountCredentials(ctx, rt.tenantUUID, rt.accountUUID)
	if err != nil {
		return nil, err
	}
	return s.providerFactory.Build(rt.channelCode, rt.appType, credentials)
}

func (s *StaffLiveCodeService) syncRemoteContactWay(ctx context.Context, item *acqmodel.StaffLiveCode) error {
	if s == nil || s.repo == nil || item == nil {
		return ErrStaffLiveCodeServiceNotReady
	}
	if s.accountResolver == nil || s.providerFactory == nil {
		return nil
	}
	rt, err := s.resolveRuntime(ctx, item)
	if err != nil {
		return err
	}
	app, err := s.buildProvider(ctx, rt)
	if err != nil {
		return err
	}
	if app == nil || app.ExternalContactContactWay == nil {
		return errors.New("wecom contact way client unavailable")
	}
	userIDs, err := s.repo.ResolveExternalMemberIDs(ctx, rt.tenantUUID, rt.accountUUID, item.MemberUUIDs)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return ErrStaffMemberBindingNotConfirmed
	}
	state := strings.TrimSpace(item.State)
	if state == "" {
		state = "st-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
		item.State = state
	}
	remark := strings.TrimSpace(item.ActivityName)
	if remark == "" {
		remark = "员工活码"
	}
	if strings.TrimSpace(item.ConfigID) == "" {
		contactWayType := 1
		if len(userIDs) > 1 {
			contactWayType = 2
		}
		addRes, addErr := app.ExternalContactContactWay.Add(ctx, &pwcontactwayreq.RequestAddContactWay{
			Type:       contactWayType,
			Scene:      2,
			Style:      1,
			Remark:     remark,
			SkipVerify: true,
			State:      state,
			User:       userIDs,
		})
		if addErr != nil {
			return fmt.Errorf("wecom add_contact_way failed: %w", addErr)
		}
		if addRes == nil {
			return errors.New("wecom add_contact_way failed: empty response")
		}
		if addRes.ErrCode != 0 {
			return fmt.Errorf("wecom add_contact_way failed: %d %s", addRes.ErrCode, strings.TrimSpace(addRes.ErrMsg))
		}
		item.ConfigID = strings.TrimSpace(addRes.ConfigID)
		item.QRCode = strings.TrimSpace(addRes.QRCode)
		if item.ConfigID == "" {
			return errors.New("wecom add_contact_way failed: empty config_id")
		}
	} else {
		updateRes, updateErr := app.ExternalContactContactWay.Update(ctx, &pwcontactwayreq.RequestUpdateContactWay{
			ConfigID:   item.ConfigID,
			Remark:     remark,
			Style:      1,
			SkipVerify: true,
			State:      state,
			User:       userIDs,
		})
		if updateErr != nil {
			return fmt.Errorf("wecom update_contact_way failed: %w", updateErr)
		}
		if updateRes == nil {
			return errors.New("wecom update_contact_way failed: empty response")
		}
		if updateRes.ErrCode != 0 {
			return fmt.Errorf("wecom update_contact_way failed: %d %s", updateRes.ErrCode, strings.TrimSpace(updateRes.ErrMsg))
		}
	}
	getRes, getErr := app.ExternalContactContactWay.Get(ctx, item.ConfigID)
	if getErr != nil {
		return fmt.Errorf("wecom get_contact_way failed: %w", getErr)
	}
	if getRes == nil {
		return errors.New("wecom get_contact_way failed: empty response")
	}
	if getRes.ErrCode != 0 {
		return fmt.Errorf("wecom get_contact_way failed: %d %s", getRes.ErrCode, strings.TrimSpace(getRes.ErrMsg))
	}
	if getRes.ContactWay != nil && strings.TrimSpace(getRes.ContactWay.QrCode) != "" {
		item.QRCode = strings.TrimSpace(getRes.ContactWay.QrCode)
	}
	if strings.TrimSpace(item.QRCode) == "" {
		return errors.New("wecom get_contact_way failed: empty qr_code")
	}
	if strings.ToLower(strings.TrimSpace(item.Status)) == acqmodel.LiveCodeStatusDraft {
		item.Status = acqmodel.LiveCodeStatusActive
	}
	return s.repo.Update(ctx, item)
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

func (s *StaffLiveCodeService) Delete(ctx context.Context, tenantUUID, staffCodeUUID string) error {
	if s == nil || s.repo == nil {
		return ErrStaffLiveCodeServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	staffCodeUUID = strings.ToLower(strings.TrimSpace(staffCodeUUID))
	if tenantUUID == "" || staffCodeUUID == "" {
		return ErrInvalidStaffLiveCodePayload
	}
	item, err := s.repo.GetByUUID(ctx, tenantUUID, staffCodeUUID)
	if err != nil {
		return err
	}
	if err := s.deleteRemoteContactWay(ctx, item); err != nil {
		return err
	}
	if s.welcomeAttemptRepo != nil {
		if delErr := s.welcomeAttemptRepo.DeleteByStaffCodeUUID(ctx, tenantUUID, staffCodeUUID); delErr != nil {
			return delErr
		}
	}
	if s.welcomeConfigRepo != nil {
		if delErr := s.welcomeConfigRepo.DeleteByStaffCodeUUID(ctx, tenantUUID, staffCodeUUID); delErr != nil {
			return delErr
		}
	}
	return s.repo.Delete(ctx, tenantUUID, staffCodeUUID)
}

func (s *StaffLiveCodeService) deleteRemoteContactWay(ctx context.Context, item *acqmodel.StaffLiveCode) error {
	if s == nil || item == nil {
		return ErrStaffLiveCodeServiceNotReady
	}
	configID := strings.TrimSpace(item.ConfigID)
	if configID == "" {
		return nil
	}
	if s.accountResolver == nil || s.providerFactory == nil {
		return errors.New("cannot delete remote contact_way: account resolver unavailable")
	}
	rt, err := s.resolveRuntime(ctx, item)
	if err != nil {
		return err
	}
	app, err := s.buildProvider(ctx, rt)
	if err != nil {
		return err
	}
	if app == nil || app.ExternalContactContactWay == nil {
		return errors.New("wecom contact way client unavailable")
	}
	resp, err := app.ExternalContactContactWay.Delete(ctx, configID)
	if err != nil {
		return fmt.Errorf("wecom del_contact_way failed: %w", err)
	}
	if resp == nil {
		return errors.New("wecom del_contact_way failed: empty response")
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("wecom del_contact_way failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	return nil
}

func uniqueLowerStrings(values []string) []string {
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

func uniqueTrimmedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		clean := strings.TrimSpace(value)
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
