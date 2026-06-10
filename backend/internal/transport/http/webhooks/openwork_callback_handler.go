package webhooks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	kernelmodels "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/models"
	openwork "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork"
	openworkmodel "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork/server/models"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialsvcmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/social_channel_governance"
	acqdto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OpenWorkCallbackHandler struct {
	platformRepo *repository.ChannelPlatformSettingRepository
	openWorkRepo *repository.OpenWorkFoundationRepository
	foundation   *socialsvc.OpenWorkFoundationService
	deps         *app.Deps
	publisher    fwwsbus.Publisher
}

const (
	TopicOpenWorkAuthStatus = "openwork.auth.status"
)

func NewOpenWorkCallbackHandler(platformRepo *repository.ChannelPlatformSettingRepository, openWorkRepo *repository.OpenWorkFoundationRepository, foundation *socialsvc.OpenWorkFoundationService, deps *app.Deps, publisher fwwsbus.Publisher) *OpenWorkCallbackHandler {
	h := &OpenWorkCallbackHandler{
		platformRepo: platformRepo,
		openWorkRepo: openWorkRepo,
		foundation:   foundation,
		deps:         deps,
		publisher:    publisher,
	}
	h.startWorkerLoop()
	return h
}

func (h *OpenWorkCallbackHandler) Handle(c *gin.Context) {
	if h == nil || h.platformRepo == nil {
		c.String(http.StatusServiceUnavailable, "openwork callback unavailable")
		return
	}

	templateID := strings.TrimSpace(c.Param("suite_id"))
	ackStartedAt := time.Now()
	logrus.WithFields(logrus.Fields{
		"module":        "openwork_callback",
		"method":        c.Request.Method,
		"path":          c.Request.URL.Path,
		"query":         c.Request.URL.RawQuery,
		"msg_signature": strings.TrimSpace(c.Query("msg_signature")) != "",
		"timestamp":     strings.TrimSpace(c.Query("timestamp")),
		"nonce":         strings.TrimSpace(c.Query("nonce")) != "",
	}).Info("openwork callback received")
	rawBody, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		c.String(http.StatusBadRequest, "read request body failed: "+readErr.Error())
		return
	}
	if len(rawBody) == 0 {
		c.Request.Body = http.NoBody
	} else {
		c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))
	}
	logrus.WithFields(logrus.Fields{
		"module":   "openwork_callback",
		"body_len": len(rawBody),
		"raw_body": string(rawBody),
	}).Info("openwork callback raw request body")

	record, err := h.platformRepo.GetByChannelProvider(c.Request.Context(), "wechat", "openwork")
	if err != nil {
		c.String(http.StatusBadRequest, "openwork platform config not found")
		return
	}
	if !record.Enabled {
		c.String(http.StatusBadRequest, "openwork platform config is disabled")
		return
	}

	cfgTemplateID := readMapString(record.Config, "template_id")
	if cfgTemplateID == "" {
		cfgTemplateID = readMapString(record.Config, "suite_id")
	}
	templates := readTemplatesFromConfig(record.Config)
	token := readMapString(record.Config, "token")
	aesKey := readMapString(record.Config, "aes_key")
	msgSignature := strings.TrimSpace(c.Query("msg_signature"))
	timestamp := strings.TrimSpace(c.Query("timestamp"))
	nonce := strings.TrimSpace(c.Query("nonce"))
	if templateID != "" && cfgTemplateID != "" && cfgTemplateID != templateID && indexOfTemplateRow(templates, templateID) < 0 {
		c.String(http.StatusBadRequest, "suite_id mismatch")
		return
	}
	if templateID != "" {
		cfgTemplateID = templateID
	} else if cfgTemplateID == "" {
		cfgTemplateID = templateID
	}
	if token == "" || aesKey == "" {
		c.String(http.StatusBadRequest, "token/aes_key is required")
		return
	}
	if msgSignature == "" || timestamp == "" || nonce == "" {
		c.String(http.StatusBadRequest, "msg_signature/timestamp/nonce is required")
		return
	}
	if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
		c.String(http.StatusBadRequest, "timestamp must be unix seconds")
		return
	}
	if _, err := strconv.ParseInt(nonce, 10, 64); err != nil {
		c.String(http.StatusBadRequest, "nonce must be numeric")
		return
	}
	if c.Request.Method == http.MethodGet && strings.TrimSpace(c.Query("echostr")) == "" {
		c.String(http.StatusBadRequest, "echostr is required for GET verify")
		return
	}

	app, err := openwork.NewOpenWork(&openwork.UserConfig{
		AppID:  cfgTemplateID,
		Token:  token,
		AESKey: aesKey,
	})
	if err != nil {
		c.String(http.StatusBadRequest, "openwork init failed: "+err.Error())
		return
	}

	httpResp, err := safeOpenWorkNotify(app, c.Request, func(cb *kernelmodels.Callback, ev openworkmodel.IEvent, raw interface{}) interface{} {
		infoType := strings.ToLower(strings.TrimSpace(ev.GetInfoType()))
		if infoType == "" {
			infoType = strings.ToLower(strings.TrimSpace(readEventStringField(ev, "Event", "InfoType")))
		}
		if infoType == "" {
			infoType = strings.ToLower(strings.TrimSpace(readEventStringField(raw, "Event", "InfoType")))
		}
		eventTemplateID := strings.TrimSpace(ev.GetSuiteID())
		if eventTemplateID == "" {
			eventTemplateID = cfgTemplateID
		}
		logrus.WithFields(logrus.Fields{
			"module":         "openwork_callback",
			"suite_id":       eventTemplateID,
			"event_type":     infoType,
			"callback_type":  fmt.Sprintf("%T", cb),
			"event_obj_type": fmt.Sprintf("%T", ev),
			"raw_obj_type":   fmt.Sprintf("%T", raw),
			"callback_obj":   marshalLogObject(cb),
			"event_obj":      marshalLogObject(ev),
			"raw_obj":        marshalLogObject(raw),
		}).Info("openwork callback payload object")
		logrus.WithFields(logrus.Fields{
			"module":     "openwork_callback",
			"event_type": infoType,
			"suite_id":   eventTemplateID,
		}).Info("openwork callback event parsed")
		if err := h.enqueueCallbackTask(c.Request.Context(), eventTemplateID, infoType, ev, raw, msgSignature, timestamp, nonce); err != nil {
			logrus.WithFields(logrus.Fields{
				"module":      "openwork_callback",
				"suite_id":    eventTemplateID,
				"event_type":  infoType,
				"error":       err.Error(),
				"tenant_uuid": h.resolveTenantUUID(),
			}).Warn("openwork callback enqueue task failed")
			socialobs.RecordOpenWorkCallback("enqueue_failed", infoType)
		} else {
			socialobs.RecordOpenWorkCallback("accepted", infoType)
			socialobs.ObserveOpenWorkCallbackAckLatency(float64(time.Since(ackStartedAt).Milliseconds()), infoType)
		}
		return "success"
	})
	if err != nil {
		c.String(http.StatusBadRequest, "openwork callback verify failed: "+err.Error())
		return
	}
	if httpResp == nil {
		c.String(http.StatusBadRequest, "openwork callback empty response")
		return
	}
	defer httpResp.Body.Close()

	for k, values := range httpResp.Header {
		for _, v := range values {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Status(httpResp.StatusCode)
	_, _ = io.Copy(c.Writer, httpResp.Body)
}

func (h *OpenWorkCallbackHandler) saveSuiteTicket(ctx context.Context, templateID, suiteTicket string) error {
	if h == nil || h.platformRepo == nil || suiteTicket == "" {
		return nil
	}
	record, err := h.platformRepo.GetByChannelProvider(ctx, "wechat", "openwork")
	if err != nil {
		return err
	}
	cfg := record.Config
	if cfg == nil {
		cfg = datatypes.JSONMap{}
	}
	templateID = strings.TrimSpace(templateID)
	now := time.Now().UTC().Format(time.RFC3339)
	defaultTemplateID := readMapString(cfg, "default_template_id")
	templates := readTemplatesFromConfig(cfg)
	if templateID != "" {
		idx := indexOfTemplateRow(templates, templateID)
		if idx < 0 {
			templates = append(templates, map[string]any{"template_id": templateID})
			idx = len(templates) - 1
		}
		templates[idx]["template_id"] = templateID
		templates[idx]["template_ticket"] = suiteTicket
		templates[idx]["template_ticket_updated_at"] = now
		templates[idx]["template_ticket_source"] = "callback"
		if defaultTemplateID == "" {
			defaultTemplateID = templateID
		}
	}
	if defaultTemplateID == "" {
		defaultTemplateID = readMapString(cfg, "template_id")
	}
	if idx := indexOfTemplateRow(templates, defaultTemplateID); idx >= 0 {
		templates[idx]["is_default"] = true
		cfg["template_id"] = readAnyString(templates[idx]["template_id"])
		cfg["template_secret"] = readAnyString(templates[idx]["template_secret"])
		cfg["provider_corpid"] = readAnyString(templates[idx]["provider_corpid"])
		cfg["provider_secret"] = readAnyString(templates[idx]["provider_secret"])
		cfg["template_ticket"] = readAnyString(templates[idx]["template_ticket"])
		cfg["template_ticket_updated_at"] = readAnyString(templates[idx]["template_ticket_updated_at"])
		cfg["template_ticket_source"] = readAnyString(templates[idx]["template_ticket_source"])
	} else {
		if templateID != "" {
			cfg["template_id"] = templateID
		}
		cfg["template_ticket"] = suiteTicket
		cfg["template_ticket_updated_at"] = now
		cfg["template_ticket_source"] = "callback"
	}
	cfg["default_template_id"] = defaultTemplateID
	cfg["templates"] = templates
	logrus.WithFields(logrus.Fields{
		"module":      "openwork_callback",
		"template_id": templateID,
		"ticket_len":  len(suiteTicket),
	}).Info("openwork callback suite_ticket persisted")
	_, err = h.platformRepo.UpsertByChannelProvider(ctx, "wechat", "openwork", record.Enabled, cfg)
	return err
}

func (h *OpenWorkCallbackHandler) publishAuthStatus(ctx context.Context, templateID, infoType, corpID, agentID string) {
	if h == nil || h.publisher == nil {
		return
	}
	templateID = strings.TrimSpace(templateID)
	infoType = strings.ToLower(strings.TrimSpace(infoType))
	if templateID == "" || infoType == "" {
		return
	}
	status := "pending"
	message := "waiting_callback"
	switch infoType {
	case "create_auth", "change_auth", "reset_permanent_code":
		status = "authorized"
		message = "authorization_completed"
	case "cancel_auth":
		status = "failed"
		message = "authorization_canceled"
	}
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	if h.deps != nil && h.deps.Config != nil && h.deps.Config.GRPCUpstream != nil {
		if candidate := strings.TrimSpace(h.deps.Config.GRPCUpstream.TenantUUID); candidate != "" {
			tenantUUID = candidate
		}
	}
	payload := map[string]any{
		"template_id": templateID,
		"status":      status,
		"message":     message,
		"event_type":  infoType,
		"corp_id":     strings.TrimSpace(corpID),
		"agent_id":    strings.TrimSpace(agentID),
		"checked_at":  time.Now().UTC().Format(time.RFC3339Nano),
	}
	result := h.publisher.Publish(ctx, TopicOpenWorkAuthStatus, payload, fwwsbus.PublishOptions{
		TenantUUID: tenantUUID,
	})
	logFields := logrus.Fields{
		"module":      "openwork_callback",
		"topic":       TopicOpenWorkAuthStatus,
		"tenant_uuid": tenantUUID,
		"template_id": templateID,
		"event_type":  infoType,
		"status":      status,
		"corp_id":     strings.TrimSpace(corpID),
		"agent_id":    strings.TrimSpace(agentID),
	}
	if !result.OK {
		logFields["error_code"] = result.ErrorCode
		logFields["error_message"] = result.ErrorMessage
		logrus.WithFields(logFields).Warn("openwork callback ws publish failed")
		return
	}
	logrus.WithFields(logFields).Info("openwork callback ws published")
}

func readTemplatesFromConfig(cfg datatypes.JSONMap) []map[string]any {
	if cfg == nil {
		return nil
	}
	raw, ok := cfg["templates"]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		if listMap, okMap := raw.([]map[string]any); okMap {
			out := make([]map[string]any, 0, len(listMap))
			for _, row := range listMap {
				cp := map[string]any{}
				for k, v := range row {
					cp[k] = v
				}
				out = append(out, cp)
			}
			return out
		}
		return nil
	}
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cp := map[string]any{}
		for k, v := range row {
			cp[k] = v
		}
		out = append(out, cp)
	}
	return out
}

