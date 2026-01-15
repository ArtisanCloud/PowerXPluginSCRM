package http

import "testing"

func TestMarketplaceRBACEntriesContainListings(t *testing.T) {
	entries := marketplacePublicRBACEntries("/api/v1")
	perm, ok := entries["GET:/api/v1/marketplace/listings"]
	if !ok {
		t.Fatalf("expected listings GET entry in RBAC map, got %+v", entries)
	}
	if perm.Resource != "marketplace.listings" || perm.Action != "read" {
		t.Fatalf("unexpected permission %+v", perm)
	}
}
