package recommendation

import (
	"context"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
)

// ListingMetricsProvider fetches signals using listing repository data.
type ListingMetricsProvider struct {
	repo *mrepo.ListingRepository
}

// NewListingMetricsProvider constructs a provider backed by listing repository.
func NewListingMetricsProvider(repo *mrepo.ListingRepository) *ListingMetricsProvider {
	return &ListingMetricsProvider{repo: repo}
}

// FetchSignals loads published listings and converts them into recommendation signals.
func (p *ListingMetricsProvider) FetchSignals(ctx context.Context, tenantID string) ([]Signal, error) {
	if p == nil || p.repo == nil {
		return nil, nil
	}
	listings, _, err := p.repo.List(ctx, tenantID, mrepo.ListingQuery{Status: []string{dbm.ListingStatusPublished}})
	if err != nil {
		return nil, err
	}
	signals := make([]Signal, 0, len(listings))
	for _, listing := range listings {
		if listing == nil {
			continue
		}
		signal := PrepareSignalFromListing(listing)
		signals = append(signals, signal)
	}
	return signals, nil
}