func indexOfTemplateRow(rows []map[string]any, templateID string) int {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return -1
	}
	for i, row := range rows {
		if strings.TrimSpace(readAnyString(row["template_id"])) == templateID {
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

func marshalLogObject(v any) string {
	if v == nil {
		return ""
	}
	encoded, err := json.Marshal(v)
	if err == nil {
		return string(encoded)
	}
	return fmt.Sprintf("%+v", v)
}

func safeOpenWorkNotify(app *openwork.OpenWork, req *http.Request, handler func(*kernelmodels.Callback, openworkmodel.IEvent, interface{}) interface{}) (resp *http.Response, err error) {
	if app == nil || app.Server == nil {
		return nil, errors.New("openwork server is not initialized")
	}
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("openwork notify panic: %v", p)
			resp = nil
		}
	}()
	return app.Server.Notify(req, handler)
}

func (h *OpenWorkCallbackHandler) enqueueCallbackTask(ctx context.Context, templateID, infoType string, ev openworkmodel.IEvent, raw any, msgSignature, timestamp, nonce string) error {
	if h == nil || h.openWorkRepo == nil || ev == nil {
		return nil
	}
	tenantUUID := h.resolveTenantUUID()
	corpID := strings.TrimSpace(readEventStringField(ev, "AuthCorpID", "CorpID"))
	agentID := strings.TrimSpace(readEventStringField(ev, "AgentID"))
	authCode := strings.TrimSpace(readEventStringField(ev, "AuthCode"))
	state := firstNonEmpty(
		readEventStringField(ev, "State"),
		readEventStringField(raw, "State"),
	)
	suiteTicket := readEventStringField(ev, "SuiteTicket")
	eventTime := readEventUnixField(ev, "Timestamp", "TimeStamp")
	eventKey := buildOpenWorkEventKey(tenantUUID, templateID, infoType, authCode, msgSignature, timestamp, nonce, ev)
	callbackKey := buildOpenWorkCallbackKey(infoType, authCode, msgSignature, timestamp, nonce, eventKey)
	var eventAt *time.Time
	if eventTime > 0 {
		t := time.Unix(eventTime, 0).UTC()
		eventAt = &t
	}
	payload := map[string]any{
		"state":         state,
		"event_key":     eventKey,
		"msg_signature": strings.TrimSpace(msgSignature),
		"timestamp":     strings.TrimSpace(timestamp),
		"nonce":         strings.TrimSpace(nonce),
	}
	if chatID := firstNonEmpty(
		readEventStringField(ev, "ChatId", "ChatID"),
		readEventStringField(raw, "ChatId", "ChatID"),
	); chatID != "" {
		payload["chat_id"] = strings.TrimSpace(chatID)
	}
	if externalUserID := firstNonEmpty(
		readEventStringField(ev, "ExternalUserID", "ExternalUserid", "ExternalUserId", "UserID", "UserId"),
		readEventStringField(raw, "ExternalUserID", "ExternalUserid", "ExternalUserId", "UserID", "UserId"),
		readExternalUserIDFromRaw(raw),
	); externalUserID != "" {
		payload["external_userid"] = strings.TrimSpace(externalUserID)
	}
	if welcomeCode := firstNonEmpty(
		readEventStringField(ev, "WelcomeCode", "Welcome_Code"),
		readEventStringField(raw, "WelcomeCode", "Welcome_Code"),
	); welcomeCode != "" {
		payload["welcome_code"] = strings.TrimSpace(welcomeCode)
	}
	if changeType := firstNonEmpty(readEventStringField(ev, "ChangeType", "UpdateDetail"), readEventStringField(raw, "ChangeType", "UpdateDetail")); changeType != "" {
		payload["change_type"] = strings.ToLower(strings.TrimSpace(changeType))
	}
	if operatorUserID := firstNonEmpty(
		readEventStringField(ev, "UserID", "UserId", "Userid", "FollowUserID", "FollowUserid"),
		readEventStringField(raw, "UserID", "UserId", "Userid", "FollowUserID", "FollowUserid"),
	); operatorUserID != "" {
		payload["operator_userid"] = strings.TrimSpace(operatorUserID)
	}
	if authCode != "" {
		payload["auth_code_present"] = true
	}
	task := &socialsvcmodel.WeComOpenCallbackTask{
		TenantUUID:   tenantUUID,
		SuiteID:      strings.TrimSpace(templateID),
		EventType:    strings.ToLower(strings.TrimSpace(infoType)),
		CallbackKey:  callbackKey,
		EventKey:     eventKey,
		AuthCode:     authCode,
		CorpID:       corpID,
		AgentID:      agentID,
		SuiteTicket:  strings.TrimSpace(suiteTicket),
		State:        strings.TrimSpace(state),
		MsgSignature: strings.TrimSpace(msgSignature),
		Nonce:        strings.TrimSpace(nonce),
		Status:       socialsvcmodel.OpenWorkCallbackTaskReceived,
		MaxAttempts:  3,
		Payload:      payload,
		EventTime:    eventAt,
	}
	if ts, parseErr := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64); parseErr == nil && ts > 0 {
		task.Timestamp = ts
	}
	saved, created, err := h.openWorkRepo.EnqueueCallbackTask(ctx, task)
	if err != nil {
		return err
	}
	if !created {
		socialobs.RecordOpenWorkIdempotentHit("callback_key", infoType)
	}
	logrus.WithFields(logrus.Fields{
		"module":         "openwork_callback",
		"tenant_uuid":    tenantUUID,
		"suite_id":       templateID,
		"event_type":     infoType,
		"task_uuid":      saved.TaskUUID,
		"callback_key":   callbackKey,
		"idempotent_hit": !created,
	}).Info("openwork callback task enqueued")
	return nil
}

