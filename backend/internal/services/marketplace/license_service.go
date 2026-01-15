package marketplace

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	marketobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/marketplace"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// BillingClient represents a billing engine integration.
type BillingClient interface {
	ChargeSubscription(ctx context.Context, tenantID string, plan *dbm.PricingPlan, metadata map[string]any) (string, error)
}

// LicenseAuthority communicates with the centralized license server.
type LicenseAuthority interface {
	Issue(ctx context.Context, req *LicenseIssueRequest) (*LicenseIssueResponse, error)
	Renew(ctx context.Context, req *LicenseRenewRequest) (*LicenseIssueResponse, error)
	Revoke(ctx context.Context, licenseID string, reason string) error
	Verify(ctx context.Context, token string) (bool, error)
}

// LicenseCache provides temporary license caching for verification.
type LicenseCache interface {
	Get(ctx context.Context, tenantID, listingID string) (*dbm.License, bool)
	Set(ctx context.Context, tenantID, listingID string, license *dbm.License, ttl time.Duration)
	Delete(ctx context.Context, tenantID, listingID string)
}

// LicenseIssueRequest captures issuance payload for authority calls.
type LicenseIssueRequest struct {
	TenantUuid string
	ListingID  string
	PlanID     string
	Metadata   map[string]any
}

// LicenseRenewRequest captures renew payload for authority calls.
type LicenseRenewRequest struct {
	LicenseID  string
	TenantUuid string
	PlanID     string
	Metadata   map[string]any
}

// LicenseIssueResponse represents license authority response.
type LicenseIssueResponse struct {
	Token     string
	ExpiresAt time.Time
	Metadata  map[string]any
}

// IssueLicenseParams encapsulates inputs for issuing license.
type IssueLicenseParams struct {
	TenantUuid  string
	ListingID   string
	PlanID      string
	IssuedBy    string
	Trial       bool
	Metadata    map[string]any
	ExpiresAt   time.Time
	SkipBilling bool
}

// RenewLicenseParams encapsulates inputs for renewal.
type RenewLicenseParams struct {
	LicenseID    string
	TenantUuid   string
	IssuedBy     string
	PlanID       string
	RenewalToken string
	Metadata     map[string]any
	ExpiresAt    time.Time
}

// VerifyResult describes verification result.
type VerifyResult struct {
	License *dbm.License
	Valid   bool
	Reason  string
}

// LicenseService orchestrates plan lookup, billing, license issuance and caching.
type LicenseService struct {
	cfg           *config.Config
	pricingRepo   *mrepo.PricingRepository
	licenseRepo   *mrepo.LicenseRepository
	taxClient     *TaxProviderClient
	billingClient BillingClient
	authority     LicenseAuthority
	cache         LicenseCache
	logger        *logrus.Entry
}

// NewLicenseService constructs the service with dependencies.
func NewLicenseService(cfg *config.Config, pricingRepo *mrepo.PricingRepository, licenseRepo *mrepo.LicenseRepository, taxClient *TaxProviderClient, billing BillingClient, authority LicenseAuthority, cache LicenseCache, logger *logrus.Entry) *LicenseService {
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace_license_service")
	}
	return &LicenseService{
		cfg:           cfg,
		pricingRepo:   pricingRepo,
		licenseRepo:   licenseRepo,
		taxClient:     taxClient,
		billingClient: billing,
		authority:     authority,
		cache:         cache,
		logger:        logger,
	}
}

