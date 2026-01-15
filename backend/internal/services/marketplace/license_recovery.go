package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// LicenseRecoveryService reconciles delayed or inconsistent license issuance flows.
type LicenseRecoveryService struct {
	licenseService *LicenseService
	licenseRepo    *mrepo.LicenseRepository
	pricingRepo    *mrepo.PricingRepository
	logger         *logrus.Entry
	defaultActor   string
}

// RecoveryRequest describes the payload required to reconcile a delayed issuance.
type RecoveryRequest struct {
	TenantUuid string
	ListingID  string
	PlanID     string
	BillingID  string
	IssuedBy   string
	Metadata   map[string]any
}

// NewLicenseRecoveryService constructs a recovery coordinator.
func NewLicenseRecoveryService(licenseService *LicenseService, licenseRepo *mrepo.LicenseRepository, pricingRepo *mrepo.PricingRepository, logger *logrus.Entry) *LicenseRecoveryService {
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace_license_recovery")
	}
	return &LicenseRecoveryService{
		licenseService: licenseService,
		licenseRepo:    licenseRepo,
		pricingRepo:    pricingRepo,
		logger:         logger,
		defaultActor:   "marketplace.recovery",
	}
}

// RecoverIssuance re-creates a missing license from a successful billing transaction.
func (s *LicenseRecoveryService) RecoverIssuance(ctx context.Context, req RecoveryRequest) (*dbm.License, bool, error) {
	if s == nil || s.licenseService == nil || s.licenseRepo == nil {
		return nil, false, errors.New("recovery service not fully configured")
	}
	if hasEmpty(req.TenantUuid, req.ListingID, req.PlanID, req.BillingID) {
		return nil, false, errors.New("tenant_uuid, listing_id, plan_id and billing_id are required")
	}

	existing, err := s.licenseRepo.FindByBillingID(ctx, req.TenantUuid, req.BillingID)
	if err == nil && existing != nil {
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	if s.pricingRepo != nil {
		plan, err := s.pricingRepo.GetPlan(ctx, req.TenantUuid, req.PlanID)
		if err != nil {
			return nil, false, err
		}
		if plan != nil && !strings.EqualFold(plan.ListingID, req.ListingID) {
			return nil, false, errors.New("plan does not belong to listing for recovery")
		}
	}

	metadata := map[string]any{
		"billing_id": req.BillingID,
		"recovery":   true,
	}
	for k, v := range req.Metadata {
		metadata[k] = v
	}
	if _, ok := metadata["recovery_at"]; !ok {
		metadata["recovery_at"] = time.Now().UTC().Format(time.RFC3339)
	}

	issuedBy := strings.TrimSpace(req.IssuedBy)
	if issuedBy == "" {
		issuedBy = s.defaultActor
	}

	license, err := s.licenseService.IssueLicense(ctx, IssueLicenseParams{
		TenantUuid:  req.TenantUuid,
		ListingID:   req.ListingID,
		PlanID:      req.PlanID,
		IssuedBy:    issuedBy,
		Metadata:    metadata,
		SkipBilling: true,
	})
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"tenant_uuid": req.TenantUuid,
				"listing_id":  req.ListingID,
				"plan_id":     req.PlanID,
				"billing_id":  req.BillingID,
			}).Error("license recovery failed")
		}
		return nil, false, err
	}

	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"tenant_uuid": req.TenantUuid,
			"listing_id":  req.ListingID,
			"plan_id":     req.PlanID,
			"license_id":  license.ID,
			"billing_id":  req.BillingID,
		}).Info("license recovered after billing delay")
	}
	return license, true, nil
}