func (h *OpenWorkCallbackHandler) startWorkerLoop() {
	if h == nil || h.openWorkRepo == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			task, ok, err := h.openWorkRepo.ClaimNextCallbackTask(ctx)
			cancel()
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"module": "openwork_callback",
					"error":  err.Error(),
				}).Warn("openwork callback claim task failed")
				continue
			}
			if !ok || task == nil {
				continue
			}
			h.processClaimedCallbackTask(task)
		}
	}()
}

func (h *OpenWorkCallbackHandler) processClaimedCallbackTask(task *socialsvcmodel.WeComOpenCallbackTask) {
	if h == nil || task == nil || h.foundation == nil || h.openWorkRepo == nil {
		return
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	corpID := strings.TrimSpace(task.CorpID)
	agentID := strings.TrimSpace(task.AgentID)

	payload := map[string]any{
		"state": strings.TrimSpace(task.State),
	}
	if strings.TrimSpace(task.AuthCode) != "" {
		payload["auth_code_present"] = true
	}
	if task.Payload != nil {
		for k, v := range task.Payload {
			payload[k] = v
		}
	}
	eventTime := int64(0)
	if task.EventTime != nil {
		eventTime = task.EventTime.Unix()
	}
	event, binding, err := h.foundation.IngestEvent(ctx, socialsvc.OpenWorkEventIngestInput{
		TenantUUID:  strings.ToLower(strings.TrimSpace(task.TenantUUID)),
		SuiteID:     strings.TrimSpace(task.SuiteID),
		EventType:   strings.ToLower(strings.TrimSpace(task.EventType)),
		EventKey:    strings.TrimSpace(task.EventKey),
		SuiteTicket: strings.TrimSpace(task.SuiteTicket),
		CorpID:      corpID,
		AgentID:     agentID,
		EventTime:   eventTime,
		Payload:     payload,
	})
	if err != nil {
		_ = h.openWorkRepo.MarkCallbackTaskFailed(ctx, task.TaskUUID, err.Error(), nextCallbackRetryAt(task.AttemptCount), false)
		socialobs.RecordOpenWorkCallback("process_failed", task.EventType)
		logrus.WithFields(logrus.Fields{
			"module":      "openwork_callback",
			"task_uuid":   task.TaskUUID,
			"tenant_uuid": task.TenantUUID,
			"suite_id":    task.SuiteID,
			"event_type":  task.EventType,
			"error":       err.Error(),
		}).Warn("openwork callback ingest task failed")
		return
	}
	if corpID == "" && binding != nil {
		corpID = strings.TrimSpace(readModelString(binding, "CorpID"))
	}
	if agentID == "" && binding != nil {
		agentID = strings.TrimSpace(readModelString(binding, "AgentID"))
	}
	authCompleteResult := "skipped"
	authCompleteReason := "no_auth_code"
	idempotentConverged := false
	if strings.TrimSpace(task.AuthCode) != "" {
		authStart := time.Now()
		complete, completeErr := h.foundation.CompleteAuthorization(ctx, socialsvc.OpenWorkAuthorizeCompleteInput{
			TenantUUID: strings.ToLower(strings.TrimSpace(task.TenantUUID)),
			TemplateID: strings.TrimSpace(task.SuiteID),
			AuthCode:   strings.TrimSpace(task.AuthCode),
			SetDefault: false,
		})
		if completeErr != nil {
			if isInvalidAuthCodeError(completeErr) && h.shouldConvergeInvalidAuthCode(ctx, task, corpID) {
				idempotentConverged = true
				authCompleteResult = "success"
				authCompleteReason = "invalid_auth_code_converged"
				socialobs.RecordOpenWorkIdempotentHit("invalid_auth_code_converged", task.EventType)
			} else {
				authCompleteResult = "failed"
				authCompleteReason = normalizeAuthCompleteReason(completeErr.Error())
				forceReauthorize := isInvalidAuthCodeError(completeErr)
				_ = h.openWorkRepo.MarkCallbackTaskFailed(ctx, task.TaskUUID, completeErr.Error(), nextCallbackRetryAt(task.AttemptCount), forceReauthorize)
				socialobs.RecordOpenWorkAuthComplete("failed", authCompleteReason)
				socialobs.ObserveOpenWorkAuthCompleteLatency(float64(time.Since(authStart).Milliseconds()), "failed")
				logrus.WithFields(logrus.Fields{
					"module":      "openwork_callback",
					"task_uuid":   task.TaskUUID,
					"tenant_uuid": task.TenantUUID,
					"suite_id":    task.SuiteID,
					"event_type":  task.EventType,
					"error":       completeErr.Error(),
				}).Warn("openwork callback auto complete authorization failed")
				return
			}
		} else {
			authCompleteResult = "success"
			authCompleteReason = "ok"
			if corpID == "" {
				corpID = strings.TrimSpace(complete.CorpID)
			}
			if agentID == "" {
				agentID = strings.TrimSpace(complete.AgentID)
			}
			logrus.WithFields(logrus.Fields{
				"module":          "openwork_callback",
				"tenant_uuid":     task.TenantUUID,
				"suite_id":        task.SuiteID,
				"event_type":      task.EventType,
				"corp_id":         complete.CorpID,
				"agent_id":        complete.AgentID,
				"binding_uuid":    complete.BindingUUID,
				"binding_status":  complete.Status,
				"has_permanent":   strings.TrimSpace(complete.PermanentCode) != "",
				"channel_account": complete.ChannelAccountUUID,
			}).Info("openwork callback auto complete authorization success")
		}
		socialobs.RecordOpenWorkAuthComplete(authCompleteResult, authCompleteReason)
		socialobs.ObserveOpenWorkAuthCompleteLatency(float64(time.Since(authStart).Milliseconds()), authCompleteResult)
	}
	if strings.EqualFold(task.EventType, "suite_ticket") && strings.TrimSpace(task.SuiteTicket) != "" {
		if err := h.saveSuiteTicket(ctx, task.SuiteID, task.SuiteTicket); err != nil {
			_ = h.openWorkRepo.MarkCallbackTaskFailed(ctx, task.TaskUUID, err.Error(), nextCallbackRetryAt(task.AttemptCount), false)
			socialobs.RecordOpenWorkCallback("process_failed", task.EventType)
			return
		}
	}
	_ = h.openWorkRepo.MarkCallbackTaskSucceeded(ctx, task.TaskUUID, corpID, agentID, idempotentConverged || task.IdempotentHit)
	h.processStaffLiveCodeContactEvent(ctx, task, corpID, agentID)
	h.processCustomerGroupIncrementalTag(ctx, task, corpID, agentID)
	h.publishAuthStatus(ctx, task.SuiteID, task.EventType, corpID, agentID)
	socialobs.RecordOpenWorkCallback("processed", task.EventType)
	socialobs.ObserveOpenWorkAuthCompleteLatency(float64(time.Since(started).Milliseconds()), "total")
	logrus.WithFields(logrus.Fields{
		"module":       "openwork_callback",
		"task_uuid":    task.TaskUUID,
		"tenant_uuid":  task.TenantUUID,
		"suite_id":     task.SuiteID,
		"event_type":   task.EventType,
		"corp_id":      corpID,
		"agent_id":     agentID,
		"event_uuid":   readModelString(event, "EventUUID"),
		"binding_uuid": readModelString(binding, "BindingUUID"),
	}).Info("openwork callback task processed")
}

func (h *OpenWorkCallbackHandler) processStaffLiveCodeContactEvent(ctx context.Context, task *socialsvcmodel.WeComOpenCallbackTask, fallbackCorpID, fallbackAgentID string) {
	if h == nil || task == nil || h.deps == nil || h.deps.DB == nil {
		return
	}
	eventType := strings.ToLower(strings.TrimSpace(task.EventType))
	changeType := strings.ToLower(strings.TrimSpace(readPayloadString(task.Payload, "change_type")))
	if eventType != "change_external_contact" {
		return
	}
	if changeType == "del_external_contact" || changeType == "del_follow_user" {
		h.processStaffLiveCodeContactRemoved(ctx, task, fallbackCorpID, fallbackAgentID)
		return
	}
	if changeType != "add_external_contact" {
		return
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(task.TenantUUID))
	if tenantUUID == "" {
		return
	}
	state := strings.TrimSpace(readPayloadString(task.Payload, "state"))
	externalUserID := strings.TrimSpace(readPayloadString(task.Payload, "external_userid"))
	welcomeCode := strings.TrimSpace(readPayloadString(task.Payload, "welcome_code"))
	if state == "" || externalUserID == "" {
		h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
			TenantUUID:       tenantUUID,
			EventType:        eventType,
			ChangeType:       changeType,
			State:            state,
			ExternalUserID:   externalUserID,
			WelcomeCode:      welcomeCode,
			ProcessingStatus: "skipped",
			ProcessingError:  "missing state or external_userid",
		}, task.Payload)
		logrus.WithFields(logrus.Fields{
			"module":          "openwork_callback",
			"event_type":      eventType,
			"change_type":     changeType,
			"tenant_uuid":     tenantUUID,
			"state":           state,
			"external_userid": externalUserID,
		}).Warn("openwork callback staff live code skipped: missing state or external_userid")
		return
	}

	accountRepo := repository.NewAccountRepository(h.deps.DB)
	account, err := resolveCallbackChannelAccount(ctx, accountRepo, tenantUUID, strings.TrimSpace(task.CorpID), strings.TrimSpace(task.AgentID), fallbackCorpID, fallbackAgentID)
	if err != nil || account == nil {
		h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
			TenantUUID:       tenantUUID,
			EventType:        eventType,
			ChangeType:       changeType,
			State:            state,
			ExternalUserID:   externalUserID,
			WelcomeCode:      welcomeCode,
			ProcessingStatus: "failed",
			ProcessingError:  "channel account unresolved",
		}, task.Payload)
		logrus.WithFields(logrus.Fields{
			"module":          "openwork_callback",
			"event_type":      eventType,
			"change_type":     changeType,
			"tenant_uuid":     tenantUUID,
			"corp_id":         strings.TrimSpace(task.CorpID),
			"agent_id":        strings.TrimSpace(task.AgentID),
			"state":           state,
			"external_userid": externalUserID,
			"error":           errString(err),
		}).Warn("openwork callback staff live code skipped: channel account unresolved")
		return
	}

	bundle := acqrepo.NewBundle(h.deps.DB)
	items, listErr := bundle.StaffLiveCodes.List(ctx, tenantUUID, acqrepo.StaffLiveCodeListFilter{
		Status: "active",
		Limit:  2000,
	})
	if listErr != nil {
		h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
			TenantUUID:         tenantUUID,
			ChannelAccountUUID: strings.ToLower(strings.TrimSpace(account.AccountUUID)),
			EventType:          eventType,
			ChangeType:         changeType,
			State:              state,
			ExternalUserID:     externalUserID,
			WelcomeCode:        welcomeCode,
			ProcessingStatus:   "failed",
			ProcessingError:    listErr.Error(),
		}, task.Payload)
		logrus.WithFields(logrus.Fields{
			"module":      "openwork_callback",
			"tenant_uuid": tenantUUID,
			"state":       state,
			"error":       listErr.Error(),
		}).Warn("openwork callback staff live code list failed")
		return
	}
	channelAccountUUID := strings.ToLower(strings.TrimSpace(account.AccountUUID))
	var targetItem *acqdto.StaffLiveCodeContactApplyTarget
	for _, item := range items {
		if item == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.ChannelAccountUUID)) != channelAccountUUID {
			continue
		}
		if strings.TrimSpace(item.State) != state {
			continue
		}
		targetItem = &acqdto.StaffLiveCodeContactApplyTarget{
			StaffCodeUUID:      strings.ToLower(strings.TrimSpace(item.StaffCodeUUID)),
			TenantUUID:         strings.ToLower(strings.TrimSpace(item.TenantUUID)),
			ChannelAccountUUID: strings.ToLower(strings.TrimSpace(item.ChannelAccountUUID)),
			CorpTagIDs:         append([]string{}, item.CorpTagIDs...),
			WelcomeMode:        "",
			ContentBlocksRaw:   nil,
		}
		break
	}
	if targetItem == nil {
		h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
			TenantUUID:         tenantUUID,
			ChannelAccountUUID: channelAccountUUID,
			EventType:          eventType,
			ChangeType:         changeType,
			State:              state,
			ExternalUserID:     externalUserID,
			WelcomeCode:        welcomeCode,
			ProcessingStatus:   "skipped",
			ProcessingError:    "state not matched",
		}, task.Payload)
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"event_type":           eventType,
			"change_type":          changeType,
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": channelAccountUUID,
			"state":                state,
			"external_userid":      externalUserID,
		}).Warn("openwork callback staff live code skipped: state not matched")
		return
	}
	if cfg, cfgErr := bundle.StaffWelcomeConfigs.GetByStaffCodeUUID(ctx, tenantUUID, targetItem.StaffCodeUUID); cfgErr == nil && cfg != nil {
		targetItem.WelcomeMode = strings.ToLower(strings.TrimSpace(cfg.WelcomeMode))
		targetItem.ContentBlocksRaw = append([]byte{}, cfg.ContentBlocks...)
	}
	// 历史数据里 corp_tag_ids 可能被写成全小写；回调执行前按同步快照纠正为企微原始大小写。
	tagRecordRepo := repository.NewTagRecordRepository(h.deps.DB)
	if records, recErr := tagRecordRepo.ListByChannel(ctx, tenantUUID, channelAccountUUID, 500); recErr == nil && len(records) > 0 {
		targetItem.CorpTagIDs = canonicalizeCorpTagIDsBySnapshot(targetItem.CorpTagIDs, records)
	}

	resolver := &callbackGroupChatAccountResolver{
		accountRepo:  accountRepo,
		openworkRepo: h.openWorkRepo,
		platformRepo: h.platformRepo,
	}
	applier := acqdto.NewStaffLiveCodeContactEventService(resolver)
	applyResult, applyErr := applier.Apply(ctx, acqdto.StaffLiveCodeContactEventApplyRequest{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: channelAccountUUID,
		ExternalUserID:     externalUserID,
		WelcomeCode:        welcomeCode,
		Target:             targetItem,
	})
	h.upsertExternalContactOwner(ctx, task, tenantUUID, channelAccountUUID, state, externalUserID, applyResult)
	h.upsertLeadFromExternalContact(ctx, tenantUUID, account, externalUserID, applyResult)
	if applyErr != nil {
		// 负责人关系应与欢迎语发送结果解耦：即使欢迎语失败（如 41051），也要落当前 owner 关系。
		h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
			TenantUUID:         tenantUUID,
			StaffCodeUUID:      targetItem.StaffCodeUUID,
			ChannelAccountUUID: channelAccountUUID,
			EventType:          eventType,
			ChangeType:         changeType,
			State:              state,
			ExternalUserID:     externalUserID,
			WelcomeCode:        welcomeCode,
			ProcessingStatus:   "failed",
			ProcessingError:    applyErr.Error(),
		}, task.Payload)
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"event_type":           eventType,
			"change_type":          changeType,
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": channelAccountUUID,
			"state":                state,
			"external_userid":      externalUserID,
			"error":                applyErr.Error(),
		}).Warn("openwork callback staff live code apply failed")
		return
	}
	h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
		TenantUUID:         tenantUUID,
		StaffCodeUUID:      targetItem.StaffCodeUUID,
		ChannelAccountUUID: channelAccountUUID,
		EventType:          eventType,
		ChangeType:         changeType,
		State:              state,
		ExternalUserID:     externalUserID,
		WelcomeCode:        welcomeCode,
		ProcessingStatus:   "applied",
	}, task.Payload)
	logrus.WithFields(logrus.Fields{
		"module":               "openwork_callback",
		"event_type":           eventType,
		"change_type":          changeType,
		"tenant_uuid":          tenantUUID,
		"channel_account_uuid": channelAccountUUID,
		"state":                state,
		"external_userid":      externalUserID,
		"welcome_code":         welcomeCode != "",
	}).Info("openwork callback staff live code apply completed")
}

