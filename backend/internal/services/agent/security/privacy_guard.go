package security

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	privmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/privacy"
	privrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/privacy"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// PrivacyGuard enforces consent scope and outbound data controls for agent flows.
type PrivacyGuard struct {
	repo     *privrepo.Repository
	cfg      *config.Config
	logger   *logrus.Entry
	allowset map[string]struct{}
	filter   *AIFilter
}

// NewPrivacyGuard constructs the agent privacy guard using the provided DB handle.
func NewPrivacyGuard(db *gorm.DB, cfg *config.Config, logger *logrus.Entry) *PrivacyGuard {
	allow := make(map[string]struct{})
	if cfg != nil && cfg.Security != nil {
		for _, host := range cfg.Security.GatewayAllowlist {
			if h := strings.TrimSpace(strings.ToLower(host)); h != "" {
				allow[h] = struct{}{}
			}
		}
	}
	return &PrivacyGuard{
		repo:     privrepo.NewRepository(db),
		cfg:      cfg,
		logger:   logger,
		allowset: allow,
		filter:   NewAIFilter(cfg),
	}
}

// EnsureConsent verifies that the tenant possesses active consent for all assets.
func (g *PrivacyGuard) EnsureConsent(ctx context.Context, tenantID string, requiredAssets []string) error {
	if len(requiredAssets) == 0 {
		return nil
	}
	tokens, err := g.repo.ActiveConsentTokens(ctx, tenantID, time.Now().UTC())
	if err != nil {
		return err
	}
	scope := make(map[string]struct{})
	for _, token := range tokens {
		values, parseErr := token.ScopeValues()
		if parseErr != nil {
			if g.logger != nil {
				g.logger.WithError(parseErr).
					WithField("token_id", token.ID).
					Warn("failed to parse consent scope")
			}
			continue
		}
		for _, asset := range values {
			scope[asset] = struct{}{}
		}
	}
	for _, asset := range requiredAssets {
		if _, ok := scope[asset]; !ok {
			return fmt.Errorf("asset %s not covered by active consent", asset)
		}
	}
	return nil
}

// ActiveConsentTokens fetches active consent tokens for the tenant.
func (g *PrivacyGuard) ActiveConsentTokens(ctx context.Context, tenantID string) ([]*privmodel.ConsentToken, error) {
	return g.repo.ActiveConsentTokens(ctx, tenantID, time.Now().UTC())
}

// ValidateOutboundURL ensures outbound communication stays within the gateway allowlist.
func (g *PrivacyGuard) ValidateOutboundURL(raw string) error {
	if raw == "" {
		return errors.New("outbound url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if g.cfg != nil && g.cfg.Security != nil && g.cfg.Security.RequireTLS13 {
		if parsed.Scheme != "https" {
			return errors.New("outbound request must use https")
		}
	}
	host := strings.ToLower(parsed.Hostname())
	if len(g.allowset) > 0 {
		if _, ok := g.allowset[host]; !ok {
			return fmt.Errorf("host %s not in gateway allowlist", host)
		}
	}
	return nil
}

// RecordLifecycleEvent proxies to the repository for lifecycle evidence.
func (g *PrivacyGuard) RecordLifecycleEvent(ctx context.Context, evt *privmodel.LifecycleEvent) (*privmodel.LifecycleEvent, error) {
	return g.repo.CreateLifecycleEvent(ctx, evt)
}

// FilterAIData masks sensitive keys within AI input/output payloads.
func (g *PrivacyGuard) FilterAIData(data map[string]interface{}) map[string]interface{} {
	if g.filter == nil {
		return data
	}
	return g.filter.FilterMap(data)
}