// IssueLicense issues a new license for the given plan.
func (s *LicenseService) IssueLicense(ctx context.Context, params IssueLicenseParams) (*dbm.License, error) {
	start := time.Now()
	resultLabel := "success"
	defer func() {
		provider := ""
		if s.taxClient != nil {
			provider = s.taxClient.Provider()
		}
		marketobs.ObserveLicenseVerification(resultLabel, provider, params.TenantUuid, time.Since(start))
	}()

	if hasEmpty(params.TenantUuid, params.ListingID, params.PlanID) {
		resultLabel = "invalid_request"
		return nil, errors.New("tenant_uuid, listing_id, plan_id are required")
	}

	plan, err := s.pricingRepo.GetPlan(ctx, params.TenantUuid, params.PlanID)
	if err != nil {
		resultLabel = "plan_not_found"
		return nil, err
	}
	if !strings.EqualFold(plan.ListingID, params.ListingID) {
		resultLabel = "plan_mismatch"
		return nil, fmt.Errorf("plan %s does not belong to listing %s", plan.ID, params.ListingID)
	}
	if status := strings.TrimSpace(strings.ToLower(plan.Status)); status != "" && status != "active" {
		resultLabel = "plan_inactive"
		return nil, fmt.Errorf("plan %s is not active", plan.ID)
	}

	settlementConfig := s.cfg.IntegrationRevenueSplit()
	settlementCurrency := strings.ToUpper(strings.TrimSpace(settlementConfig.Currency))
	if settlementCurrency == "" {
		settlementCurrency = strings.ToUpper(strings.TrimSpace(plan.Currency))
	}
	if settlementCurrency == "" {
		settlementCurrency = "USD"
	}
	exchangeRate := parseExchangeRate(params.Metadata)
	if strings.EqualFold(settlementCurrency, plan.Currency) && exchangeRate <= 0 {
		exchangeRate = 1
	}

	existingBillingID := strings.TrimSpace(billingIDFromMeta(params.Metadata))
	billingID := existingBillingID
	shouldCharge := !params.SkipBilling && plan.Amount != nil && *plan.Amount > 0 && !params.Trial
	if shouldCharge {
		if s.billingClient == nil {
			resultLabel = "billing_unavailable"
			return nil, errors.New("billing client not configured")
		}
		id, billErr := s.billingClient.ChargeSubscription(ctx, params.TenantUuid, plan, params.Metadata)
		if billErr != nil {
			resultLabel = "billing_failed"
			return nil, fmt.Errorf("billing charge failed: %w", billErr)
		}
		billingID = id
	}

	var taxResult *TaxChargeResult
	var taxErr error
	if shouldCharge && s.taxClient != nil {
		amountUnits, err := AmountToMinorUnits(plan.Currency, *plan.Amount)
		if err != nil {
			resultLabel = "invalid_currency"
			return nil, fmt.Errorf("calculate tax amount: %w", err)
		}
		itemUnits := amountUnits
		req := &TaxChargeRequest{
			TenantUuid:         params.TenantUuid,
			ListingID:          params.ListingID,
			Currency:           plan.Currency,
			AmountMinorUnits:   amountUnits,
			Jurisdiction:       "",
			ExternalReference:  billingID,
			SettlementCurrency: settlementCurrency,
			ExchangeRate:       exchangeRate,
			Items: []TaxLineItem{
				{SKU: plan.PlanCode, Quantity: 1, AmountMinorUnits: itemUnits, TaxCode: plan.PlanType},
			},
			Metadata: map[string]string{
				"plan_currency": plan.Currency,
			},
		}
		if res, err := s.taxClient.CreateTransaction(ctx, req); err == nil {
			taxResult = res
		} else {
			taxErr = err
			marketobs.IncrementTaxProviderError(s.taxClient.Provider(), "dispatch_error")
			s.logger.WithError(err).Warn("tax transaction failed")
		}
	}

	expiry := params.ExpiresAt
	if expiry.IsZero() {
		expiry = time.Now().Add(30 * 24 * time.Hour)
	}

	issueRes := &LicenseIssueResponse{}
	if s.authority != nil {
		payload := &LicenseIssueRequest{
			TenantUuid: params.TenantUuid,
			ListingID:  params.ListingID,
			PlanID:     params.PlanID,
			Metadata:   params.Metadata,
		}
		resp, err := s.authority.Issue(ctx, payload)
		if err != nil {
			return nil, fmt.Errorf("license authority issue failed: %w", err)
		}
		issueRes = resp
		if !resp.ExpiresAt.IsZero() {
			expiry = resp.ExpiresAt
		}
	} else {
		token := generateToken(params.TenantUuid, params.ListingID, params.PlanID)
		issueRes.Token = token
		issueRes.Metadata = params.Metadata
	}

	issuedAt := time.Now().UTC()
	allowance := s.offlineAllowance()
	offlineUntil := timePtr(minTime(expiry, issuedAt.Add(allowance)))
	lastValidated := issuedAt
	var metadata map[string]any
	if len(params.Metadata) > 0 {
		metadata = make(map[string]any, len(params.Metadata))
		for k, v := range params.Metadata {
			metadata[k] = v
		}
	}
	if issueRes.Metadata != nil {
		if metadata == nil {
			metadata = make(map[string]any, len(issueRes.Metadata))
		}
		for k, v := range issueRes.Metadata {
			metadata[k] = v
		}
	}

	renewalToken := uuid.NewString()
	if metadata == nil {
		metadata = map[string]any{}
	}
	if taxResult != nil {
		metadata["tax_transaction_id"] = taxResult.ExternalTransactionID
	}
	if billingID != "" {
		metadata["billing_id"] = billingID
	}
	metadata["settlement_currency"] = settlementCurrency
	if exchangeRate > 0 {
		metadata["exchange_rate"] = exchangeRate
	}

	license := &dbm.License{
		TenantUuid:      params.TenantUuid,
		ListingID:       params.ListingID,
		PlanID:          params.PlanID,
		LicenseToken:    issueRes.Token,
		Status:          statusFromTrial(params.Trial),
		IssuedAt:        issuedAt,
		ExpiresAt:       expiry,
		OfflineUntil:    offlineUntil,
		IssuedBy:        stringPtr(params.IssuedBy),
		RenewalToken:    stringPtr(renewalToken),
		LastValidatedAt: &lastValidated,
		Metadata:        toJSONMap(metadata),
	}

	eventPayload := map[string]any{
		"plan_id": params.PlanID,
	}
	finalBillingID := billingID
	if finalBillingID == "" {
		finalBillingID = billingIDFromMeta(metadata)
	}
	if finalBillingID != "" {
		eventPayload["billing_id"] = finalBillingID
	}
	if taxResult != nil && taxResult.ExternalTransactionID != "" {
		eventPayload["tax_transaction_id"] = taxResult.ExternalTransactionID
	}
	event := &dbm.LicenseEvent{
		TenantUuid:   params.TenantUuid,
		EventType:    dbm.LicenseEventIssued,
		EventPayload: toJSONMap(eventPayload),
	}

	if err := s.licenseRepo.CreateLicense(ctx, license, event); err != nil {
		return nil, err
	}

	if taxResult != nil {
		chargeCurrency := strings.ToUpper(strings.TrimSpace(taxResult.Currency))
		if chargeCurrency == "" {
			chargeCurrency = strings.ToUpper(strings.TrimSpace(plan.Currency))
		}
		units := taxResult.TaxAmountMinorUnits
		if units == 0 && taxResult.TaxAmountCents != 0 {
			units = taxResult.TaxAmountCents
		}
		taxAmount := MinorUnitsToAmount(chargeCurrency, units)
		settlement := strings.ToUpper(strings.TrimSpace(taxResult.SettlementCurrency))
		if settlement == "" {
			settlement = settlementCurrency
		}
		exRate := taxResult.ExchangeRate
		if exRate <= 0 {
			if strings.EqualFold(settlement, chargeCurrency) {
				exRate = 1
			} else if exchangeRate > 0 {
				exRate = exchangeRate
			}
		}
		var settlementAmount *float64
		if exRate > 0 {
			value := taxAmount * exRate
			settlementAmount = &value
		}
		txn := &dbm.TaxTransaction{
			TenantUuid:            params.TenantUuid,
			BillingID:             billingID,
			ExternalProvider:      s.taxClient.Provider(),
			ExternalTransactionID: stringPtr(taxResult.ExternalTransactionID),
			Jurisdiction:          taxResult.Jurisdiction,
			TaxAmount:             taxAmount,
			Currency:              chargeCurrency,
			SettlementCurrency:    settlement,
			ExchangeRate: func() *float64 {
				if exRate > 0 {
					return &exRate
				}
				return nil
			}(),
			TaxAmountSettlement: settlementAmount,
			RawPayload:          jsonFromBytes(taxResult.RawPayload),
			Status:              "completed",
		}
		_ = s.licenseRepo.RecordTaxTransaction(ctx, txn)
	} else if shouldCharge && s.taxClient != nil && billingID != "" {
		failPayload := map[string]any{}
		if taxErr != nil {
			failPayload["error"] = taxErr.Error()
		}
		txn := &dbm.TaxTransaction{
			TenantUuid:         params.TenantUuid,
			BillingID:          billingID,
			ExternalProvider:   s.taxClient.Provider(),
			Currency:           strings.ToUpper(strings.TrimSpace(plan.Currency)),
			SettlementCurrency: settlementCurrency,
			ExchangeRate: func() *float64 {
				if exchangeRate > 0 {
					val := exchangeRate
					return &val
				}
				return nil
			}(),
			TaxAmount:  0,
			Status:     "failed",
			RawPayload: toJSONMap(failPayload),
		}
		_ = s.licenseRepo.RecordTaxTransaction(ctx, txn)
	}

	if s.cache != nil {
		ttl := licenseCacheTTL(license, allowance)
		s.cache.Set(ctx, params.TenantUuid, params.ListingID, license, ttl)
	}
	s.emitRenewalScheduled(license)
	return license, nil
}