func (h *OpenWorkCallbackHandler) processStaffLiveCodeContactRemoved(ctx context.Context, task *socialsvcmodel.WeComOpenCallbackTask, fallbackCorpID, fallbackAgentID string) {
	if h == nil || task == nil || h.deps == nil || h.deps.DB == nil {
		return
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(task.TenantUUID))
	if tenantUUID == "" {
		return
	}
	changeType := strings.ToLower(strings.TrimSpace(readPayloadString(task.Payload, "change_type")))
	state := strings.TrimSpace(readPayloadString(task.Payload, "state"))
	externalUserID := strings.TrimSpace(readPayloadString(task.Payload, "external_userid"))
	if externalUserID == "" {
		h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
			TenantUUID:       tenantUUID,
			EventType:        "change_external_contact",
			ChangeType:       changeType,
			State:            state,
			ExternalUserID:   externalUserID,
			ProcessingStatus: "skipped",
			ProcessingError:  "missing external_userid",
		}, task.Payload)
		logrus.WithFields(logrus.Fields{
			"module":          "openwork_callback",
			"event_type":      "change_external_contact",
			"change_type":     changeType,
			"tenant_uuid":     tenantUUID,
			"state":           state,
			"external_userid": externalUserID,
		}).Warn("openwork callback staff live code remove skipped: missing external_userid")
		return
	}

	accountRepo := repository.NewAccountRepository(h.deps.DB)
	account, err := resolveCallbackChannelAccount(ctx, accountRepo, tenantUUID, strings.TrimSpace(task.CorpID), strings.TrimSpace(task.AgentID), fallbackCorpID, fallbackAgentID)
	if err != nil || account == nil {
		h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
			TenantUUID:       tenantUUID,
			EventType:        "change_external_contact",
			ChangeType:       changeType,
			State:            state,
			ExternalUserID:   externalUserID,
			ProcessingStatus: "failed",
			ProcessingError:  "channel account unresolved",
		}, task.Payload)
		logrus.WithFields(logrus.Fields{
			"module":          "openwork_callback",
			"event_type":      "change_external_contact",
			"change_type":     changeType,
			"tenant_uuid":     tenantUUID,
			"corp_id":         strings.TrimSpace(task.CorpID),
			"agent_id":        strings.TrimSpace(task.AgentID),
			"state":           state,
			"external_userid": externalUserID,
			"error":           errString(err),
		}).Warn("openwork callback staff live code remove skipped: channel account unresolved")
		return
	}
	h.recordStaffContactEvent(ctx, &acqmodel.StaffContactEvent{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: strings.ToLower(strings.TrimSpace(account.AccountUUID)),
		EventType:          "change_external_contact",
		ChangeType:         changeType,
		State:              state,
		ExternalUserID:     externalUserID,
		ProcessingStatus:   "removed",
	}, task.Payload)
	channelAccountUUID := strings.ToLower(strings.TrimSpace(account.AccountUUID))
	h.deleteExternalContactOwner(ctx, tenantUUID, channelAccountUUID, externalUserID)
	h.markLeadDisconnectedByExternalContact(ctx, tenantUUID, channelAccountUUID, externalUserID)

	logrus.WithFields(logrus.Fields{
		"module":               "openwork_callback",
		"event_type":           "change_external_contact",
		"change_type":          changeType,
		"tenant_uuid":          tenantUUID,
		"channel_account_uuid": channelAccountUUID,
		"state":                state,
		"external_userid":      externalUserID,
		"corp_id":              strings.TrimSpace(task.CorpID),
		"agent_id":             strings.TrimSpace(task.AgentID),
	}).Info("openwork callback staff live code relation removed")
}

