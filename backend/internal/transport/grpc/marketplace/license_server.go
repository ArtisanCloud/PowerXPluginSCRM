package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	svc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/marketplace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// LicenseServer implements the gRPC interface bridging to the marketplace license service.
type LicenseServer struct {
	svc *svc.LicenseService
	UnimplementedLicenseServiceServer
}

// NewLicenseServer constructs a gRPC license server.
func NewLicenseServer(service *svc.LicenseService) *LicenseServer {
	return &LicenseServer{svc: service}
}

func (s *LicenseServer) Issue(ctx context.Context, req *IssueLicenseRequest) (*IssueLicenseResponse, error) {
	if s.svc == nil {
		return nil, status.Error(codes.Unavailable, "license service not configured")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	params := svc.IssueLicenseParams{
		TenantUuid: req.TenantId,
		ListingID:  req.ListingId,
		PlanID:     req.PlanId,
		IssuedBy:   firstNonEmpty(req.IssuedBy, req.TenantId),
		Trial:      req.Trial,
		Metadata:   map[string]any{},
	}
	if req.PaymentIntentId != "" {
		params.Metadata["payment_intent_id"] = req.PaymentIntentId
	}
	if req.ExpiresAtUnix > 0 {
		params.ExpiresAt = time.Unix(req.ExpiresAtUnix, 0).UTC()
	}

	license, err := s.svc.IssueLicense(ctx, params)
	if err != nil {
		return nil, mapLicenseError(err)
	}
	return &IssueLicenseResponse{License: toLicenseProto(license)}, nil
}

func (s *LicenseServer) Renew(ctx context.Context, req *RenewLicenseRequest) (*RenewLicenseResponse, error) {
	if s.svc == nil {
		return nil, status.Error(codes.Unavailable, "license service not configured")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	params := svc.RenewLicenseParams{
		TenantUuid:   req.TenantId,
		LicenseID:    req.LicenseId,
		PlanID:       req.PlanId,
		RenewalToken: req.RenewalToken,
		IssuedBy:     req.TenantId,
	}
	if req.ExpiresAtUnix > 0 {
		params.ExpiresAt = time.Unix(req.ExpiresAtUnix, 0).UTC()
	}

	license, err := s.svc.RenewLicense(ctx, params)
	if err != nil {
		return nil, mapLicenseError(err)
	}
	return &RenewLicenseResponse{License: toLicenseProto(license)}, nil
}

func (s *LicenseServer) Verify(ctx context.Context, req *VerifyLicenseRequest) (*VerifyLicenseResponse, error) {
	if s.svc == nil {
		return nil, status.Error(codes.Unavailable, "license service not configured")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	result, err := s.svc.VerifyLicense(ctx, req.TenantId, req.ListingId)
	if err != nil {
		return nil, mapLicenseError(err)
	}
	resp := &VerifyLicenseResponse{Valid: result.Valid, Reason: result.Reason}
	if result.License != nil {
		resp.License = toLicenseProto(result.License)
	}
	return resp, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func mapLicenseError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, err.Error())
	case strings.Contains(err.Error(), "not configured"):
		return status.Error(codes.Unavailable, err.Error())
	case strings.Contains(err.Error(), "invalid"):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toLicenseProto(l *dbm.License) *License {
	if l == nil {
		return nil
	}
	proto := &License{
		Id:            l.ID,
		TenantId:      l.TenantUuid,
		ListingId:     l.ListingID,
		PlanId:        l.PlanID,
		Status:        l.Status,
		Token:         l.LicenseToken,
		ExpiresAtUnix: l.ExpiresAt.UTC().Unix(),
		RenewalToken:  valueOrEmpty(l.RenewalToken),
	}
	if l.OfflineUntil != nil {
		proto.OfflineUntilUnix = l.OfflineUntil.UTC().Unix()
	}
	if currency, ok := l.Metadata["settlement_currency"].(string); ok {
		proto.SettlementCurrency = currency
	}
	if rate, ok := l.Metadata["exchange_rate"].(float64); ok {
		proto.ExchangeRate = rate
	}
	return proto
}

func valueOrEmpty(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
