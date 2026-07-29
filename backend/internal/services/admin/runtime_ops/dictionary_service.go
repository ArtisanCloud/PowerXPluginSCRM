package runtime_ops

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
)

const (
	NamespaceLeadTrafficPlatform = "scrm.lead.traffic_platform"
	NamespaceLeadTrafficSource   = "scrm.lead.traffic_source"
)

var (
	ErrDictionaryServiceUnavailable = errors.New("dictionary service unavailable")
	ErrInvalidDictionaryPayload     = errors.New("invalid dictionary payload")
)

type DictionaryService struct {
	deps      *app.Deps
	localRepo *leadrepo.LeadSourceCatalogRepository
}

type DictionaryItem struct {
	ItemID    string `json:"item_id"`
	Namespace string `json:"namespace"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	Sort      int    `json:"sort"`
	Enabled   bool   `json:"enabled"`
}

type DictionaryCreateRequest struct {
	Namespace string
	Code      string
	Label     string
	Sort      int
	Enabled   *bool
}

type DictionaryUpdateRequest struct {
	Code    string
	Label   string
	Sort    *int
	Enabled *bool
}

func NewDictionaryService(deps *app.Deps) *DictionaryService {
	s := &DictionaryService{deps: deps}
	if deps != nil && deps.DB != nil {
		s.localRepo = leadrepo.NewLeadSourceCatalogRepository(deps.DB)
	}
	return s
}

func (s *DictionaryService) List(ctx context.Context, tenantUUID, namespace string, enabledOnly bool) ([]DictionaryItem, error) {
	namespace = normalizeNamespace(namespace)
	if !isLocalNamespace(namespace) && s.deps != nil && s.deps.ProviderMode == iamservice.ProviderModeDelegated {
		return s.listDelegated(ctx, namespace, enabledOnly)
	}
	return s.listLocal(ctx, tenantUUID, namespace, enabledOnly)
}

func (s *DictionaryService) Create(ctx context.Context, tenantUUID string, req DictionaryCreateRequest) (*DictionaryItem, error) {
	ns := normalizeNamespace(req.Namespace)
	code := strings.TrimSpace(strings.ToLower(req.Code))
	label := strings.TrimSpace(req.Label)
	if ns == "" || code == "" || label == "" {
		return nil, ErrInvalidDictionaryPayload
	}
	if !isLocalNamespace(ns) && s.deps != nil && s.deps.ProviderMode == iamservice.ProviderModeDelegated {
		payload := map[string]any{
			"namespace": ns,
			"code":      code,
			"label":     label,
			"sort":      req.Sort,
		}
		if req.Enabled != nil {
			payload["enabled"] = *req.Enabled
		}
		var out DictionaryItem
		if err := s.proxyRequest(ctx, http.MethodPost, "/admin/runtime/dictionaries", payload, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	if s.localRepo == nil {
		return nil, ErrDictionaryServiceUnavailable
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	created, err := s.localRepo.Create(ctx, &leadmodel.LeadSourceCatalog{
		CatalogUUID: uuid.NewString(),
		Category:    namespaceToStorageCategory(ns),
		Code:        code,
		Label:       label,
		Sort:        req.Sort,
		Enabled:     enabled,
	})
	if err != nil {
		return nil, err
	}
	item := mapLocalItem(created)
	return &item, nil
}

func (s *DictionaryService) Update(ctx context.Context, tenantUUID, itemID, namespace string, req DictionaryUpdateRequest) (*DictionaryItem, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return nil, ErrInvalidDictionaryPayload
	}
	ns := normalizeNamespace(namespace)
	if !isLocalNamespace(ns) && s.deps != nil && s.deps.ProviderMode == iamservice.ProviderModeDelegated {
		payload := map[string]any{}
		if ns != "" {
			payload["namespace"] = ns
		}
		if strings.TrimSpace(req.Code) != "" {
			payload["code"] = strings.TrimSpace(req.Code)
		}
		if strings.TrimSpace(req.Label) != "" {
			payload["label"] = strings.TrimSpace(req.Label)
		}
		if req.Sort != nil {
			payload["sort"] = *req.Sort
		}
		if req.Enabled != nil {
			payload["enabled"] = *req.Enabled
		}
		if len(payload) == 0 {
			return nil, ErrInvalidDictionaryPayload
		}
		var out DictionaryItem
		if err := s.proxyRequest(ctx, http.MethodPatch, "/admin/runtime/dictionaries/"+url.PathEscape(itemID), payload, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	if s.localRepo == nil {
		return nil, ErrDictionaryServiceUnavailable
	}
	updates := map[string]any{}
	if ns != "" {
		updates["category"] = namespaceToStorageCategory(ns)
	}
	if strings.TrimSpace(req.Code) != "" {
		updates["code"] = strings.TrimSpace(strings.ToLower(req.Code))
	}
	if strings.TrimSpace(req.Label) != "" {
		updates["label"] = strings.TrimSpace(req.Label)
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(updates) == 0 {
		return nil, ErrInvalidDictionaryPayload
	}
	updated, err := s.localRepo.UpdateByUUID(ctx, itemID, updates)
	if err != nil {
		return nil, err
	}
	item := mapLocalItem(updated)
	return &item, nil
}

func (s *DictionaryService) Delete(ctx context.Context, tenantUUID, itemID, namespace string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrInvalidDictionaryPayload
	}
	ns := normalizeNamespace(namespace)
	if !isLocalNamespace(ns) && s.deps != nil && s.deps.ProviderMode == iamservice.ProviderModeDelegated {
		path := "/admin/runtime/dictionaries/" + url.PathEscape(itemID)
		if ns != "" {
			path += "?namespace=" + url.QueryEscape(ns)
		}
		return s.proxyRequest(ctx, http.MethodDelete, path, nil, nil)
	}
	if s.localRepo == nil {
		return ErrDictionaryServiceUnavailable
	}
	return s.localRepo.DeleteByUUID(ctx, itemID)
}

func (s *DictionaryService) listLocal(ctx context.Context, tenantUUID, namespace string, enabledOnly bool) ([]DictionaryItem, error) {
	if s.localRepo == nil {
		return nil, ErrDictionaryServiceUnavailable
	}
	category := ""
	if namespace != "" {
		category = namespaceToStorageCategory(namespace)
	}
	items, err := s.localRepo.List(ctx, category, enabledOnly)
	if err != nil {
		return nil, err
	}
	out := make([]DictionaryItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapLocalItem(item))
	}
	return out, nil
}

func (s *DictionaryService) listDelegated(ctx context.Context, namespace string, enabledOnly bool) ([]DictionaryItem, error) {
	query := url.Values{}
	if namespace != "" {
		query.Set("namespace", namespace)
	}
	if enabledOnly {
		query.Set("enabled", "true")
	}
	path := "/admin/runtime/dictionaries"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp struct {
		Items []DictionaryItem `json:"items"`
	}
	if err := s.proxyRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (s *DictionaryService) proxyRequest(ctx context.Context, method, path string, payload any, out any) error {
	if s == nil || s.deps == nil || s.deps.AuthProxy == nil {
		return ErrDictionaryServiceUnavailable
	}
	if err := s.deps.AuthProxy.ProxyRequest(ctx, method, path, payload, out, nil); err != nil {
		return fmt.Errorf("delegated dictionary request failed: %w", err)
	}
	return nil
}

func mapLocalItem(item *leadmodel.LeadSourceCatalog) DictionaryItem {
	namespace := storageCategoryToNamespace(item.Category)
	return DictionaryItem{
		ItemID:    item.CatalogUUID,
		Namespace: namespace,
		Code:      item.Code,
		Label:     item.Label,
		Sort:      item.Sort,
		Enabled:   item.Enabled,
	}
}

func normalizeNamespace(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func isLocalNamespace(namespace string) bool {
	switch normalizeNamespace(namespace) {
	case "", NamespaceLeadTrafficPlatform, NamespaceLeadTrafficSource:
		return true
	default:
		return false
	}
}

func namespaceToStorageCategory(namespace string) string {
	ns := normalizeNamespace(namespace)
	switch ns {
	case NamespaceLeadTrafficPlatform:
		return leadmodel.SourceCatalogCategoryTrafficPlatform
	case NamespaceLeadTrafficSource:
		return leadmodel.SourceCatalogCategoryTrafficSource
	default:
		return ns
	}
}

func storageCategoryToNamespace(category string) string {
	switch strings.TrimSpace(strings.ToLower(category)) {
	case leadmodel.SourceCatalogCategoryTrafficPlatform:
		return NamespaceLeadTrafficPlatform
	case leadmodel.SourceCatalogCategoryTrafficSource:
		return NamespaceLeadTrafficSource
	default:
		return strings.TrimSpace(strings.ToLower(category))
	}
}