func (h *OpenWorkCallbackHandler) upsertExternalContactOwner(ctx context.Context, task *socialsvcmodel.WeComOpenCallbackTask, tenantUUID, channelAccountUUID, state, externalUserID string, applyResult *acqdto.StaffLiveCodeContactApplyResult) {
	if h == nil || h.deps == nil || h.deps.DB == nil {
		return
	}
	operatorUserID := strings.TrimSpace(readPayloadString(task.Payload, "operator_userid"))
	if operatorUserID == "" {
		operatorUserID = strings.TrimSpace(readPayloadString(task.Payload, "userid"))
	}
	if applyResult != nil && strings.TrimSpace(applyResult.OperatorUserID) != "" {
		operatorUserID = strings.TrimSpace(applyResult.OperatorUserID)
	}
	if operatorUserID == "" {
		return
	}
	now := time.Now().UTC()
	ownerMemberID := h.resolveMainMemberIDByExternalUserID(ctx, tenantUUID, channelAccountUUID, operatorUserID)
	payloadRaw, _ := json.Marshal(task.Payload)
	record := &acqmodel.ExternalContactOwner{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: channelAccountUUID,
		ExternalUserID:     externalUserID,
		OwnerWeComUserID:   operatorUserID,
		OwnerMemberID:      ownerMemberID,
		Source:             "callback",
		State:              strings.TrimSpace(state),
		LastEventType:      strings.ToLower(strings.TrimSpace(task.EventType)),
		LastChangeType:     strings.ToLower(strings.TrimSpace(readPayloadString(task.Payload, "change_type"))),
		LastEventPayload:   datatypes.JSON(payloadRaw),
		LastEventAt:        &now,
		UpdatedAt:          now,
	}
	if len(record.LastEventPayload) == 0 {
		record.LastEventPayload = datatypes.JSON([]byte(`{}`))
	}
	if err := h.deps.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "tenant_uuid"},
				{Name: "channel_account_uuid"},
				{Name: "external_userid"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"owner_wecom_userid": operatorUserID,
				"owner_member_id":    ownerMemberID,
				"source":             "callback",
				"state":              strings.TrimSpace(state),
				"last_event_type":    strings.ToLower(strings.TrimSpace(task.EventType)),
				"last_change_type":   strings.ToLower(strings.TrimSpace(readPayloadString(task.Payload, "change_type"))),
				"last_event_payload": record.LastEventPayload,
				"last_event_at":      now,
				"updated_at":         now,
				"version":            gorm.Expr("COALESCE(acquisition_external_contact_owners.version, 0) + 1"),
			}),
		}).
		Create(record).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": channelAccountUUID,
			"external_userid":      externalUserID,
			"owner_wecom_userid":   operatorUserID,
			"error":                err.Error(),
		}).Warn("openwork callback owner relation upsert failed")
		return
	}
	logrus.WithFields(logrus.Fields{
		"module":               "openwork_callback",
		"tenant_uuid":          tenantUUID,
		"channel_account_uuid": channelAccountUUID,
		"external_userid":      externalUserID,
		"owner_wecom_userid":   operatorUserID,
		"owner_member_id":      ownerMemberID,
	}).Info("openwork callback owner relation upsert completed")
}

func (h *OpenWorkCallbackHandler) upsertLeadFromExternalContact(ctx context.Context, tenantUUID string, account *socialmodel.ChannelAccount, externalUserID string, applyResult *acqdto.StaffLiveCodeContactApplyResult) {
	if h == nil || h.deps == nil || h.deps.DB == nil || account == nil {
		return
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID := strings.ToLower(strings.TrimSpace(account.AccountUUID))
	externalUserID = strings.TrimSpace(externalUserID)
	if tenantUUID == "" || channelAccountUUID == "" || externalUserID == "" {
		return
	}

	displayName := externalUserID
	if applyResult != nil && strings.TrimSpace(applyResult.ExternalDisplayName) != "" {
		displayName = strings.TrimSpace(applyResult.ExternalDisplayName)
	}
	ownerWeComUserID := ""
	if applyResult != nil {
		ownerWeComUserID = strings.TrimSpace(applyResult.OperatorUserID)
	}
	ownerMemberID := ""
	if ownerWeComUserID != "" {
		ownerMemberID = h.resolveMainMemberIDByExternalUserID(ctx, tenantUUID, channelAccountUUID, ownerWeComUserID)
	}

	var leadUUID string
	activityTable := leadmodel.LeadActivity{}.TableName()
	if err := h.deps.DB.WithContext(ctx).
		Table(activityTable).
		Select("lead_uuid").
		Where("tenant_uuid = ? AND activity_type = ? AND payload ->> 'source_account_uuid' = ? AND payload ->> 'external_wechat_id' = ?",
			tenantUUID, leadmodel.LeadActivityTypeSyncTrace, channelAccountUUID, externalUserID).
		Order("updated_at DESC").
		Limit(1).
		Scan(&leadUUID).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": channelAccountUUID,
			"external_userid":      externalUserID,
			"error":                err.Error(),
		}).Warn("openwork callback lead lookup by external identity failed")
	}

	now := time.Now().UTC()
	if strings.TrimSpace(leadUUID) == "" {
		lead := &leadmodel.Lead{
			TenantUUID:        tenantUUID,
			DisplayName:       displayName,
			Status:            leadmodel.LeadStatusAssigned,
			OwnerUserUUID:     ownerMemberID,
			SourceChannel:     strings.ToLower(strings.TrimSpace(account.ChannelCode)),
			SourceAppType:     strings.ToLower(strings.TrimSpace(account.AppType)),
			SourceAccountUUID: &channelAccountUUID,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := h.deps.DB.WithContext(ctx).Create(lead).Error; err != nil {
			logrus.WithFields(logrus.Fields{
				"module":               "openwork_callback",
				"tenant_uuid":          tenantUUID,
				"channel_account_uuid": channelAccountUUID,
				"external_userid":      externalUserID,
				"error":                err.Error(),
			}).Warn("openwork callback lead create failed")
			return
		}
		leadUUID = strings.TrimSpace(lead.LeadUUID)
	}
	if leadUUID == "" {
		return
	}

	updateFields := map[string]any{
		"updated_at": now,
	}
	if displayName != "" {
		updateFields["display_name"] = displayName
	}
	if ownerMemberID != "" {
		updateFields["owner_user_uuid"] = ownerMemberID
		updateFields["status"] = leadmodel.LeadStatusAssigned
	}
	if err := h.deps.DB.WithContext(ctx).
		Model(&leadmodel.Lead{}).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Updates(updateFields).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": channelAccountUUID,
			"external_userid":      externalUserID,
			"lead_uuid":            leadUUID,
			"error":                err.Error(),
		}).Warn("openwork callback lead update failed")
	}

	payload := datatypes.JSONMap{
		"source_account_uuid":   channelAccountUUID,
		"external_wechat_id":    externalUserID,
		"owner_external_userid": ownerWeComUserID,
		"owner_user_uuid":       ownerMemberID,
		"display_name":          displayName,
	}
	activity := &leadmodel.LeadActivity{
		TenantUUID:   tenantUUID,
		LeadUUID:     leadUUID,
		ActivityType: leadmodel.LeadActivityTypeSyncTrace,
		Payload:      payload,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.deps.DB.WithContext(ctx).Create(activity).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": channelAccountUUID,
			"external_userid":      externalUserID,
			"lead_uuid":            leadUUID,
			"error":                err.Error(),
		}).Warn("openwork callback lead sync_trace upsert failed")
		return
	}
	logrus.WithFields(logrus.Fields{
		"module":               "openwork_callback",
		"tenant_uuid":          tenantUUID,
		"channel_account_uuid": channelAccountUUID,
		"external_userid":      externalUserID,
		"lead_uuid":            leadUUID,
		"owner_user_uuid":      ownerMemberID,
	}).Info("openwork callback lead upsert completed")
}

