package social_channel_governance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"gorm.io/datatypes"
)

const wecomGetSuiteTokenEndpoint = "https://qyapi.weixin.qq.com/cgi-bin/service/get_suite_token"

var (
	ErrWeComTemplateNotFound      = errors.New("wecom template not found")
	ErrWeComTemplateAlreadyExists = errors.New("wecom template already exists")
)

type ChannelPlatformSettingService struct {
	repo *repository.ChannelPlatformSettingRepository
}

func NewChannelPlatformSettingService(repo *repository.ChannelPlatformSettingRepository) *ChannelPlatformSettingService {
	return &ChannelPlatformSettingService{repo: repo}
}

type WeComOpenWorkTemplate struct {
	TemplateID              string
	TemplateSecret          string
	TemplateTicket          string
	TemplateTicketUpdatedAt string
	TemplateTicketSource    string
	ProviderCorpID          string
	ProviderSecret          string
	IsDefault               bool
}

type WeComOpenWorkPlatformConfig struct {
	Enabled                 bool
	TemplateID              string
	TemplateSecret          string
	TemplateTicket          string
	TemplateTicketUpdatedAt string
	TemplateTicketSource    string
	ProviderCorpID          string
	ProviderSecret          string
	Token                   string
	AESKey                  string
	HTTPDebug               bool
	CallbackHost            string
	RedirectURI             string
	DefaultTemplateID       string
	Templates               []WeComOpenWorkTemplate
}

type WeComSuiteTicketStatus struct {
	Ready                bool
	TemplateID           string
	TemplateTicket       string
	TemplateTicketMasked string
	TemplateTicketSource string
	UpdatedAt            string
	CheckedAt            string
	Message              string
}

type WeComSuiteTicketVerifyResult struct {
	Valid               bool
	TemplateID          string
	CheckedAt           string
	ErrCode             int
	ErrMsg              string
	ExpiresIn           int
	HasSuiteAccessToken bool
	Message             string
}

func (s *ChannelPlatformSettingService) GetWeComOpenWorkConfig(ctx context.Context) (*WeComOpenWorkPlatformConfig, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("channel platform setting service unavailable")
	}
	record, err := s.repo.GetByChannelProvider(ctx, "wechat", "openwork")
	if err != nil {
		if errors.Is(err, repository.ErrChannelPlatformSettingNotFound) {
			return &WeComOpenWorkPlatformConfig{Enabled: true}, nil
		}
		return nil, err
	}
	return parseWeComOpenWorkConfig(record.Enabled, record.Config), nil
}

