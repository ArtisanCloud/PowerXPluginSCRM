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
	socialsvcmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
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
		if err := h.enqueueCallbackTask(c.Request.Context(), eventTemplateID, infoType, ev, msgSignature, timestamp, nonce); err != nil {
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

func (h *OpenWorkCallbackHandler) enqueueCallbackTask(ctx context.Context, templateID, infoType string, ev openworkmodel.IEvent, msgSignature, timestamp, nonce string) error {
	if h == nil || h.openWorkRepo == nil || ev == nil {
		return nil
	}
	tenantUUID := h.resolveTenantUUID()
	corpID := strings.TrimSpace(readEventStringField(ev, "AuthCorpID", "CorpID"))
	agentID := strings.TrimSpace(readEventStringField(ev, "AgentID"))
	authCode := strings.TrimSpace(readEventStringField(ev, "AuthCode"))
	state := readEventStringField(ev, "State")
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