func (h *OpenWorkCallbackHandler) deleteExternalContactOwner(ctx context.Context, tenantUUID, channelAccountUUID, externalUserID string) {
	if h == nil || h.deps == nil || h.deps.DB == nil {
		return
	}
	if err := h.deps.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_account_uuid = ? AND external_userid = ?", tenantUUID, channelAccountUUID, strings.TrimSpace(externalUserID)).
		Delete(&acqmodel.ExternalContactOwner{}).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": channelAccountUUID,
			"external_userid":      externalUserID,
			"error":                err.Error(),
		}).Warn("openwork callback owner relation delete failed")
	}
}

func (h *OpenWorkCallbackHandler) markLeadDisconnectedByExternalContact(ctx context.Context, tenantUUID, channelAccountUUID, externalUserID string) {
	if h == nil || h.deps == nil || h.deps.DB == nil {
		return
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	externalUserID = strings.TrimSpace(externalUserID)
	if tenantUUID == "" || channelAccountUUID == "" || externalUserID == "" {
		return
	}
	var leadUUID string
	activityTable := leadmodel.LeadActivity{}.TableName()
	if err := h.deps.DB.WithContext(ctx).
		Table(activityTable).
		Select("lead_uuid").
		Where("tenant_uuid = ? AND activity_type = ? AND payload ->> 'source_account_uuid' = ? AND payload ->> 'external_wechat_id' = ?",
			tenantUUID, leadmodel.LeadActivityTypeSyncTrace, channelAccountUUID, externalUserID).
		Order("updated_at DESC").
		Limit(1).
		Scan(&leadUUID).Error; err != nil {
		return
	}
	leadUUID = strings.TrimSpace(leadUUID)
	if leadUUID == "" {
		return
	}
	now := time.Now().UTC()
	if err := h.deps.DB.WithContext(ctx).
		Model(&leadmodel.Lead{}).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Updates(map[string]any{
			"status":          leadmodel.LeadStatusDisconnected,
			"owner_user_uuid": "",
			"updated_at":      now,
		}).Error; err != nil {
		return
	}
	logrus.WithFields(logrus.Fields{
		"module":               "openwork_callback",
		"tenant_uuid":          tenantUUID,
		"channel_account_uuid": channelAccountUUID,
		"external_userid":      externalUserID,
		"lead_uuid":            leadUUID,
		"status":               leadmodel.LeadStatusDisconnected,
	}).Info("openwork callback lead marked disconnected")
	h.markActiveOpportunitiesRiskForDisconnectedLead(ctx, tenantUUID, leadUUID, channelAccountUUID, externalUserID)
}

func (h *OpenWorkCallbackHandler) markActiveOpportunitiesRiskForDisconnectedLead(ctx context.Context, tenantUUID, leadUUID, channelAccountUUID, externalUserID string) {
	if h == nil || h.deps == nil || h.deps.DB == nil {
		return
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return
	}
	var items []oppmodel.OpportunityRecord
	if err := h.deps.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ? AND stage IN ?", tenantUUID, leadUUID, []string{oppmodel.StageOpen, oppmodel.StageQualified, oppmodel.StageProposal, oppmodel.StageNegotiation}).
		Find(&items).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"module":      "openwork_callback",
			"tenant_uuid": tenantUUID,
			"lead_uuid":   leadUUID,
			"error":       err.Error(),
		}).Warn("openwork callback opportunity risk query failed")
		return
	}
	for _, item := range items {
		actor := firstValidUUID(item.UpdatedBy, item.CreatedBy, item.OwnerUserUUID)
		if actor == "" {
			logrus.WithFields(logrus.Fields{
				"module":           "openwork_callback",
				"tenant_uuid":      tenantUUID,
				"lead_uuid":        leadUUID,
				"opportunity_uuid": item.OpportunityUUID,
			}).Warn("openwork callback opportunity risk skipped: no valid actor uuid")
			continue
		}
		if err := markOpportunityRiskFlagTx(ctx, h.deps.DB, &item, actor, map[string]any{
			"flag":                 "disconnected",
			"source":               "openwork_callback",
			"lead_uuid":            leadUUID,
			"channel_account_uuid": strings.ToLower(strings.TrimSpace(channelAccountUUID)),
			"external_userid":      strings.TrimSpace(externalUserID),
		}); err != nil {
			logrus.WithFields(logrus.Fields{
				"module":           "openwork_callback",
				"tenant_uuid":      tenantUUID,
				"lead_uuid":        leadUUID,
				"opportunity_uuid": item.OpportunityUUID,
				"error":            err.Error(),
			}).Warn("openwork callback opportunity risk mark failed")
			continue
		}
		logrus.WithFields(logrus.Fields{
			"module":           "openwork_callback",
			"tenant_uuid":      tenantUUID,
			"lead_uuid":        leadUUID,
			"opportunity_uuid": item.OpportunityUUID,
			"risk_flag":        "disconnected",
		}).Info("openwork callback opportunity marked risk")
	}
}

func markOpportunityRiskFlagTx(ctx context.Context, db *gorm.DB, item *oppmodel.OpportunityRecord, actor string, payload map[string]any) error {
	if db == nil || item == nil {
		return nil
	}
	now := time.Now().UTC()
	flags := parseStringJSONList(item.RiskFlags)
	exists := false
	for _, flag := range flags {
		if flag == "disconnected" {
			exists = true
			break
		}
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if !exists {
			flags = append(flags, "disconnected")
			raw, _ := json.Marshal(flags)
			if err := tx.Model(&oppmodel.OpportunityRecord{}).
				Where("tenant_uuid = ? AND opportunity_uuid = ?", item.TenantUUID, item.OpportunityUUID).
				Updates(map[string]any{
					"risk_flags": datatypes.JSON(raw),
					"updated_by": actor,
					"updated_at": now,
				}).Error; err != nil {
				return err
			}
		}
		rawPayload, _ := json.Marshal(payload)
		activity := &oppmodel.OpportunityActivity{
			ActivityUUID:     uuid.NewString(),
			TenantUUID:       item.TenantUUID,
			OpportunityUUID:  item.OpportunityUUID,
			ActivityType:     oppmodel.ActivityRiskFlag,
			ToStage:          item.Stage,
			Payload:          datatypes.JSON(rawPayload),
			OperatorUserUUID: actor,
			CreatedAt:        now,
		}
		return tx.Create(activity).Error
	})
}