func (s *ChannelPlatformSettingService) SaveWeComOpenWorkConfig(ctx context.Context, in WeComOpenWorkPlatformConfig) (*WeComOpenWorkPlatformConfig, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("channel platform setting service unavailable")
	}
	in = sanitizeWeComOpenWorkConfig(in)
	existingRecord, err := s.repo.GetByChannelProvider(ctx, "wechat", "openwork")
	if err != nil && !errors.Is(err, repository.ErrChannelPlatformSettingNotFound) {
		return nil, err
	}

	var existingCfg *WeComOpenWorkPlatformConfig
	if existingRecord != nil {
		existingCfg = parseWeComOpenWorkConfig(existingRecord.Enabled, existingRecord.Config)
	}

	hasTemplateID := in.TemplateID != ""
	hasTemplateSecret := in.TemplateSecret != ""
	if hasTemplateID != hasTemplateSecret {
		return nil, errors.New("template_id 和 template_secret 需要同时填写")
	}
	if hasTemplateID && !isValidOpenWorkTemplateID(in.TemplateID) {
		return nil, errors.New("template_id 格式非法，仅允许字母/数字/_/-")
	}

	if len(in.Templates) == 0 && existingCfg != nil && in.TemplateID != "" {
		in.Templates = cloneTemplateList(existingCfg.Templates)
	}
	if in.DefaultTemplateID == "" && existingCfg != nil && in.TemplateID != "" {
		in.DefaultTemplateID = existingCfg.DefaultTemplateID
	}

	if in.TemplateID != "" {
		idx := indexOfTemplate(in.Templates, in.TemplateID)
		if idx < 0 {
			in.Templates = append(in.Templates, WeComOpenWorkTemplate{TemplateID: in.TemplateID})
			idx = len(in.Templates) - 1
		}
		tpl := in.Templates[idx]
		tpl.TemplateID = in.TemplateID
		tpl.TemplateSecret = in.TemplateSecret
		tpl.ProviderCorpID = in.ProviderCorpID
		tpl.ProviderSecret = in.ProviderSecret
		if in.TemplateTicket != "" {
			tpl.TemplateTicket = in.TemplateTicket
		}
		if in.TemplateTicketUpdatedAt != "" {
			tpl.TemplateTicketUpdatedAt = in.TemplateTicketUpdatedAt
		}
		if in.TemplateTicketSource != "" {
			tpl.TemplateTicketSource = in.TemplateTicketSource
		}
		in.Templates[idx] = sanitizeWeComTemplate(tpl)
	}

	in.Templates = normalizeTemplateList(in.Templates)
	for _, tpl := range in.Templates {
		if !isValidOpenWorkTemplateID(tpl.TemplateID) {
			return nil, fmt.Errorf("template_id 格式非法: %s（仅允许字母/数字/_/-）", tpl.TemplateID)
		}
	}
	if in.DefaultTemplateID == "" {
		if in.TemplateID != "" {
			in.DefaultTemplateID = in.TemplateID
		} else if len(in.Templates) > 0 {
			in.DefaultTemplateID = in.Templates[0].TemplateID
		}
	}
	if in.DefaultTemplateID != "" && indexOfTemplate(in.Templates, in.DefaultTemplateID) < 0 {
		if len(in.Templates) > 0 {
			in.DefaultTemplateID = in.Templates[0].TemplateID
		} else {
			in.DefaultTemplateID = ""
		}
	}
	in.Templates = applyDefaultTemplateFlag(in.Templates, in.DefaultTemplateID)

	primary := selectPrimaryTemplate(&in)
	if primary != nil {
		in.TemplateID = primary.TemplateID
		in.TemplateSecret = primary.TemplateSecret
		in.TemplateTicket = primary.TemplateTicket
		in.TemplateTicketUpdatedAt = primary.TemplateTicketUpdatedAt
		in.TemplateTicketSource = primary.TemplateTicketSource
		in.ProviderCorpID = primary.ProviderCorpID
		in.ProviderSecret = primary.ProviderSecret
	}

	reconcileTemplateTicketMeta(&in, existingCfg)
	if primary := selectPrimaryTemplate(&in); primary != nil {
		primary.TemplateTicket = in.TemplateTicket
		primary.TemplateTicketUpdatedAt = in.TemplateTicketUpdatedAt
		primary.TemplateTicketSource = in.TemplateTicketSource
		primary.TemplateSecret = in.TemplateSecret
		primary.ProviderCorpID = in.ProviderCorpID
		primary.ProviderSecret = in.ProviderSecret
	}

	record, err := s.repo.UpsertByChannelProvider(ctx, "wechat", "openwork", in.Enabled, buildWeComOpenWorkConfigJSON(in))
	if err != nil {
		return nil, err
	}
	return parseWeComOpenWorkConfig(record.Enabled, record.Config), nil
}

func (s *ChannelPlatformSettingService) ListWeComOpenWorkTemplates(ctx context.Context) ([]WeComOpenWorkTemplate, error) {
	cfg, err := s.GetWeComOpenWorkConfig(ctx)
	if err != nil {
		return nil, err
	}
	return cloneTemplateList(cfg.Templates), nil
}

