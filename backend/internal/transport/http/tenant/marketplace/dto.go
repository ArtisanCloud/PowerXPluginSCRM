package marketplace

import (
	"strconv"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
)

type createLicenseRequest struct {
	TenantUuid        string `json:"tenant_uuid"`
	ListingID         string `json:"listing_id" binding:"required"`
	PlanID            string `json:"plan_id" binding:"required"`
	PaymentIntentID   string `json:"payment_intent_id" binding:"required"`
	TrialOverrideDays *int   `json:"trial_override_days"`
}

type renewLicenseRequest struct {
	RenewalToken string `json:"renewal_token"`
	PlanID       string `json:"plan_id"`
}

type offlineExtendRequest struct {
	RequestedHours int `json:"requested_hours" binding:"required,gte=1,lte=72"`
}

type licenseResponse struct {
	ID                 string   `json:"id"`
	ListingID          string   `json:"listing_id"`
	PlanID             string   `json:"plan_id"`
	Status             string   `json:"status"`
	ExpiresAt          string   `json:"expires_at"`
	Token              string   `json:"token"`
	OfflineUntil       *string  `json:"offline_until,omitempty"`
	RenewalToken       string   `json:"renewal_token,omitempty"`
	SettlementCurrency string   `json:"settlement_currency,omitempty"`
	ExchangeRate       *float64 `json:"exchange_rate,omitempty"`
}

func newLicenseResponse(license *dbm.License) *licenseResponse {
	if license == nil {
		return nil
	}
	resp := &licenseResponse{
		ID:        license.ID,
		ListingID: license.ListingID,
		PlanID:    license.PlanID,
		Status:    license.Status,
		ExpiresAt: license.ExpiresAt.UTC().Format(time.RFC3339),
		Token:     license.LicenseToken,
		RenewalToken: func() string {
			if license.RenewalToken != nil {
				return *license.RenewalToken
			}
			return ""
		}(),
	}
	if license.OfflineUntil != nil {
		val := license.OfflineUntil.UTC().Format(time.RFC3339)
		resp.OfflineUntil = &val
	}
	if currency, ok := license.Metadata["settlement_currency"].(string); ok && currency != "" {
		resp.SettlementCurrency = currency
	}
	if rate, ok := license.Metadata["exchange_rate"].(float64); ok && rate > 0 {
		resp.ExchangeRate = &rate
	} else if str, ok := license.Metadata["exchange_rate"].(string); ok {
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			resp.ExchangeRate = &f
		}
	}
	return resp
}