// RenewLicense renews an existing license.
func (s *LicenseService) RenewLicense(ctx context.Context, params RenewLicenseParams) (*dbm.License, error) {
	if hasEmpty(params.TenantUuid, params.LicenseID) {
		return nil, errors.New("tenant_uuid and license_id are required")
	}

	license, err := s.licenseRepo.GetLicense(ctx, params.TenantUuid, params.LicenseID)
	if err != nil {
		return nil, err
	}

	planChanged := false
	if strings.TrimSpace(params.PlanID) != "" && !strings.EqualFold(params.PlanID, license.PlanID) {
		plan, err := s.pricingRepo.GetPlan(ctx, params.TenantUuid, params.PlanID)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(plan.ListingID, license.ListingID) {
			return nil, fmt.Errorf("plan %s does not belong to license listing %s", plan.ID, license.ListingID)
		}
		license.PlanID = plan.ID
		planChanged = true
	}

	if token := strings.TrimSpace(params.RenewalToken); token != "" {
		if license.RenewalToken != nil && token != strings.TrimSpace(*license.RenewalToken) {
			return nil, errors.New("invalid renewal token")
		}
	}

	expiry := params.ExpiresAt
	if expiry.IsZero() {
		expiry = time.Now().Add(30 * 24 * time.Hour)
	}

	if s.authority != nil {
		resp, err := s.authority.Renew(ctx, &LicenseRenewRequest{
			LicenseID:  license.ID,
			TenantUuid: params.TenantUuid,
			PlanID:     license.PlanID,
			Metadata:   params.Metadata,
		})
		if err != nil {
			return nil, err
		}
		if resp.Token != "" {
			license.LicenseToken = resp.Token
		}
		if !resp.ExpiresAt.IsZero() {
			expiry = resp.ExpiresAt
		}
	}

	newToken := uuid.NewString()
	validatedAt := time.Now()
	allowance := s.offlineAllowance()
	offlinePtr := timePtr(minTime(expiry, validatedAt.Add(allowance)))
	fields := map[string]any{
		"expires_at":        expiry,
		"status":            dbm.LicenseStatusActive,
		"last_validated_at": validatedAt,
		"renewal_token":     newToken,
		"offline_until":     offlinePtr,
	}
	if planChanged {
		fields["plan_id"] = license.PlanID
	}
	if err := s.licenseRepo.UpdateLicenseToken(ctx, params.TenantUuid, license.ID, fields); err != nil {
		return nil, err
	}

	event := &dbm.LicenseEvent{
		TenantUuid: params.TenantUuid,
		LicenseID:  license.ID,
		EventType:  dbm.LicenseEventRenewed,
		EventPayload: toJSONMap(map[string]any{
			"expires_at": expiry,
			"plan_id":    license.PlanID,
		}),
	}
	_ = s.licenseRepo.CreateEvent(ctx, event)

	if planChanged {
		license.PlanID = fields["plan_id"].(string)
	}
	license.ExpiresAt = expiry
	license.Status = dbm.LicenseStatusActive
	license.LastValidatedAt = &validatedAt
	license.RenewalToken = &newToken
	license.OfflineUntil = offlinePtr
	if s.cache != nil {
		ttl := licenseCacheTTL(license, allowance)
		s.cache.Set(ctx, params.TenantUuid, license.ListingID, license, ttl)
	}

	s.emitRenewalScheduled(license)
	return license, nil
}

