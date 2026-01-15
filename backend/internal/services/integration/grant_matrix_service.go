package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// ErrGrantMatrixNotLoaded 表示无法读取配置。
var ErrGrantMatrixNotLoaded = errors.New("grant matrix entries not available")

// ErrGrantMatrixDenied 表示 ToolScope 未被允许访问目标资源。
var ErrGrantMatrixDenied = errors.New("tool scope not permitted for requested resource")

// GrantMatrixService 提供策略查询与校验能力。
type GrantMatrixService struct {
	loader *GrantMatrixLoader
	logger *logrus.Entry
}

// NewGrantMatrixService 构造服务。
func NewGrantMatrixService(loader *GrantMatrixLoader, logger *logrus.Entry) *GrantMatrixService {
	if logger == nil {
		logger = logrus.WithField("component", "integration.grant_matrix_service")
	}
	return &GrantMatrixService{
		loader: loader,
		logger: logger,
	}
}

// List 返回匹配过滤条件的策略条目。
func (s *GrantMatrixService) List(ctx context.Context, scope, channel string) ([]GrantMatrixEntry, error) {
	if s.loader == nil {
		return nil, ErrGrantMatrixNotLoaded
	}

	entries, err := s.loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	scope = strings.TrimSpace(scope)
	channel = strings.ToUpper(strings.TrimSpace(channel))

	if scope == "" && channel == "" {
		return entries, nil
	}

	filtered := make([]GrantMatrixEntry, 0, len(entries))
	for _, entry := range entries {
		if scope != "" && !strings.EqualFold(entry.Scope, scope) {
			continue
		}
		if channel != "" && !strings.EqualFold(entry.Channel, channel) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}

// EnsureAccess 校验 ToolScope 是否允许访问资源。
func (s *GrantMatrixService) EnsureAccess(ctx context.Context, toolScope, channel, resource, action string) (*GrantMatrixEntry, error) {
	if s.loader == nil {
		return nil, ErrGrantMatrixNotLoaded
	}

	entries, err := s.loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	toolScope = strings.TrimSpace(toolScope)
	channel = strings.ToUpper(strings.TrimSpace(channel))
	resource = normalizeResource(resource)
	action = strings.ToUpper(strings.TrimSpace(action))

	for _, entry := range entries {
		if !strings.EqualFold(entry.Scope, toolScope) {
			continue
		}
		if channel != "" && !strings.EqualFold(entry.Channel, channel) {
			continue
		}
		if !resourceMatches(entry.Resource, resource) {
			continue
		}
		if action != "" && !strings.EqualFold(entry.Action, action) {
			continue
		}
		return &entry, nil
	}

	return nil, fmt.Errorf("%w: scope=%s channel=%s resource=%s action=%s", ErrGrantMatrixDenied, toolScope, channel, resource, action)
}

// InvalidateCache 主动失效缓存（供审批或配置变更调用）。
func (s *GrantMatrixService) InvalidateCache() {
	if s.loader != nil {
		s.loader.Invalidate()
	}
}

// WarmCache 预热缓存。
func (s *GrantMatrixService) WarmCache(ctx context.Context) {
	if s.loader != nil {
		s.loader.Warm(ctx)
	}
}

func normalizeResource(resource string) string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return resource
	}
	if !strings.HasPrefix(resource, "/") {
		resource = "/" + resource
	}
	return resource
}

func resourceMatches(pattern, candidate string) bool {
	pattern = normalizeResource(pattern)
	candidate = normalizeResource(candidate)

	if pattern == candidate {
		return true
	}

	// 支持简单的前缀匹配，以容纳通配路径。
	if strings.HasSuffix(pattern, "*") {
		base := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(candidate, base)
	}

	return false
}
