package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	work "github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	pwmsgtemplate "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/messageTemplate"
	pwmsgreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/messageTemplate/request"
	pwtag "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/tag"
	pwtagreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/tag/request"
)

type StaffLiveCodeContactApplyTarget struct {
	StaffCodeUUID      string
	TenantUUID         string
	ChannelAccountUUID string
	CorpTagIDs         []string
	WelcomeMode        string
	ContentBlocksRaw   []byte
}

type StaffLiveCodeContactEventApplyRequest struct {
	TenantUUID         string
	ChannelAccountUUID string
	ExternalUserID     string
	WelcomeCode        string
	Target             *StaffLiveCodeContactApplyTarget
}

type StaffLiveCodeContactEventService struct {
	resolver StaffDefaultAccountResolver
}

type StaffLiveCodeContactApplyResult struct {
	OperatorUserID      string
	ExternalDisplayName string
}

func NewStaffLiveCodeContactEventService(resolver StaffDefaultAccountResolver) *StaffLiveCodeContactEventService {
	return &StaffLiveCodeContactEventService{resolver: resolver}
}

func (s *StaffLiveCodeContactEventService) Apply(ctx context.Context, req StaffLiveCodeContactEventApplyRequest) (*StaffLiveCodeContactApplyResult, error) {
	if s == nil || s.resolver == nil || req.Target == nil {
		return nil, errors.New("staff contact event service not ready")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(req.TenantUUID))
	accountUUID := strings.ToLower(strings.TrimSpace(req.ChannelAccountUUID))
	externalUserID := strings.TrimSpace(req.ExternalUserID)
	welcomeCode := strings.TrimSpace(req.WelcomeCode)
	if tenantUUID == "" || accountUUID == "" || externalUserID == "" {
		return nil, errors.New("tenant/account/external user is required")
	}

	account, err := s.resolver.GetChannelAccount(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	credentials, err := s.resolver.GetChannelAccountCredentials(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	app, err := s.buildWorkApp(account, credentials)
	if err != nil {
		return nil, err
	}
	if app == nil || app.ExternalContact == nil {
		return nil, errors.New("wecom external contact client unavailable")
	}

	operatorUserID, externalDisplayName, err := s.resolveOperatorUserID(ctx, app, externalUserID)
	if err != nil {
		return nil, err
	}
	result := &StaffLiveCodeContactApplyResult{
		OperatorUserID:      operatorUserID,
		ExternalDisplayName: externalDisplayName,
	}

	if err := s.applyCorpTags(ctx, app, operatorUserID, externalUserID, req.Target.CorpTagIDs); err != nil {
		return result, err
	}
	if err := s.sendWelcomeIfNeeded(ctx, app, welcomeCode, req.Target); err != nil {
		return result, err
	}
	return result, nil
}

func (s *StaffLiveCodeContactEventService) buildWorkApp(account *GroupChatAccountProfile, credentials map[string]string) (*work.Work, error) {
	if account == nil {
		return nil, errors.New("channel account profile unavailable")
	}
	channelCode := strings.ToLower(strings.TrimSpace(account.ChannelCode))
	appType := strings.ToLower(strings.TrimSpace(account.AppType))
	if channelCode != "wechat" {
		return nil, fmt.Errorf("unsupported channel: %s", channelCode)
	}
	switch appType {
	case "wecom":
		return buildWeComSelfBuiltWorkApp(credentials)
	case "openwork":
		return buildWeComOpenWorkApp(credentials)
	default:
		return nil, fmt.Errorf("unsupported app type: %s", appType)
	}
}

func (s *StaffLiveCodeContactEventService) resolveOperatorUserID(ctx context.Context, app *work.Work, externalUserID string) (string, string, error) {
	detailResp, err := app.ExternalContact.Get(ctx, externalUserID, "")
	if err != nil {
		return "", "", fmt.Errorf("externalcontact.get request failed: %w", err)
	}
	if detailResp == nil {
		return "", "", errors.New("externalcontact.get empty response")
	}
	if detailResp.ErrCode != 0 {
		return "", "", fmt.Errorf("externalcontact.get failed: %d %s", detailResp.ErrCode, strings.TrimSpace(detailResp.ErrMSG))
	}
	externalDisplayName := ""
	if detailResp.ExternalContact != nil {
		externalDisplayName = strings.TrimSpace(detailResp.ExternalContact.Name)
	}
	for _, follow := range detailResp.FollowUsers {
		if follow == nil {
			continue
		}
		userID := strings.TrimSpace(follow.UserID)
		if userID == "" {
			userID = strings.TrimSpace(follow.OperUserID)
		}
		if userID != "" {
			return userID, externalDisplayName, nil
		}
	}
	return "", externalDisplayName, errors.New("operator user id not found")
}

func (s *StaffLiveCodeContactEventService) applyCorpTags(ctx context.Context, app *work.Work, operatorUserID, externalUserID string, corpTagIDs []string) error {
	tagIDs := normalizeCorpTagIDs(corpTagIDs)
	if len(tagIDs) == 0 {
		return nil
	}
	tagClient, err := pwtag.NewClient(app)
	if err != nil {
		return fmt.Errorf("init wecom tag client failed: %w", err)
	}
	if tagClient == nil {
		return errors.New("wecom tag client unavailable")
	}
	resp, err := tagClient.MarkTag(ctx, &pwtagreq.RequestTagMarkTag{
		UserID:         operatorUserID,
		ExternalUserID: externalUserID,
		AddTag:         tagIDs,
	})
	if err != nil {
		return fmt.Errorf("mark_tag request failed: %w", err)
	}
	if resp == nil {
		return errors.New("mark_tag empty response")
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("mark_tag failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	return nil
}

func (s *StaffLiveCodeContactEventService) sendWelcomeIfNeeded(ctx context.Context, app *work.Work, welcomeCode string, target *StaffLiveCodeContactApplyTarget) error {
	if target == nil {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(target.WelcomeMode))
	if mode == "" {
		mode = "send"
	}
	if mode != "send" {
		return nil
	}
	if strings.TrimSpace(welcomeCode) == "" {
		return errors.New("missing welcome_code for send_welcome_msg")
	}
	msgReq, err := buildSendWelcomeRequest(welcomeCode, target.ContentBlocksRaw)
	if err != nil {
		return err
	}
	client, err := pwmsgtemplate.NewClient(app)
	if err != nil {
		return fmt.Errorf("init wecom message template client failed: %w", err)
	}
	if client == nil {
		return errors.New("wecom message template client unavailable")
	}
	resp, err := client.SendWelcomeMsg(ctx, msgReq)
	if err != nil {
		return fmt.Errorf("send_welcome_msg request failed: %w", err)
	}
	if resp == nil {
		return errors.New("send_welcome_msg empty response")
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("send_welcome_msg failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	return nil
}

func buildSendWelcomeRequest(welcomeCode string, raw []byte) (*pwmsgreq.RequestSendWelcomeMsg, error) {
	blocks := make([]map[string]any, 0)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &blocks); err != nil {
			return nil, fmt.Errorf("parse welcome content blocks failed: %w", err)
		}
	}
	req := &pwmsgreq.RequestSendWelcomeMsg{
		WelcomeCode: strings.TrimSpace(welcomeCode),
	}
	attachments := make([]pwmsgreq.MessageTemplateInterface, 0, 2)
	textContent := ""
	for _, block := range blocks {
		t := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", block["type"])))
		switch t {
		case "text":
			content := strings.TrimSpace(fmt.Sprintf("%v", block["content"]))
			if content != "" && textContent == "" {
				textContent = content
			}
		case "image":
			mediaID := strings.TrimSpace(fmt.Sprintf("%v", block["media_id"]))
			url := strings.TrimSpace(fmt.Sprintf("%v", block["url"]))
			if mediaID == "" && url == "" {
				continue
			}
			attachments = append(attachments, &pwmsgreq.Attachment{
				MsgType: "image",
				Image: &pwmsgreq.Image{
					MediaID: mediaID,
					PicURL:  url,
				},
			})
		case "link":
			title := strings.TrimSpace(fmt.Sprintf("%v", block["title"]))
			url := strings.TrimSpace(fmt.Sprintf("%v", block["url"]))
			desc := strings.TrimSpace(fmt.Sprintf("%v", block["desc"]))
			if title == "" && url == "" {
				continue
			}
			attachments = append(attachments, &pwmsgreq.Attachment{
				MsgType: "link",
				Link: &pwmsgreq.Link{
					Title: title,
					URL:   url,
					Desc:  desc,
				},
			})
		case "mini_program":
			title := strings.TrimSpace(fmt.Sprintf("%v", block["title"]))
			appID := strings.TrimSpace(fmt.Sprintf("%v", block["appid"]))
			page := strings.TrimSpace(fmt.Sprintf("%v", block["page"]))
			if appID == "" || page == "" {
				continue
			}
			attachments = append(attachments, &pwmsgreq.Attachment{
				MsgType: "miniprogram",
				MiniProgram: &pwmsgreq.MiniProgram{
					Title: title,
					AppID: appID,
					Page:  page,
				},
			})
		}
	}
	if textContent == "" {
		textContent = "欢迎添加，我们将尽快联系你"
	}
	req.Text = &pwmsgreq.TextOfMessage{Content: textContent}
	if len(attachments) > 0 {
		req.Attachments = attachments
	}
	return req, nil
}