// GetLicense returns a license by identifier.
func (s *LicenseService) GetLicense(ctx context.Context, tenantID, licenseID string) (*dbm.License, error) {
	if hasEmpty(tenantID, licenseID) {
		return nil, errors.New("tenant_uuid and license_id are required")
	}
	return s.licenseRepo.GetLicense(ctx, tenantID, licenseID)
}

// VerifyLicense checks the cache and repository to ensure license validity.
func (s *LicenseService) VerifyLicense(ctx context.Context, tenantID, listingID string) (*VerifyResult, error) {
	if hasEmpty(tenantID, listingID) {
		return nil, errors.New("tenant_uuid and listing_id are required")
	}
	allowance := s.offlineAllowance()
	if s.cache != nil {
		if cached, ok := s.cache.Get(ctx, tenantID, listingID); ok && cached != nil {
			return &VerifyResult{License: cached, Valid: !cached.ExpiresAt.Before(time.Now())}, nil
		}
	}
	license, err := s.licenseRepo.FindActiveLicense(ctx, tenantID, listingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &VerifyResult{Valid: false, Reason: "license not found"}, nil
		}
		return nil, err
	}
	valid := !license.ExpiresAt.Before(time.Now())
	if s.cache != nil {
		ttl := licenseCacheTTL(license, allowance)
		s.cache.Set(ctx, tenantID, listingID, license, ttl)
	}
	return &VerifyResult{License: license, Valid: valid}, nil
}

