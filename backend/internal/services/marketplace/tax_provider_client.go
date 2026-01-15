package marketplace

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	marketobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/marketplace"
	"github.com/sirupsen/logrus"
)

var (
	// ErrProviderNotConfigured indicates the client was created without a backing provider adapter.
	ErrProviderNotConfigured = errors.New("tax provider not configured")
	// ErrNotImplemented indicates the concrete adapter has not yet implemented an operation.
	ErrNotImplemented = errors.New("tax provider operation not implemented")
	// ErrReplayNotSupported indicates the provider does not support replay semantics.
	ErrReplayNotSupported = errors.New("tax provider replay not supported")
)

// TaxChargeRequest captures the minimal context required for tax calculations.
type TaxChargeRequest struct {
	TenantUuid         string
	ListingID          string
	Currency           string
	AmountCents        int64
	AmountMinorUnits   int64
	SettlementCurrency string
	ExchangeRate       float64
	Jurisdiction       string
	ExternalReference  string
	Items              []TaxLineItem
	Metadata           map[string]string
}

// TaxLineItem describes per-item tax metadata (SKU/plan level granularity).
type TaxLineItem struct {
	SKU              string
	Quantity         int
	AmountCents      int64
	AmountMinorUnits int64
	TaxCode          string
}

// TaxChargeResult captures the provider response metadata.
type TaxChargeResult struct {
	ExternalTransactionID string
	TaxAmountCents        int64
	TaxAmountMinorUnits   int64
	Currency              string
	SettlementCurrency    string
	ExchangeRate          float64
	Jurisdiction          string
	RawPayload            []byte
}

// providerAdapter abstracts vendor specific integrations.
type providerAdapter interface {
	Name() string
	Dispatch(ctx context.Context, req *TaxChargeRequest) (*TaxChargeResult, error)
	Replay(ctx context.Context, externalTransactionID string) error
}

// TaxProviderClient orchestrates tax provider dispatch and replay with retry semantics.
type TaxProviderClient struct {
	provider string
	retries  []time.Duration
	adapter  providerAdapter
	logger   *logrus.Entry
}

// NewTaxProviderClient constructs a retry-aware client for the configured tax provider.
func NewTaxProviderClient(cfg *config.Config, httpClient *http.Client, logger *logrus.Entry) (*TaxProviderClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config required to initialize tax provider client")
	}

	provider := cfg.IntegrationTaxProvider()
	if provider == "" {
		return nil, fmt.Errorf("integration.billing.tax_provider is not configured")
	}

	schedule := cfg.BillingRetrySchedule()
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.BillingHTTPTimeout()}
	} else if httpClient.Timeout == 0 {
		httpClient.Timeout = cfg.BillingHTTPTimeout()
	}

	client := &TaxProviderClient{
		provider: provider,
		retries:  schedule,
		logger:   logger,
	}

	switch provider {
	case "stripe_tax":
		client.adapter = newStripeAdapter(httpClient, cfg.StripeTaxConfig(), logger)
	case "avalara":
		client.adapter = newAvalaraAdapter(httpClient, cfg.AvalaraConfig(), logger)
	default:
		return nil, fmt.Errorf("unsupported tax provider %q", provider)
	}

	return client, nil
}

// Provider returns the currently configured provider identifier.
func (c *TaxProviderClient) Provider() string {
	if c == nil {
		return ""
	}
	return c.provider
}

// CreateTransaction performs a best-effort dispatch with retries according to configuration.
func (c *TaxProviderClient) CreateTransaction(ctx context.Context, req *TaxChargeRequest) (*TaxChargeResult, error) {
	if c == nil || c.adapter == nil {
		return nil, ErrProviderNotConfigured
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := normalizeChargeRequest(req); err != nil {
		return nil, err
	}

	attempts := len(c.retries) + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if err := waitWithContext(ctx, c.retries[i-1]); err != nil {
				return nil, err
			}
		}

		result, err := c.adapter.Dispatch(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		c.recordFailure("dispatch", err)

		if !c.shouldRetry(err) || i == attempts-1 || ctx.Err() != nil {
			break
		}

		if c.logger != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"provider": c.provider,
				"attempt":  i + 1,
			}).Warn("tax provider dispatch failed, retrying")
		}
	}

	if lastErr == nil {
		lastErr = ErrNotImplemented
	}
	return nil, lastErr
}

// ReplayTransaction attempts to replay a previously failed tax transaction.
func (c *TaxProviderClient) ReplayTransaction(ctx context.Context, externalTransactionID string) error {
	if c == nil || c.adapter == nil {
		return ErrProviderNotConfigured
	}
	if ctx == nil {
		ctx = context.Background()
	}

	attempts := len(c.retries) + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if err := waitWithContext(ctx, c.retries[i-1]); err != nil {
				return err
			}
		}

		err := c.adapter.Replay(ctx, externalTransactionID)
		if err == nil {
			return nil
		}
		lastErr = err
		c.recordFailure("replay", err)

		if !c.shouldRetry(err) || i == attempts-1 || ctx.Err() != nil {
			break
		}

		if c.logger != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"provider": c.provider,
				"attempt":  i + 1,
			}).Warn("tax provider replay failed, retrying")
		}
	}

	if lastErr == nil {
		lastErr = ErrNotImplemented
	}
	return lastErr
}

