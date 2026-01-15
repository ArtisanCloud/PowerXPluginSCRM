package marketplace

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

func TestNewTaxProviderClientUnsupported(t *testing.T) {
	cfg := &config.Config{
		Integration: &config.IntegrationConfig{
			Billing: config.IntegrationBillingConfig{
				TaxProvider: "unknown",
			},
		},
	}

	_, err := NewTaxProviderClient(cfg, nil, nil)
	if err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
}

func TestCreateTransactionNotImplemented(t *testing.T) {
	cfg := &config.Config{
		Integration: &config.IntegrationConfig{
			Billing: config.IntegrationBillingConfig{
				TaxProvider: "stripe_tax",
			},
		},
	}

	client, err := NewTaxProviderClient(cfg, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, err = client.CreateTransaction(context.Background(), &TaxChargeRequest{Currency: "USD"})
	if err == nil {
		t.Fatal("expected ErrNotImplemented, got nil")
	}
	if err != ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestAmountToMinorUnits(t *testing.T) {
	units, err := AmountToMinorUnits("JPY", 1234)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if units != 1234 {
		t.Fatalf("expected 1234 units for JPY, got %d", units)
	}

	units, err = AmountToMinorUnits("USD", 12.34)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if units != 1234 {
		t.Fatalf("expected 1234 cents for USD, got %d", units)
	}

	if _, err = AmountToMinorUnits("", 10); err == nil {
		t.Fatal("expected error for empty currency")
	}

	value := MinorUnitsToAmount("BHD", 12345)
	if value != 12.345 {
		t.Fatalf("expected 12.345 for BHD minor conversion, got %f", value)
	}
}
