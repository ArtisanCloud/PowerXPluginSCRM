package templatepb

import (
	"context"
	"errors"
	"strings"

	dbtemplate "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	srvtemplates "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

const (
	defaultPageSize = 20
	defaultPage     = 1
	maxPageSize     = 200
)

// Server implements the TemplateService gRPC server.
type Server struct {
	UnimplementedTemplateServiceServer
	service *srvtemplates.TemplateService
}

// NewServer builds a Template gRPC server backed by the existing TemplateService.
func NewServer(service *srvtemplates.TemplateService) *Server {
	return &Server{service: service}
}

func (s *Server) ensureService() error {
	if s == nil || s.service == nil {
		return status.Error(codes.Unavailable, "template service not configured")
	}
	return nil
}

func (s *Server) ListTemplates(ctx context.Context, req *ListTemplatesRequest) (*ListTemplatesResponse, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if req == nil {
		req = &ListTemplatesRequest{}
	}
	ctx, err := s.ensureTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	page := normalizePage(req.GetPage())
	size := normalizePageSize(req.GetPageSize())

	result, err := s.service.List(ctx, req.GetQ(), page, size)
	if err != nil {
		return nil, mapTemplateError(err)
	}
	resp := &ListTemplatesResponse{
		Page:     int32(page),
		PageSize: int32(size),
		Total:    0,
		Items:    []*Template{},
	}
	if result != nil {
		resp.Page = int32(result.PageIndex)
		resp.PageSize = int32(result.PageSize)
		resp.Total = result.Total
		if len(result.List) > 0 {
			resp.Items = make([]*Template, 0, len(result.List))
			for _, tpl := range result.List {
				resp.Items = append(resp.Items, toProtoTemplate(tpl))
			}
		}
	}
	return resp, nil
}

func (s *Server) GetTemplate(ctx context.Context, req *GetTemplateRequest) (*TemplateResponse, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if req == nil || req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	ctx, err := s.ensureTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	tpl, err := s.service.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, mapTemplateError(err)
	}
	return &TemplateResponse{Template: toProtoTemplate(tpl)}, nil
}

func (s *Server) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*TemplateResponse, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if req == nil || strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	ctx, err := s.ensureTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	tpl, err := s.service.Create(ctx, req.GetName(), req.GetDescription(), req.GetContent())
	if err != nil {
		return nil, mapTemplateError(err)
	}
	return &TemplateResponse{Template: toProtoTemplate(tpl)}, nil
}

func (s *Server) UpdateTemplate(ctx context.Context, req *UpdateTemplateRequest) (*TemplateResponse, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if req == nil || req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	ctx, err := s.ensureTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	tpl, err := s.service.Update(ctx, req.GetId(), req.GetName(), req.GetDescription(), req.GetContent())
	if err != nil {
		return nil, mapTemplateError(err)
	}
	return &TemplateResponse{Template: toProtoTemplate(tpl)}, nil
}

func (s *Server) DeleteTemplate(ctx context.Context, req *DeleteTemplateRequest) (*emptypb.Empty, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if req == nil || req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	ctx, err := s.ensureTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	if err := s.service.Delete(ctx, req.GetId()); err != nil {
		return nil, mapTemplateError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) BatchCloneTemplates(ctx context.Context, req *BatchCloneTemplatesRequest) (*BatchCloneTemplatesResponse, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if req == nil || len(req.GetSourceIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "source_ids is required")
	}
	ctx, err := s.ensureTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	res, err := s.service.BatchClone(ctx, req.GetSourceIds(), int(req.GetCopies()), srvtemplates.BatchCloneOptions{
		NamePrefix:        req.GetNamePrefix(),
		DescriptionPrefix: req.GetDescriptionPrefix(),
	})
	if err != nil {
		return nil, mapTemplateError(err)
	}
	resp := &BatchCloneTemplatesResponse{CreatedIds: []uint64{}, Failed: []*BatchCloneFailure{}}
	if res != nil {
		if len(res.CreatedIDs) > 0 {
			resp.CreatedIds = append(resp.CreatedIds, res.CreatedIDs...)
		}
		if len(res.Failed) > 0 {
			resp.Failed = make([]*BatchCloneFailure, 0, len(res.Failed))
			for _, failure := range res.Failed {
				resp.Failed = append(resp.Failed, &BatchCloneFailure{SourceId: failure.SourceID, Reason: failure.Reason})
			}
		}
	}
	return resp, nil
}

func (s *Server) ValidateTemplate(ctx context.Context, req *ValidateTemplateRequest) (*ValidateTemplateResponse, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	if req == nil || req.GetId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	ctx, err := s.ensureTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	res, err := s.service.Validate(ctx, req.GetId(), req.GetRules(), req.GetStrict())
	if err != nil {
		return nil, mapTemplateError(err)
	}
	resp := &ValidateTemplateResponse{TemplateId: req.GetId(), Valid: true, Violations: []*ValidationViolation{}}
	if res != nil {
		resp.TemplateId = res.TemplateID
		resp.Valid = res.Valid
		if len(res.Violations) > 0 {
			resp.Violations = make([]*ValidationViolation, 0, len(res.Violations))
			for _, v := range res.Violations {
				resp.Violations = append(resp.Violations, &ValidationViolation{
					Rule:     v.Rule,
					Field:    v.Field,
					Severity: v.Severity,
					Message:  v.Message,
				})
			}
		}
	}
	return resp, nil
}

func (s *Server) ensureTenant(ctx context.Context, tenant string) (context.Context, error) {
	if tenant != "" {
		return authx.ContextWithTenantUUID(ctx, tenant), nil
	}
	if t, ok := authx.TenantUUIDFromContext(ctx); ok && t != "" {
		return ctx, nil
	}
	return ctx, status.Error(codes.InvalidArgument, "tenant_uuid is required")
}

func normalizePage(page int32) int {
	if page <= 0 {
		return defaultPage
	}
	return int(page)
}

func normalizePageSize(size int32) int {
	if size <= 0 {
		return defaultPageSize
	}
	if size > maxPageSize {
		return maxPageSize
	}
	return int(size)
}

func toProtoTemplate(tpl *dbtemplate.Template) *Template {
	if tpl == nil {
		return nil
	}
	proto := &Template{
		Id:          tpl.ID,
		Name:        tpl.Name,
		Description: tpl.Description,
		Content:     tpl.Content,
		TenantUuid:  tpl.TenantUuid,
	}
	if !tpl.CreatedAt.IsZero() {
		proto.CreatedAtUnix = tpl.CreatedAt.UTC().Unix()
	}
	if !tpl.UpdatedAt.IsZero() {
		proto.UpdatedAtUnix = tpl.UpdatedAt.UTC().Unix()
	}
	return proto
}

func mapTemplateError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, gorm.ErrInvalidData):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, repository.ErrTenantUuidRequired):
		return status.Error(codes.InvalidArgument, "tenant_uuid is required")
	case errors.Is(err, authx.ErrTenantMissing):
		return status.Error(codes.InvalidArgument, "tenant context missing")
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