func parseStringJSONList(raw datatypes.JSON) []string {
	var out []string
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func firstValidUUID(values ...string) string {
	for _, value := range values {
		parsed, err := uuid.Parse(strings.TrimSpace(value))
		if err == nil && parsed != uuid.Nil {
			return strings.ToLower(parsed.String())
		}
	}
	return ""
}

func (h *OpenWorkCallbackHandler) resolveMainMemberIDByExternalUserID(ctx context.Context, tenantUUID, channelAccountUUID, externalUserID string) string {
	if h == nil || h.deps == nil || h.deps.DB == nil {
		return ""
	}
	var row struct {
		MainMemberID string `gorm:"column:main_member_id"`
	}
	if err := h.deps.DB.WithContext(ctx).
		Table(orgmodel.MemberBinding{}.TableName()).
		Where("tenant_uuid = ? AND channel_account_uuid = ? AND external_member_id = ?", tenantUUID, channelAccountUUID, strings.TrimSpace(externalUserID)).
		Order("updated_at DESC").
		Limit(1).
		Select("main_member_id").
		Scan(&row).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(row.MainMemberID)
}

func (h *OpenWorkCallbackHandler) recordStaffContactEvent(ctx context.Context, evt *acqmodel.StaffContactEvent, payload any) {
	if h == nil || h.deps == nil || h.deps.DB == nil || evt == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil || len(raw) == 0 {
		evt.Payload = datatypes.JSON([]byte(`{}`))
	} else {
		evt.Payload = datatypes.JSON(raw)
	}
	// 审计表里 staff_code_uuid/channel_account_uuid 为 uuid 列，回调缺失时不能写空字符串。
	evt.StaffCodeUUID = normalizeUUIDOrZero(evt.StaffCodeUUID)
	evt.ChannelAccountUUID = normalizeUUIDOrZero(evt.ChannelAccountUUID)
	if err := h.deps.DB.WithContext(ctx).Create(evt).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"module":        "openwork_callback",
			"tenant_uuid":   strings.TrimSpace(evt.TenantUUID),
			"event_type":    strings.TrimSpace(evt.EventType),
			"change_type":   strings.TrimSpace(evt.ChangeType),
			"external_user": strings.TrimSpace(evt.ExternalUserID),
			"error":         err.Error(),
		}).Warn("openwork callback staff contact event persist failed")
	}
}

func normalizeUUIDOrZero(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "00000000-0000-0000-0000-000000000000"
	}
	return v
}

func (h *OpenWorkCallbackHandler) processCustomerGroupIncrementalTag(ctx context.Context, task *socialsvcmodel.WeComOpenCallbackTask, fallbackCorpID, fallbackAgentID string) {
	if h == nil || task == nil || h.deps == nil || h.deps.DB == nil {
		return
	}
	eventType := strings.ToLower(strings.TrimSpace(task.EventType))
	if eventType == "" {
		return
	}
	changeType := strings.ToLower(strings.TrimSpace(readPayloadString(task.Payload, "change_type")))
	joined := changeType == "add_external_chat" || changeType == "add_external_contact" || changeType == "add_member"
	if !joined {
		return
	}
	chatID := strings.TrimSpace(readPayloadString(task.Payload, "chat_id"))
	externalUserID := strings.TrimSpace(readPayloadString(task.Payload, "external_userid"))
	if chatID == "" || externalUserID == "" {
		return
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(task.TenantUUID))
	if tenantUUID == "" {
		return
	}
	accountRepo := repository.NewAccountRepository(h.deps.DB)
	account, err := resolveCallbackChannelAccount(ctx, accountRepo, tenantUUID, strings.TrimSpace(task.CorpID), strings.TrimSpace(task.AgentID), fallbackCorpID, fallbackAgentID)
	if err != nil || account == nil {
		logrus.WithFields(logrus.Fields{
			"module":          "openwork_callback",
			"event_type":      eventType,
			"change_type":     changeType,
			"tenant_uuid":     tenantUUID,
			"corp_id":         strings.TrimSpace(task.CorpID),
			"agent_id":        strings.TrimSpace(task.AgentID),
			"chat_id":         chatID,
			"external_userid": externalUserID,
			"error":           errString(err),
		}).Warn("openwork callback incremental tag skipped: channel account unresolved")
		return
	}
	bundle := acqrepo.NewBundle(h.deps.DB)
	resolver := &callbackGroupChatAccountResolver{
		accountRepo:  accountRepo,
		openworkRepo: h.openWorkRepo,
		platformRepo: h.platformRepo,
	}
	groupSvc := acqdto.NewGroupLiveCodeService(bundle.GroupLiveCodes, bundle.GroupChatSnapshots, resolver)
	if applyErr := groupSvc.ApplyMemberCorpTagsForJoinEvent(ctx, acqdto.GroupLiveCodeIncrementalTagRequest{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: strings.ToLower(strings.TrimSpace(account.AccountUUID)),
		ChatID:             chatID,
		ExternalUserID:     externalUserID,
	}); applyErr != nil {
		logrus.WithFields(logrus.Fields{
			"module":               "openwork_callback",
			"event_type":           eventType,
			"change_type":          changeType,
			"tenant_uuid":          tenantUUID,
			"channel_account_uuid": strings.TrimSpace(account.AccountUUID),
			"chat_id":              chatID,
			"external_userid":      externalUserID,
			"error":                applyErr.Error(),
		}).Warn("openwork callback incremental tag apply failed")
		return
	}
	logrus.WithFields(logrus.Fields{
		"module":               "openwork_callback",
		"event_type":           eventType,
		"change_type":          changeType,
		"tenant_uuid":          tenantUUID,
		"channel_account_uuid": strings.TrimSpace(account.AccountUUID),
		"chat_id":              chatID,
		"external_userid":      externalUserID,
	}).Info("openwork callback incremental tag apply completed")
}

