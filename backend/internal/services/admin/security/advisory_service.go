package security

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	secmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
	secrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/security"
	secobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CreateAdvisoryParams captures the required inputs for opening a vulnerability advisory.
type CreateAdvisoryParams struct {
	Reference        string
	Severity         string
	Summary          string
	DetailsMarkdown  string
	AffectedVersions []string
	SlaDeadline      *time.Time
}

// PublishAdvisoryParams captures metadata needed for the publish transition.
type PublishAdvisoryParams struct {
	AdvisoryID       string
	PatchedInVersion string
	NotifyChannels   []string
	Metadata         datatypes.JSONMap
}

// AdvisoryService orchestrates advisory lifecycle transitions and notifications.
type AdvisoryService struct {
	db            *gorm.DB
	advisories    *secrepo.AdvisoryRepository
	distributions *secrepo.DistributionRepository
	logger        *logrus.Entry
	now           func() time.Time
}

// NewAdvisoryService constructs the advisory service.
func NewAdvisoryService(db *gorm.DB, logger *logrus.Entry) *AdvisoryService {
	svc := &AdvisoryService{
		db:            db,
		advisories:    secrepo.NewAdvisoryRepository(db),
		distributions: secrepo.NewDistributionRepository(db),
		logger:        logger,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
	return svc
}

// WithClock overrides the internal clock (useful for testing).
func (s *AdvisoryService) WithClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// CreateAdvisory stores a new advisory and computes SLA deadlines.
func (s *AdvisoryService) CreateAdvisory(ctx context.Context, params CreateAdvisoryParams) (*secmodel.Advisory, error) {
	if params.Reference == "" {
		return nil, errors.New("reference is required")
	}
	severity := normalizeSeverity(params.Severity)
	if severity == "" {
		return nil, fmt.Errorf("unsupported severity: %s", params.Severity)
	}
	if params.Summary == "" {
		return nil, errors.New("summary is required")
	}
	if len(params.AffectedVersions) == 0 {
		params.AffectedVersions = []string{}
	}
	now := s.now()
	deadline := params.SlaDeadline
	if deadline == nil {
		computed := computeSLADeadline(severity, now)
		deadline = &computed
	}
	advisory := &secmodel.Advisory{
		Reference:       params.Reference,
		Severity:        severity,
		Status:          secmodel.AdvisoryStatusOpen,
		Summary:         params.Summary,
		DetailsMarkdown: params.DetailsMarkdown,
		SlaDeadline:     deadline,
	}
	advisory.SetAffectedVersions(params.AffectedVersions)
	result, err := s.advisories.Create(ctx, advisory)
	if err != nil {
		return nil, err
	}
	meta := map[string]interface{}{
		"severity":     severity,
		"reference":    params.Reference,
		"sla_deadline": deadline.UTC().Format(time.RFC3339),
	}
	secobs.EmitAdvisoryDetected(s.logger, result, meta)
	secobs.RecordAdvisoryLifecycle(severity, secmodel.AdvisoryStatusOpen)
	return result, nil
}

// ListAdvisories fetches advisory records filtered by severity/status.
func (s *AdvisoryService) ListAdvisories(ctx context.Context, severities, statuses []string, limit int) ([]*secmodel.Advisory, error) {
	filter := secrepo.AdvisoryListFilter{
		Severities: severities,
		Statuses:   statuses,
		Limit:      limit,
	}
	advisories, err := s.advisories.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	secobs.UpdateOpenAdvisories(groupOpenBySeverity(advisories))
	return advisories, nil
}

// PublishAdvisory promotes the advisory to published and queues notifications.
func (s *AdvisoryService) PublishAdvisory(ctx context.Context, params PublishAdvisoryParams) (*secmodel.Advisory, []*secmodel.AdvisoryDistribution, error) {
	if params.AdvisoryID == "" {
		return nil, nil, errors.New("advisory_id is required")
	}
	if params.PatchedInVersion == "" {
		return nil, nil, errors.New("patched_in_version is required")
	}
	now := s.now()
	var (
		advisory      *secmodel.Advisory
		distributions []*secmodel.AdvisoryDistribution
	)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := s.advisories.WithTx(tx)
		distRepo := s.distributions.WithTx(tx)

		current, err := repo.GetByID(ctx, params.AdvisoryID)
		if err != nil {
			return err
		}
		if current.Status == secmodel.AdvisoryStatusClosed {
			return errors.New("cannot publish a closed advisory")
		}
		if current.Status == secmodel.AdvisoryStatusPublished {
			return errors.New("advisory already published")
		}

		updates := map[string]interface{}{
			"status":             secmodel.AdvisoryStatusPublished,
			"patched_in_version": params.PatchedInVersion,
			"published_at":       now,
		}
		if current.PatchedAt == nil {
			updates["patched_at"] = now
		}

		if err := repo.UpdateFields(ctx, current.ID, updates); err != nil {
			return err
		}

		updated, err := repo.GetByID(ctx, current.ID)
		if err != nil {
			return err
		}
		advisory = updated

		channels := sanitizeChannels(params.NotifyChannels)
		if len(channels) == 0 {
			channels = []string{secmodel.DistributionChannelMarketplace}
		}
		for _, channel := range channels {
			record := &secmodel.AdvisoryDistribution{
				AdvisoryID: advisory.ID,
				TenantUuid: "*",
				Channel:    channel,
				Status:     secmodel.DistributionStatusPending,
			}
			if params.Metadata != nil {
				record.Metadata = params.Metadata
			}
			row, err := distRepo.Upsert(ctx, record)
			if err != nil {
				return err
			}
			distributions = append(distributions, row)

			payload := map[string]interface{}{
				"status": secmodel.DistributionStatusPending,
			}
			if params.Metadata != nil {
				for k, v := range params.Metadata {
					payload[k] = v
				}
			}
			secobs.QueueAdvisoryNotification(s.logger, advisory, channel, payload)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	meta := map[string]interface{}{
		"patched_in_version": params.PatchedInVersion,
		"channels":           channelsToInterfaces(distributions),
	}
	secobs.EmitAdvisoryRemediated(s.logger, advisory, meta)
	secobs.RecordAdvisoryLifecycle(advisory.Severity, advisory.Status)
	s.refreshOpenGauge(ctx)
	return advisory, distributions, nil
}

// CloseAdvisory marks the advisory as closed once acknowledgements are complete.
func (s *AdvisoryService) CloseAdvisory(ctx context.Context, id string) (*secmodel.Advisory, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	now := s.now()
	if err := s.advisories.UpdateFields(ctx, id, map[string]interface{}{
		"status":    secmodel.AdvisoryStatusClosed,
		"closed_at": now,
	}); err != nil {
		return nil, err
	}
	updated, err := s.advisories.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	secobs.RecordAdvisoryLifecycle(updated.Severity, secmodel.AdvisoryStatusClosed)
	s.refreshOpenGauge(ctx)
	return updated, nil
}

func computeSLADeadline(severity string, from time.Time) time.Time {
	switch severity {
	case secmodel.AdvisorySeverityCritical:
		return from.Add(24 * time.Hour)
	case secmodel.AdvisorySeverityHigh:
		return from.Add(72 * time.Hour)
	case secmodel.AdvisorySeverityMedium:
		return from.AddDate(0, 0, 7)
	default:
		return from.AddDate(0, 0, 14)
	}
}

func normalizeSeverity(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case secmodel.AdvisorySeverityCritical:
		return secmodel.AdvisorySeverityCritical
	case secmodel.AdvisorySeverityHigh:
		return secmodel.AdvisorySeverityHigh
	case secmodel.AdvisorySeverityMedium:
		return secmodel.AdvisorySeverityMedium
	case secmodel.AdvisorySeverityLow:
		return secmodel.AdvisorySeverityLow
	default:
		return ""
	}
}

func sanitizeChannels(channels []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(channels))
	for _, ch := range channels {
		switch strings.ToUpper(strings.TrimSpace(ch)) {
		case secmodel.DistributionChannelEmail:
			if _, ok := seen[secmodel.DistributionChannelEmail]; !ok {
				seen[secmodel.DistributionChannelEmail] = struct{}{}
				out = append(out, secmodel.DistributionChannelEmail)
			}
		case secmodel.DistributionChannelWebhook:
			if _, ok := seen[secmodel.DistributionChannelWebhook]; !ok {
				seen[secmodel.DistributionChannelWebhook] = struct{}{}
				out = append(out, secmodel.DistributionChannelWebhook)
			}
		case secmodel.DistributionChannelMarketplace, "":
			if _, ok := seen[secmodel.DistributionChannelMarketplace]; !ok {
				seen[secmodel.DistributionChannelMarketplace] = struct{}{}
				out = append(out, secmodel.DistributionChannelMarketplace)
			}
		}
	}
	return out
}

func channelsToInterfaces(distributions []*secmodel.AdvisoryDistribution) []string {
	out := make([]string, 0, len(distributions))
	for _, dist := range distributions {
		out = append(out, dist.Channel)
	}
	return out
}

func (s *AdvisoryService) refreshOpenGauge(ctx context.Context) {
	openList, err := s.advisories.List(ctx, secrepo.AdvisoryListFilter{
		Statuses: []string{secmodel.AdvisoryStatusOpen},
	})
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Debug("failed to refresh advisory open gauge")
		}
		return
	}
	secobs.UpdateOpenAdvisories(groupOpenBySeverity(openList))
}

func groupOpenBySeverity(advisories []*secmodel.Advisory) map[string]int {
	counts := map[string]int{}
	for _, adv := range advisories {
		if adv == nil {
			continue
		}
		if adv.Status != secmodel.AdvisoryStatusOpen {
			continue
		}
		key := adv.Severity
		if key == "" {
			key = "unknown"
		}
		counts[key]++
	}
	return counts
}