func (s *ChannelPlatformSettingService) CreateWeComOpenWorkTemplate(ctx context.Context, in WeComOpenWorkTemplate) (*WeComOpenWorkTemplate, error) {
	cfg, err := s.GetWeComOpenWorkConfig(ctx)
	if err != nil {
		return nil, err
	}
	in = sanitizeWeComTemplate(in)
	if in.TemplateID == "" || in.TemplateSecret == "" {
		return nil, errors.New("template_id 和 template_secret 不能为空")
	}
	if !isValidOpenWorkTemplateID(in.TemplateID) {
		return nil, errors.New("template_id 格式非法，仅允许字母/数字/_/-")
	}
	if indexOfTemplate(cfg.Templates, in.TemplateID) >= 0 {
		return nil, ErrWeComTemplateAlreadyExists
	}
	cfg.Templates = append(cfg.Templates, in)
	if cfg.DefaultTemplateID == "" {
		cfg.DefaultTemplateID = in.TemplateID
	}
	cfg = prepareConfigForTemplateCRUD(cfg)
	saved, err := s.SaveWeComOpenWorkConfig(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	idx := indexOfTemplate(saved.Templates, in.TemplateID)
	if idx < 0 {
		return nil, ErrWeComTemplateNotFound
	}
	out := saved.Templates[idx]
	return &out, nil
}

func (s *ChannelPlatformSettingService) UpdateWeComOpenWorkTemplate(ctx context.Context, templateID string, in WeComOpenWorkTemplate) (*WeComOpenWorkTemplate, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, errors.New("template_id is required")
	}
	if !isValidOpenWorkTemplateID(templateID) {
		return nil, errors.New("template_id 格式非法，仅允许字母/数字/_/-")
	}
	cfg, err := s.GetWeComOpenWorkConfig(ctx)
	if err != nil {
		return nil, err
	}
	idx := indexOfTemplate(cfg.Templates, templateID)
	if idx < 0 {
		return nil, ErrWeComTemplateNotFound
	}
	in = sanitizeWeComTemplate(in)
	if in.TemplateID != "" && in.TemplateID != templateID {
		return nil, errors.New("template_id 不允许修改")
	}
	if in.TemplateSecret == "" {
		return nil, errors.New("template_secret 不能为空")
	}
	in.TemplateID = templateID
	in.IsDefault = cfg.Templates[idx].IsDefault
	cfg.Templates[idx] = in
	cfg = prepareConfigForTemplateCRUD(cfg)
	saved, err := s.SaveWeComOpenWorkConfig(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	outIdx := indexOfTemplate(saved.Templates, templateID)
	if outIdx < 0 {
		return nil, ErrWeComTemplateNotFound
	}
	out := saved.Templates[outIdx]
	return &out, nil
}

func (s *ChannelPlatformSettingService) DeleteWeComOpenWorkTemplate(ctx context.Context, templateID string) error {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return errors.New("template_id is required")
	}
	if !isValidOpenWorkTemplateID(templateID) {
		return errors.New("template_id 格式非法，仅允许字母/数字/_/-")
	}
	cfg, err := s.GetWeComOpenWorkConfig(ctx)
	if err != nil {
		return err
	}
	idx := indexOfTemplate(cfg.Templates, templateID)
	if idx < 0 {
		return ErrWeComTemplateNotFound
	}
	cfg.Templates = append(cfg.Templates[:idx], cfg.Templates[idx+1:]...)
	if cfg.DefaultTemplateID == templateID {
		cfg.DefaultTemplateID = ""
		if len(cfg.Templates) > 0 {
			cfg.DefaultTemplateID = cfg.Templates[0].TemplateID
		}
	}
	cfg = prepareConfigForTemplateCRUD(cfg)
	_, err = s.SaveWeComOpenWorkConfig(ctx, *cfg)
	return err
}

func (s *ChannelPlatformSettingService) SetDefaultWeComOpenWorkTemplate(ctx context.Context, templateID string) (*WeComOpenWorkTemplate, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, errors.New("template_id is required")
	}
	if !isValidOpenWorkTemplateID(templateID) {
		return nil, errors.New("template_id 格式非法，仅允许字母/数字/_/-")
	}
	cfg, err := s.GetWeComOpenWorkConfig(ctx)
	if err != nil {
		return nil, err
	}
	if indexOfTemplate(cfg.Templates, templateID) < 0 {
		return nil, ErrWeComTemplateNotFound
	}
	cfg.DefaultTemplateID = templateID
	cfg = prepareConfigForTemplateCRUD(cfg)
	saved, err := s.SaveWeComOpenWorkConfig(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	idx := indexOfTemplate(saved.Templates, templateID)
	if idx < 0 {
		return nil, ErrWeComTemplateNotFound
	}
	out := saved.Templates[idx]
	return &out, nil
}