func canonicalizeCorpTagIDsBySnapshot(tagIDs []string, records []socialmodel.SyncTagRecord) []string {
	if len(tagIDs) == 0 || len(records) == 0 {
		return tagIDs
	}
	lookup := make(map[string]string, len(records))
	for _, record := range records {
		// 员工标签快照(source=wecom_staff)与外部联系人标签不是同一域，这里只用外部联系人标签快照纠正。
		if strings.TrimSpace(strings.ToLower(record.Source)) != "wecom" {
			continue
		}
		raw := strings.TrimSpace(record.RemoteTagID)
		if raw == "" {
			continue
		}
		lookup[strings.ToLower(raw)] = raw
	}
	out := make([]string, 0, len(tagIDs))
	seen := make(map[string]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		clean := strings.TrimSpace(tagID)
		if clean == "" {
			continue
		}
		if mapped, ok := lookup[strings.ToLower(clean)]; ok && strings.TrimSpace(mapped) != "" {
			clean = strings.TrimSpace(mapped)
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func resolveCallbackChannelAccount(
	ctx context.Context,
	accountRepo *repository.AccountRepository,
	tenantUUID, corpID, agentID, fallbackCorpID, fallbackAgentID string,
) (*socialmodel.ChannelAccount, error) {
	if accountRepo == nil {
		return nil, errors.New("account repository unavailable")
	}
	candidateCorpID := strings.TrimSpace(corpID)
	if candidateCorpID == "" {
		candidateCorpID = strings.TrimSpace(fallbackCorpID)
	}
	candidateAgentID := strings.TrimSpace(agentID)
	if candidateAgentID == "" {
		candidateAgentID = strings.TrimSpace(fallbackAgentID)
	}
	if candidateCorpID != "" && candidateAgentID != "" {
		if exact, err := accountRepo.FindByWeComIdentity(ctx, tenantUUID, candidateCorpID, candidateAgentID); err == nil && exact != nil {
			return exact, nil
		}
	}
	accounts, err := accountRepo.ListByTenant(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	for _, item := range accounts {
		if item == nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(item.ChannelCode)) != "wechat" {
			continue
		}
		appType := strings.ToLower(strings.TrimSpace(item.AppType))
		if appType != "openwork" && appType != "wecom" {
			continue
		}
		credCorpID := strings.TrimSpace(fmt.Sprintf("%v", item.Credentials["corp_id"]))
		if credCorpID == "" {
			credCorpID = strings.TrimSpace(fmt.Sprintf("%v", item.Credentials["auth_corp_id"]))
		}
		if candidateCorpID != "" && credCorpID != "" && !strings.EqualFold(credCorpID, candidateCorpID) {
			continue
		}
		return item, nil
	}
	return nil, errors.New("matching channel account not found")
}

func readPayloadString(payload datatypes.JSONMap, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", raw))
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func readExternalUserIDFromRaw(raw any) string {
	if raw == nil {
		return ""
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	payload := map[string]any{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return ""
	}
	list, _ := payload["MemChangeList"].([]any)
	for _, item := range list {
		row, ok := item.(map[string]any)
		if !ok || row == nil {
			continue
		}
		v := strings.TrimSpace(fmt.Sprintf("%v", row["Item"]))
		if v != "" {
			return v
		}
	}
	return ""
}

type callbackGroupChatAccountResolver struct {
	accountRepo  *repository.AccountRepository
	openworkRepo *repository.OpenWorkFoundationRepository
	platformRepo *repository.ChannelPlatformSettingRepository
}

func (r *callbackGroupChatAccountResolver) ResolveDefaultChannelAccount(ctx context.Context, tenantUUID, channel, appType string) (string, error) {
	if r == nil || r.accountRepo == nil {
		return "", repository.ErrAccountNotFound
	}
	accounts, err := r.accountRepo.ListByTenant(ctx, tenantUUID)
	if err != nil {
		return "", err
	}
	for _, acc := range accounts {
		if acc == nil {
			continue
		}
		if channel != "" && !strings.EqualFold(strings.TrimSpace(acc.ChannelCode), channel) {
			continue
		}
		if appType != "" && !strings.EqualFold(strings.TrimSpace(acc.AppType), appType) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(acc.Status), socialmodel.ChannelAccountStatusConnected) {
			return strings.TrimSpace(acc.AccountUUID), nil
		}
	}
	return "", repository.ErrAccountNotFound
}

func (r *callbackGroupChatAccountResolver) GetChannelAccount(ctx context.Context, tenantUUID, accountUUID string) (*acqdto.GroupChatAccountProfile, error) {
	if r == nil || r.accountRepo == nil {
		return nil, repository.ErrAccountNotFound
	}
	acc, err := r.accountRepo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	return &acqdto.GroupChatAccountProfile{
		ChannelCode: strings.ToLower(strings.TrimSpace(acc.ChannelCode)),
		AppType:     strings.ToLower(strings.TrimSpace(acc.AppType)),
	}, nil
}

func (r *callbackGroupChatAccountResolver) GetChannelAccountCredentials(ctx context.Context, tenantUUID, accountUUID string) (map[string]string, error) {
	if r == nil || r.accountRepo == nil {
		return nil, repository.ErrAccountNotFound
	}
	acc, err := r.accountRepo.GetByAccountUUID(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range acc.Credentials {
		key := strings.TrimSpace(k)
		if key == "" || v == nil {
			continue
		}
		out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	if r != nil && r.openworkRepo != nil {
		binding, bindErr := r.openworkRepo.ResolveBindingByChannelAccount(ctx, tenantUUID, accountUUID)
		if bindErr == nil && binding != nil && strings.EqualFold(strings.TrimSpace(binding.Status), socialmodel.WeComAuthBindingStatusActive) {
			applyCredentialIfMissing(out, "corp_id", strings.TrimSpace(binding.CorpID))
			applyCredentialIfMissing(out, "auth_corp_id", strings.TrimSpace(binding.CorpID))
			applyCredentialIfMissing(out, "permanent_code", strings.TrimSpace(binding.PermanentCode))
			applyCredentialIfMissing(out, "template_id", strings.TrimSpace(binding.SuiteID))
			applyCredentialIfMissing(out, "suite_id", strings.TrimSpace(binding.SuiteID))
			applyCredentialIfMissing(out, "agent_id", strings.TrimSpace(binding.AgentID))
			applyCredentialIfMissing(out, "template_ticket", strings.TrimSpace(binding.SuiteTicket))
			applyCredentialIfMissing(out, "suite_ticket", strings.TrimSpace(binding.SuiteTicket))
		}
	}
	if r != nil && r.platformRepo != nil {
		record, recErr := r.platformRepo.GetByChannelProvider(ctx, "wechat", "openwork")
		if recErr == nil && record != nil && record.Config != nil {
			applyCredentialIfMissing(out, "token", strings.TrimSpace(readMapString(record.Config, "token")))
			applyCredentialIfMissing(out, "aes_key", strings.TrimSpace(readMapString(record.Config, "aes_key")))
			applyCredentialIfMissing(out, "template_secret", strings.TrimSpace(readMapString(record.Config, "template_secret")))
			applyCredentialIfMissing(out, "provider_corpid", strings.TrimSpace(readMapString(record.Config, "provider_corpid")))
			applyCredentialIfMissing(out, "provider_secret", strings.TrimSpace(readMapString(record.Config, "provider_secret")))
		}
	}
	return out, nil
}

func applyCredentialIfMissing(target map[string]string, key, value string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if strings.TrimSpace(value) == "" {
		return
	}
	if strings.TrimSpace(target[key]) != "" {
		return
	}
	target[key] = strings.TrimSpace(value)
}

func (h *OpenWorkCallbackHandler) shouldConvergeInvalidAuthCode(ctx context.Context, task *socialsvcmodel.WeComOpenCallbackTask, fallbackCorpID string) bool {
	if h == nil || h.openWorkRepo == nil || task == nil {
		return false
	}
	corpID := strings.TrimSpace(task.CorpID)
	if corpID == "" {
		corpID = strings.TrimSpace(fallbackCorpID)
	}
	if corpID != "" {
		binding, err := h.openWorkRepo.GetActiveBindingBySuiteAndCorp(ctx, task.TenantUUID, task.SuiteID, corpID)
		if err == nil && binding != nil {
			return true
		}
	}
	bindings, err := h.openWorkRepo.ListBindingsBySuite(ctx, task.TenantUUID, task.SuiteID, 3)
	if err != nil {
		return false
	}
	for _, b := range bindings {
		if b == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(b.Status), socialsvcmodel.WeComAuthBindingStatusActive) {
			return true
		}
	}
	return false
}

func nextCallbackRetryAt(attempt int) *time.Time {
	backoff := 3 * time.Second
	switch {
	case attempt >= 3:
		backoff = 60 * time.Second
	case attempt == 2:
		backoff = 15 * time.Second
	case attempt <= 1:
		backoff = 5 * time.Second
	}
	next := time.Now().UTC().Add(backoff)
	return &next
}

func isInvalidAuthCodeError(err error) bool {
	if err == nil {
		return false
	}
	raw := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(raw, "40078") || strings.Contains(raw, "invalid auth_code")
}

func normalizeAuthCompleteReason(reason string) string {
	raw := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case strings.Contains(raw, "40078"), strings.Contains(raw, "invalid auth_code"):
		return "invalid_auth_code"
	case strings.Contains(raw, "context canceled"):
		return "context_canceled"
	default:
		return "failed"
	}
}

func (h *OpenWorkCallbackHandler) resolveTenantUUID() string {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	if h != nil && h.deps != nil && h.deps.Config != nil && h.deps.Config.GRPCUpstream != nil {
		if candidate := strings.TrimSpace(h.deps.Config.GRPCUpstream.TenantUUID); candidate != "" {
			tenantUUID = candidate
		}
	}
	return tenantUUID
}

func buildOpenWorkCallbackKey(infoType, authCode, msgSignature, timestamp, nonce, eventKey string) string {
	infoType = strings.ToLower(strings.TrimSpace(infoType))
	authCode = strings.TrimSpace(authCode)
	if (infoType == "create_auth" || infoType == "change_auth") && authCode != "" {
		return "auth_code:" + authCode
	}
	msgSignature = strings.TrimSpace(msgSignature)
	timestamp = strings.TrimSpace(timestamp)
	nonce = strings.TrimSpace(nonce)
	if msgSignature != "" && timestamp != "" && nonce != "" {
		return "callback_signature:" + msgSignature + ":" + timestamp + ":" + nonce
	}
	eventKey = strings.TrimSpace(eventKey)
	if eventKey != "" {
		return "event_key:" + eventKey
	}
	return "fallback:" + infoType + ":" + time.Now().UTC().Format(time.RFC3339Nano)
}

func buildOpenWorkEventKey(tenantUUID, templateID, infoType, authCode, msgSignature, timestamp, nonce string, ev any) string {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	templateID = strings.TrimSpace(templateID)
	infoType = strings.ToLower(strings.TrimSpace(infoType))
	authCode = strings.TrimSpace(authCode)
	if (infoType == "create_auth" || infoType == "change_auth") && authCode != "" {
		return fmt.Sprintf("%s:%s:%s:auth_code:%s", tenantUUID, templateID, infoType, authCode)
	}
	msgSignature = strings.TrimSpace(msgSignature)
	timestamp = strings.TrimSpace(timestamp)
	nonce = strings.TrimSpace(nonce)
	if msgSignature != "" && timestamp != "" && nonce != "" {
		return fmt.Sprintf("%s:%s:%s:sig:%s:%s:%s", tenantUUID, templateID, infoType, msgSignature, timestamp, nonce)
	}
	raw := marshalLogObject(ev)
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return fmt.Sprintf("%s:%s:%s:hash:%s", tenantUUID, templateID, infoType, hex.EncodeToString(sum[:]))
}

func readEventStringField(ev any, keys ...string) string {
	value := reflect.Indirect(reflect.ValueOf(ev))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return ""
	}
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		field := value.FieldByName(key)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}
		switch field.Kind() {
		case reflect.String:
			if text := strings.TrimSpace(field.String()); text != "" {
				return text
			}
		default:
			text := strings.TrimSpace(fmt.Sprintf("%v", field.Interface()))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func readEventUnixField(ev any, keys ...string) int64 {
	value := reflect.Indirect(reflect.ValueOf(ev))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return 0
	}
	for _, key := range keys {
		field := value.FieldByName(key)
		if !field.IsValid() || !field.CanInterface() {
			continue
		}
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if out := field.Int(); out > 0 {
				return out
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if out := field.Uint(); out > 0 {
				return int64(out)
			}
		case reflect.String:
			raw := strings.TrimSpace(field.String())
			if raw == "" {
				continue
			}
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}

func readModelString(binding any, key string) string {
	if binding == nil || strings.TrimSpace(key) == "" {
		return ""
	}
	val := reflect.Indirect(reflect.ValueOf(binding))
	if !val.IsValid() || val.Kind() != reflect.Struct {
		return ""
	}
	field := val.FieldByName(key)
	if !field.IsValid() || !field.CanInterface() {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", field.Interface()))
}