// RevokeLicense transitions a license to revoked state.
func (s *LicenseService) RevokeLicense(ctx context.Context, tenantID, licenseID, reason string) error {
	if hasEmpty(tenantID, licenseID) {
		return errors.New("tenant_uuid and license_id are required")
	}
	if s.authority != nil {
		if err := s.authority.Revoke(ctx, licenseID, reason); err != nil {
			return err
		}
	}
	if err := s.licenseRepo.UpdateLicenseToken(ctx, tenantID, licenseID, map[string]any{
		"status":            dbm.LicenseStatusRevoked,
		"last_validated_at": time.Now(),
	}); err != nil {
		return err
	}
	event := &dbm.LicenseEvent{
		TenantUuid: tenantID,
		LicenseID:  licenseID,
		EventType:  dbm.LicenseEventRevoked,
		EventPayload: toJSONMap(map[string]any{
			"reason": reason,
		}),
	}
	_ = s.licenseRepo.CreateEvent(ctx, event)
	if s.cache != nil {
		s.cache.Delete(ctx, tenantID, licenseID)
	}
	return nil
}

// ExtendOffline updates offline window within 72 hour constraint.
func (s *LicenseService) ExtendOffline(ctx context.Context, tenantID, licenseID string, until time.Time) error {
	if hasEmpty(tenantID, licenseID) {
		return errors.New("tenant_uuid and license_id are required")
	}
	target := until
	allowance := s.offlineAllowance()
	if allowance <= 0 {
		allowance = 72 * time.Hour
	}
	max := time.Now().Add(allowance)
	if until.After(max) {
		target = max
	}
	if err := s.licenseRepo.UpdateOfflineWindow(ctx, tenantID, licenseID, &target); err != nil {
		return err
	}
	event := &dbm.LicenseEvent{
		TenantUuid: tenantID,
		LicenseID:  licenseID,
		EventType:  dbm.LicenseEventOfflineExtend,
		EventPayload: toJSONMap(map[string]any{
			"offline_until": target,
		}),
	}
	return s.licenseRepo.CreateEvent(ctx, event)
}

func hasEmpty(values ...string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return true
		}
	}
	return false
}

func statusFromTrial(trial bool) string {
	if trial {
		return dbm.LicenseStatusTrial
	}
	return dbm.LicenseStatusActive
}

func generateToken(tenantID, listingID, planID string) string {
	payload := fmt.Sprintf("%s|%s|%s|%s", tenantID, listingID, planID, uuid.NewString())
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func stringPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	value := s
	return &value
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func parseExchangeRate(meta map[string]any) float64 {
	if meta == nil {
		return 0
	}
	val, ok := meta["exchange_rate"]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f
		}
	}
	return 0
}

func (s *LicenseService) offlineAllowance() time.Duration {
	if s == nil || s.cfg == nil {
		return 72 * time.Hour
	}
	if window := s.cfg.LicenseOfflineAllowance(); window > 0 {
		return window
	}
	return 72 * time.Hour
}

func (s *LicenseService) reminderLead() time.Duration {
	if s == nil || s.cfg == nil {
		return 72 * time.Hour
	}
	return s.cfg.LicenseReminderLead()
}

func (s *LicenseService) reminderChannels() []string {
	if s == nil || s.cfg == nil {
		return []string{"email"}
	}
	return s.cfg.LicenseReminderChannels()
}

func (s *LicenseService) emitRenewalScheduled(license *dbm.License) {
	if s == nil || s.logger == nil || license == nil {
		return
	}
	lead := s.reminderLead()
	if lead <= 0 {
		return
	}
	scheduleAt := license.ExpiresAt.Add(-lead)
	if scheduleAt.Before(time.Now()) {
		scheduleAt = time.Now()
	}
	marketobs.EmitLicenseRenewalScheduled(s.logger, license, scheduleAt, s.reminderChannels())
}

func licenseCacheTTL(license *dbm.License, allowance time.Duration) time.Duration {
	if license == nil {
		return time.Hour
	}
	target := license.ExpiresAt
	if license.OfflineUntil != nil && !license.OfflineUntil.IsZero() && license.OfflineUntil.Before(target) {
		target = *license.OfflineUntil
	}
	ttl := time.Until(target)
	if ttl <= 0 {
		return time.Minute
	}
	if allowance > 0 && ttl > allowance {
		ttl = allowance
	}
	return ttl
}

func billingIDFromMeta(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta["billing_id"]; ok && v != nil {
		switch val := v.(type) {
		case string:
			return strings.TrimSpace(val)
		case fmt.Stringer:
			return strings.TrimSpace(val.String())
		}
	}
	return ""
}