func (s *ChannelPlatformSettingService) RefreshWeComSuiteTicketStatus(ctx context.Context) (*WeComSuiteTicketStatus, error) {
	cfg, err := s.GetWeComOpenWorkConfig(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	raw := strings.TrimSpace(cfg.TemplateTicket)
	out := &WeComSuiteTicketStatus{
		Ready:                raw != "",
		TemplateID:           strings.TrimSpace(cfg.TemplateID),
		TemplateTicket:       raw,
		TemplateTicketMasked: maskTicket(raw),
		TemplateTicketSource: strings.TrimSpace(cfg.TemplateTicketSource),
		UpdatedAt:            strings.TrimSpace(cfg.TemplateTicketUpdatedAt),
		CheckedAt:            now.Format(time.RFC3339),
		Message:              "waiting_callback",
	}
	if out.Ready {
		out.Message = "ticket_ready"
	}
	return out, nil
}

func (s *ChannelPlatformSettingService) VerifyWeComSuiteTicket(ctx context.Context) (*WeComSuiteTicketVerifyResult, error) {
	cfg, err := s.GetWeComOpenWorkConfig(ctx)
	if err != nil {
		return nil, err
	}
	suiteID := strings.TrimSpace(cfg.TemplateID)
	suiteSecret := strings.TrimSpace(cfg.TemplateSecret)
	suiteTicket := strings.TrimSpace(cfg.TemplateTicket)
	out := &WeComSuiteTicketVerifyResult{
		Valid:      false,
		TemplateID: suiteID,
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
		Message:    "verify_failed",
	}
	if suiteID == "" || suiteSecret == "" {
		out.Message = "template_id_or_template_secret_missing"
		out.ErrMsg = "缺少 template_id 或 template_secret"
		return out, nil
	}
	if suiteTicket == "" {
		out.Message = "template_ticket_missing"
		out.ErrMsg = "缺少 template_ticket（请先等待回调入库）"
		return out, nil
	}

	body, err := json.Marshal(map[string]any{
		"suite_id":     suiteID,
		"suite_secret": suiteSecret,
		"suite_ticket": suiteTicket,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wecomGetSuiteTokenEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		out.ErrMsg = strings.TrimSpace(string(raw))
		out.Message = "verify_http_failed"
		return out, nil
	}
	var apiResp struct {
		ErrCode          int    `json:"errcode"`
		ErrMsg           string `json:"errmsg"`
		SuiteAccessToken string `json:"suite_access_token"`
		ExpiresIn        int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, err
	}
	out.ErrCode = apiResp.ErrCode
	out.ErrMsg = strings.TrimSpace(apiResp.ErrMsg)
	out.ExpiresIn = apiResp.ExpiresIn
	out.HasSuiteAccessToken = strings.TrimSpace(apiResp.SuiteAccessToken) != ""
	out.Valid = out.ErrCode == 0 && out.HasSuiteAccessToken
	if out.Valid {
		out.Message = "verify_ok"
	} else {
		out.Message = "verify_failed"
	}
	return out, nil
}

func parseWeComOpenWorkConfig(enabled bool, cfg datatypes.JSONMap) *WeComOpenWorkPlatformConfig {
	out := &WeComOpenWorkPlatformConfig{
		Enabled:                 enabled,
		TemplateID:              readMapString(cfg, "template_id"),
		TemplateSecret:          readMapString(cfg, "template_secret"),
		TemplateTicket:          readMapString(cfg, "template_ticket"),
		TemplateTicketUpdatedAt: readMapString(cfg, "template_ticket_updated_at"),
		TemplateTicketSource:    readMapString(cfg, "template_ticket_source"),
		ProviderCorpID:          readMapString(cfg, "provider_corpid"),
		ProviderSecret:          readMapString(cfg, "provider_secret"),
		Token:                   readMapString(cfg, "token"),
		AESKey:                  readMapString(cfg, "aes_key"),
		HTTPDebug:               readMapBool(cfg, "http_debug"),
		CallbackHost:            readMapString(cfg, "callback_host"),
		RedirectURI:             readMapString(cfg, "redirect_uri"),
		DefaultTemplateID:       readMapString(cfg, "default_template_id"),
		Templates:               parseWeComTemplateList(cfg),
	}
	if len(out.Templates) == 0 && out.TemplateID != "" {
		out.Templates = []WeComOpenWorkTemplate{{
			TemplateID:              out.TemplateID,
			TemplateSecret:          out.TemplateSecret,
			TemplateTicket:          out.TemplateTicket,
			TemplateTicketUpdatedAt: out.TemplateTicketUpdatedAt,
			TemplateTicketSource:    out.TemplateTicketSource,
			ProviderCorpID:          out.ProviderCorpID,
			ProviderSecret:          out.ProviderSecret,
			IsDefault:               true,
		}}
		if out.DefaultTemplateID == "" {
			out.DefaultTemplateID = out.TemplateID
		}
	}
	if out.DefaultTemplateID == "" && out.TemplateID != "" {
		out.DefaultTemplateID = out.TemplateID
	}
	if out.DefaultTemplateID != "" {
		out.Templates = applyDefaultTemplateFlag(out.Templates, out.DefaultTemplateID)
	}
	if primary := selectPrimaryTemplate(out); primary != nil {
		out.TemplateID = primary.TemplateID
		out.TemplateSecret = primary.TemplateSecret
		out.TemplateTicket = primary.TemplateTicket
		out.TemplateTicketUpdatedAt = primary.TemplateTicketUpdatedAt
		out.TemplateTicketSource = primary.TemplateTicketSource
		out.ProviderCorpID = primary.ProviderCorpID
		out.ProviderSecret = primary.ProviderSecret
	}
	sanitized := sanitizeWeComOpenWorkConfig(*out)
	return &sanitized
}

func sanitizeWeComOpenWorkConfig(in WeComOpenWorkPlatformConfig) WeComOpenWorkPlatformConfig {
	in.TemplateID = strings.TrimSpace(in.TemplateID)
	in.TemplateSecret = strings.TrimSpace(in.TemplateSecret)
	in.TemplateTicket = strings.TrimSpace(in.TemplateTicket)
	in.TemplateTicketUpdatedAt = strings.TrimSpace(in.TemplateTicketUpdatedAt)
	in.TemplateTicketSource = strings.ToLower(strings.TrimSpace(in.TemplateTicketSource))
	in.ProviderCorpID = strings.TrimSpace(in.ProviderCorpID)
	in.ProviderSecret = strings.TrimSpace(in.ProviderSecret)
	in.Token = strings.TrimSpace(in.Token)
	in.AESKey = strings.TrimSpace(in.AESKey)
	in.CallbackHost = strings.TrimRight(strings.TrimSpace(in.CallbackHost), "/")
	in.RedirectURI = strings.TrimSpace(in.RedirectURI)
	in.DefaultTemplateID = strings.TrimSpace(in.DefaultTemplateID)
	in.Templates = normalizeTemplateList(in.Templates)
	if in.DefaultTemplateID != "" {
		in.Templates = applyDefaultTemplateFlag(in.Templates, in.DefaultTemplateID)
	}
	return in
}

func sanitizeWeComTemplate(in WeComOpenWorkTemplate) WeComOpenWorkTemplate {
	in.TemplateID = strings.TrimSpace(in.TemplateID)
	in.TemplateSecret = strings.TrimSpace(in.TemplateSecret)
	in.TemplateTicket = strings.TrimSpace(in.TemplateTicket)
	in.TemplateTicketUpdatedAt = strings.TrimSpace(in.TemplateTicketUpdatedAt)
	in.TemplateTicketSource = strings.ToLower(strings.TrimSpace(in.TemplateTicketSource))
	in.ProviderCorpID = strings.TrimSpace(in.ProviderCorpID)
	in.ProviderSecret = strings.TrimSpace(in.ProviderSecret)
	return in
}

func prepareConfigForTemplateCRUD(cfg *WeComOpenWorkPlatformConfig) *WeComOpenWorkPlatformConfig {
	if cfg == nil {
		return &WeComOpenWorkPlatformConfig{Enabled: true}
	}
	cfg.Templates = normalizeTemplateList(cfg.Templates)
	if cfg.DefaultTemplateID == "" && len(cfg.Templates) > 0 {
		cfg.DefaultTemplateID = cfg.Templates[0].TemplateID
	}
	if cfg.DefaultTemplateID != "" && indexOfTemplate(cfg.Templates, cfg.DefaultTemplateID) < 0 {
		if len(cfg.Templates) > 0 {
			cfg.DefaultTemplateID = cfg.Templates[0].TemplateID
		} else {
			cfg.DefaultTemplateID = ""
		}
	}
	cfg.Templates = applyDefaultTemplateFlag(cfg.Templates, cfg.DefaultTemplateID)
	if primary := selectPrimaryTemplate(cfg); primary != nil {
		cfg.TemplateID = primary.TemplateID
		cfg.TemplateSecret = primary.TemplateSecret
		cfg.TemplateTicket = primary.TemplateTicket
		cfg.TemplateTicketUpdatedAt = primary.TemplateTicketUpdatedAt
		cfg.TemplateTicketSource = primary.TemplateTicketSource
		cfg.ProviderCorpID = primary.ProviderCorpID
		cfg.ProviderSecret = primary.ProviderSecret
	} else {
		cfg.TemplateID = ""
		cfg.TemplateSecret = ""
		cfg.TemplateTicket = ""
		cfg.TemplateTicketUpdatedAt = ""
		cfg.TemplateTicketSource = ""
		cfg.ProviderCorpID = ""
		cfg.ProviderSecret = ""
	}
	return cfg
}

func parseWeComTemplateList(cfg datatypes.JSONMap) []WeComOpenWorkTemplate {
	if cfg == nil {
		return nil
	}
	raw, ok := cfg["templates"]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		if arr, ok2 := raw.([]map[string]any); ok2 {
			items = make([]any, 0, len(arr))
			for _, one := range arr {
				items = append(items, one)
			}
		} else {
			return nil
		}
	}
	out := make([]WeComOpenWorkTemplate, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tpl := sanitizeWeComTemplate(WeComOpenWorkTemplate{
			TemplateID:              readAnyString(m["template_id"]),
			TemplateSecret:          readAnyString(m["template_secret"]),
			TemplateTicket:          readAnyString(m["template_ticket"]),
			TemplateTicketUpdatedAt: readAnyString(m["template_ticket_updated_at"]),
			TemplateTicketSource:    readAnyString(m["template_ticket_source"]),
			ProviderCorpID:          readAnyString(m["provider_corpid"]),
			ProviderSecret:          readAnyString(m["provider_secret"]),
			IsDefault:               readAnyBool(m["is_default"]),
		})
		if tpl.TemplateID == "" {
			continue
		}
		out = append(out, tpl)
	}
	return normalizeTemplateList(out)
}

func normalizeTemplateList(in []WeComOpenWorkTemplate) []WeComOpenWorkTemplate {
	if len(in) == 0 {
		return nil
	}
	out := make([]WeComOpenWorkTemplate, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		tpl := sanitizeWeComTemplate(item)
		if tpl.TemplateID == "" {
			continue
		}
		if _, ok := seen[tpl.TemplateID]; ok {
			continue
		}
		seen[tpl.TemplateID] = struct{}{}
		out = append(out, tpl)
	}
	return out
}

func applyDefaultTemplateFlag(in []WeComOpenWorkTemplate, defaultTemplateID string) []WeComOpenWorkTemplate {
	defaultTemplateID = strings.TrimSpace(defaultTemplateID)
	if len(in) == 0 {
		return in
	}
	for i := range in {
		in[i].IsDefault = defaultTemplateID != "" && in[i].TemplateID == defaultTemplateID
	}
	return in
}

func selectPrimaryTemplate(cfg *WeComOpenWorkPlatformConfig) *WeComOpenWorkTemplate {
	if cfg == nil || len(cfg.Templates) == 0 {
		return nil
	}
	if cfg.DefaultTemplateID != "" {
		for i := range cfg.Templates {
			if cfg.Templates[i].TemplateID == cfg.DefaultTemplateID {
				return &cfg.Templates[i]
			}
		}
	}
	for i := range cfg.Templates {
		if cfg.Templates[i].IsDefault {
			return &cfg.Templates[i]
		}
	}
	return &cfg.Templates[0]
}

func cloneTemplateList(in []WeComOpenWorkTemplate) []WeComOpenWorkTemplate {
	if len(in) == 0 {
		return nil
	}
	out := make([]WeComOpenWorkTemplate, len(in))
	copy(out, in)
	return out
}

func reconcileTemplateTicketMeta(in *WeComOpenWorkPlatformConfig, existing *WeComOpenWorkPlatformConfig) {
	if in == nil {
		return
	}
	if existing == nil {
		if in.TemplateTicket != "" {
			if in.TemplateTicketUpdatedAt == "" {
				in.TemplateTicketUpdatedAt = time.Now().UTC().Format(time.RFC3339)
			}
			if in.TemplateTicketSource == "" {
				in.TemplateTicketSource = "manual"
			}
		}
		return
	}
	if in.TemplateTicket == "" {
		if in.TemplateID == existing.TemplateID {
			in.TemplateTicket = existing.TemplateTicket
			in.TemplateTicketUpdatedAt = existing.TemplateTicketUpdatedAt
			in.TemplateTicketSource = existing.TemplateTicketSource
			return
		}
		in.TemplateTicketUpdatedAt = ""
		in.TemplateTicketSource = ""
		return
	}
	if in.TemplateTicket == existing.TemplateTicket && in.TemplateID == existing.TemplateID {
		if in.TemplateTicketUpdatedAt == "" {
			in.TemplateTicketUpdatedAt = existing.TemplateTicketUpdatedAt
		}
		if in.TemplateTicketSource == "" {
			in.TemplateTicketSource = existing.TemplateTicketSource
		}
		return
	}
	if in.TemplateTicketUpdatedAt == "" {
		in.TemplateTicketUpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if in.TemplateTicketSource == "" {
		in.TemplateTicketSource = "manual"
	}
}

func buildWeComOpenWorkConfigJSON(in WeComOpenWorkPlatformConfig) datatypes.JSONMap {
	templates := make([]map[string]any, 0, len(in.Templates))
	for _, item := range normalizeTemplateList(in.Templates) {
		templates = append(templates, map[string]any{
			"template_id":                item.TemplateID,
			"template_secret":            item.TemplateSecret,
			"template_ticket":            item.TemplateTicket,
			"template_ticket_updated_at": item.TemplateTicketUpdatedAt,
			"template_ticket_source":     item.TemplateTicketSource,
			"provider_corpid":            item.ProviderCorpID,
			"provider_secret":            item.ProviderSecret,
			"is_default":                 item.IsDefault,
		})
	}
	cfg := datatypes.JSONMap{
		"template_id":                in.TemplateID,
		"template_secret":            in.TemplateSecret,
		"template_ticket":            in.TemplateTicket,
		"template_ticket_updated_at": in.TemplateTicketUpdatedAt,
		"template_ticket_source":     in.TemplateTicketSource,
		"provider_corpid":            in.ProviderCorpID,
		"provider_secret":            in.ProviderSecret,
		"token":                      in.Token,
		"aes_key":                    in.AESKey,
		"http_debug":                 in.HTTPDebug,
		"callback_host":              in.CallbackHost,
		"redirect_uri":               in.RedirectURI,
		"default_template_id":        in.DefaultTemplateID,
		"templates":                  templates,
	}
	return cfg
}

func indexOfTemplate(in []WeComOpenWorkTemplate, templateID string) int {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return -1
	}
	for i := range in {
		if in[i].TemplateID == templateID {
			return i
		}
	}
	return -1
}

func readAnyString(raw any) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", raw))
}

func readAnyBool(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		return v == "1" || v == "true" || v == "yes"
	default:
		return false
	}
}

func readMapString(input datatypes.JSONMap, key string) string {
	if input == nil {
		return ""
	}
	raw, ok := input[key]
	if !ok || raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", raw))
}

func readMapBool(input datatypes.JSONMap, key string) bool {
	if input == nil {
		return false
	}
	raw, ok := input[key]
	if !ok || raw == nil {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		val := strings.TrimSpace(strings.ToLower(v))
		return val == "1" || val == "true" || val == "yes"
	default:
		return false
	}
}

func maskTicket(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= 8 {
		return raw[:1] + "***" + raw[len(raw)-1:]
	}
	return raw[:4] + "***" + raw[len(raw)-4:]
}