func (c *TaxProviderClient) recordFailure(stage string, err error) {
	if err == nil {
		return
	}
	code := errorCode(err)
	marketobs.IncrementTaxProviderError(c.provider, fmt.Sprintf("%s_%s", stage, code))
}

func (c *TaxProviderClient) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrNotImplemented),
		errors.Is(err, ErrReplayNotSupported),
		errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return false
	default:
		return true
	}
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func errorCode(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrNotImplemented):
		return "not_implemented"
	case errors.Is(err, ErrReplayNotSupported):
		return "replay_unsup"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	default:
		return "unknown"
	}
}

func normalizeChargeRequest(req *TaxChargeRequest) error {
	if req == nil {
		return errors.New("nil tax charge request")
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency == "" {
		return errors.New("currency must be provided")
	}
	if req.AmountMinorUnits == 0 && req.AmountCents != 0 {
		req.AmountMinorUnits = req.AmountCents
	}
	if req.AmountCents == 0 && req.AmountMinorUnits != 0 && currencyExponent(req.Currency) == 2 {
		req.AmountCents = req.AmountMinorUnits
	}
	if req.Metadata == nil {
		req.Metadata = map[string]string{}
	}
	if req.SettlementCurrency == "" {
		req.SettlementCurrency = req.Currency
	} else {
		req.SettlementCurrency = strings.ToUpper(strings.TrimSpace(req.SettlementCurrency))
	}
	if req.ExchangeRate <= 0 {
		if req.SettlementCurrency == req.Currency {
			req.ExchangeRate = 1
		}
	}
	for i := range req.Items {
		req.Items[i].AmountMinorUnits = normalizeItemAmount(req.Currency, req.Items[i].AmountMinorUnits, req.Items[i].AmountCents)
		if req.Items[i].AmountCents == 0 && req.Items[i].AmountMinorUnits != 0 && currencyExponent(req.Currency) == 2 {
			req.Items[i].AmountCents = req.Items[i].AmountMinorUnits
		}
	}
	return nil
}

func normalizeItemAmount(currency string, minorUnits, cents int64) int64 {
	if minorUnits != 0 {
		return minorUnits
	}
	if cents != 0 {
		return cents
	}
	return 0
}

var currencyMinorUnitExponent = map[string]int{
	"BHD": 3,
	"IQD": 3,
	"JOD": 3,
	"KWD": 3,
	"LYD": 3,
	"OMR": 3,
	"TND": 3,
	"CLF": 4,
	"MRO": 1,
	"MRU": 1,
	"DJF": 0,
	"GNF": 0,
	"JPY": 0,
	"KMF": 0,
	"KRW": 0,
	"PYG": 0,
	"RWF": 0,
	"UGX": 0,
	"VND": 0,
	"VUV": 0,
	"XAF": 0,
	"XOF": 0,
	"XPF": 0,
}

func currencyExponent(currency string) int {
	if exp, ok := currencyMinorUnitExponent[strings.ToUpper(strings.TrimSpace(currency))]; ok {
		return exp
	}
	return 2
}

// AmountToMinorUnits converts a decimal amount into integer minor units for the currency.
func AmountToMinorUnits(currency string, amount float64) (int64, error) {
	if currency = strings.ToUpper(strings.TrimSpace(currency)); currency == "" {
		return 0, errors.New("currency required")
	}
	exp := currencyExponent(currency)
	multiplier := math.Pow10(exp)
	return int64(math.Round(amount * multiplier)), nil
}

// MinorUnitsToAmount converts minor units back into a decimal representation.
func MinorUnitsToAmount(currency string, units int64) float64 {
	exp := currencyExponent(currency)
	multiplier := math.Pow10(exp)
	if multiplier == 0 {
		return 0
	}
	return float64(units) / multiplier
}

type stripeAdapter struct {
	httpClient *http.Client
	cfg        config.IntegrationStripeTaxConfig
	logger     *logrus.Entry
}

func newStripeAdapter(httpClient *http.Client, cfg config.IntegrationStripeTaxConfig, logger *logrus.Entry) providerAdapter {
	return &stripeAdapter{
		httpClient: httpClient,
		cfg:        cfg,
		logger:     logger,
	}
}

func (a *stripeAdapter) Name() string {
	return "stripe_tax"
}

func (a *stripeAdapter) Dispatch(ctx context.Context, req *TaxChargeRequest) (*TaxChargeResult, error) {
	return nil, ErrNotImplemented
}

func (a *stripeAdapter) Replay(ctx context.Context, externalTransactionID string) error {
	return ErrNotImplemented
}

type avalaraAdapter struct {
	httpClient *http.Client
	cfg        config.IntegrationAvalaraConfig
	logger     *logrus.Entry
}

func newAvalaraAdapter(httpClient *http.Client, cfg config.IntegrationAvalaraConfig, logger *logrus.Entry) providerAdapter {
	return &avalaraAdapter{
		httpClient: httpClient,
		cfg:        cfg,
		logger:     logger,
	}
}

func (a *avalaraAdapter) Name() string {
	return "avalara"
}

func (a *avalaraAdapter) Dispatch(ctx context.Context, req *TaxChargeRequest) (*TaxChargeResult, error) {
	return nil, ErrNotImplemented
}

func (a *avalaraAdapter) Replay(ctx context.Context, externalTransactionID string) error {
	return ErrNotImplemented
}
