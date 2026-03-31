package webhooks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	kernelmodels "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/models"
	openwork "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork"
	openworkmodel "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork/server/models"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type OpenWorkCallbackHandler struct {
	platformRepo *repository.ChannelPlatformSettingRepository
}

func NewOpenWorkCallbackHandler(platformRepo *repository.ChannelPlatformSettingRepository) *OpenWorkCallbackHandler {
	return &OpenWorkCallbackHandler{platformRepo: platformRepo}
}

func (h *OpenWorkCallbackHandler) Handle(c *gin.Context) {
	if h == nil || h.platformRepo == nil {
		c.String(http.StatusServiceUnavailable, "openwork callback unavailable")
		return
	}

	templateID := strings.TrimSpace(c.Param("suite_id"))

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

	app, err := openwork.NewOpenWork(&openwork.UserConfig{
		AppID:  cfgTemplateID,
		Token:  token,
		AESKey: aesKey,
	})
	if err != nil {
		c.String(http.StatusBadRequest, "openwork init failed: "+err.Error())
		return
	}

	httpResp, err := app.Server.Notify(c.Request, func(_ *kernelmodels.Callback, ev openworkmodel.IEvent, _ interface{}) interface{} {
		if ticketEvent, ok := ev.(*openworkmodel.EventSuiteTicket); ok {
			eventTemplateID := strings.TrimSpace(ev.GetSuiteID())
			if eventTemplateID == "" {
				eventTemplateID = cfgTemplateID
			}
			_ = h.saveSuiteTicket(c.Request.Context(), eventTemplateID, strings.TrimSpace(ticketEvent.SuiteTicket))
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
	_, err = h.platformRepo.UpsertByChannelProvider(ctx, "wechat", "openwork", record.Enabled, cfg)
	return err
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
