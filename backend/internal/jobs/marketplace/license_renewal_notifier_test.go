package marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type stubExpiringRepo struct {
	records map[string][]*dbm.License
	err     error
}

func (s *stubExpiringRepo) ListExpiringWithin(ctx context.Context, tenantID string, window time.Duration) ([]*dbm.License, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.records[tenantID], nil
}

type stubDispatcher struct {
	calls int
	last  *dbm.License
}

func (s *stubDispatcher) DispatchRenewalReminder(_ context.Context, license *dbm.License, _ []string) error {
	s.calls++
	s.last = license
	return nil
}

func TestLicenseRenewalNotifier_ExecuteDispatchesReminders(t *testing.T) {
	cfg := &config.Config{
		Marketplace: &config.MarketplaceConfig{
			License: config.MarketplaceLicenseConfig{
				Reminder: config.MarketplaceLicenseReminderConfig{
					LeadHours: 24,
					Channels:  []string{"email", "in_app"},
				},
			},
		},
	}

	baseTime := time.Date(2025, 10, 30, 12, 0, 0, 0, time.UTC)
	expiring := &dbm.License{
		ID:        "lic-1",
		TenantUuid:  "tenant-1",
		ListingID: "listing-9",
		PlanID:    "plan-x",
		Status:    dbm.LicenseStatusActive,
		IssuedAt:  baseTime.Add(-48 * time.Hour),
		ExpiresAt: baseTime.Add(20 * time.Hour),
		RenewalToken: func() *string {
			val := "renew-1"
			return &val
		}(),
	}

	repo := &stubExpiringRepo{
		records: map[string][]*dbm.License{
			"tenant-1": {expiring},
		},
	}
	dispatcher := &stubDispatcher{}

	notifier := NewLicenseRenewalNotifier(cfg, repo, logrus.New().WithField("component", "test"), func(context.Context) ([]string, error) {
		return []string{"tenant-1"}, nil
	}, dispatcher)
	notifier.clock = func() time.Time { return baseTime }
	notifier.interval = time.Minute

	notifier.execute(context.Background())

	require.Equal(t, 1, dispatcher.calls)
	require.Equal(t, expiring.ID, dispatcher.last.ID)
}
