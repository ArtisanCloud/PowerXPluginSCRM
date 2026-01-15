package templates

import (
	"context"
	"fmt"
	"strings"
	"time"

	entmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	trepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/template"
	"gorm.io/gorm"
)

type TemplateService struct {
	TemplateRepo *trepo.TemplateRepository
}

const (
	TemplateStatusDraft     = "draft"
	TemplateStatusPublished = "published"
	TemplateStatusArchived  = "archived"

	TemplateReviewPending  = "pending"
	TemplateReviewApproved = "approved"
	TemplateReviewRejected = "rejected"
)

// BatchCloneOptions controls naming behaviors when cloning templates in bulk.
type BatchCloneOptions struct {
	NamePrefix        string
	DescriptionPrefix string
}

// BatchCloneFailure 表示单个源模板的克隆失败记录。
type BatchCloneFailure struct {
	SourceID uint64 `json:"source_id"`
	Reason   string `json:"reason"`
}

// BatchCloneResult 汇总批量克隆的成功与失败 ID。
type BatchCloneResult struct {
	CreatedIDs []uint64            `json:"created_ids"`
	Failed     []BatchCloneFailure `json:"failed"`
}

// ValidationViolation represents a single lint failure.
type ValidationViolation struct {
	Rule     string `json:"rule"`
	Field    string `json:"field"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// ValidationResult aggregates violations for a template.
type ValidationResult struct {
	TemplateID uint64                `json:"template_id"`
	Valid      bool                  `json:"valid"`
	Violations []ValidationViolation `json:"violations"`
}

func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{TemplateRepo: trepo.NewTemplateRepository(db)}
}

func (s *TemplateService) List(
	ctx context.Context,
	q string,
	page, pageSize int,
) (*repository.Page[[]*dbm.Template], error) {
	cb := func(db *gorm.DB, opt interface{}) *gorm.DB {
		if kw, _ := opt.(string); strings.TrimSpace(kw) != "" {
			p := "%" + strings.TrimSpace(kw) + "%"
			db = db.Where("(name ILIKE ? OR description ILIKE ?)", p, p)
		}
		return db.Order("id DESC")
	}

	return s.TemplateRepo.FindPage(ctx, nil, page, pageSize, cb, q)
}

func (s *TemplateService) GetByID(ctx context.Context, id uint64) (*dbm.Template, error) {
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	tpl, err := s.TemplateRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return tpl, nil
}

func (s *TemplateService) Create(
	ctx context.Context,
	name, description, content string,
) (*dbm.Template, error) {
	tpl := &dbm.Template{
		Name:         name,
		Description:  description,
		Content:      content,
		Status:       TemplateStatusDraft,
		ReviewStatus: TemplateReviewPending,
	}

	return s.TemplateRepo.Create(ctx, tpl)
}

func (s *TemplateService) Update(
	ctx context.Context,
	id uint64,
	name, description, content string,
) (*dbm.Template, error) {
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	fields := map[string]interface{}{
		"name":        name,
		"description": description,
		"content":     content,
	}
	return s.TemplateRepo.UpdateByID(ctx, id, fields)
}

func (s *TemplateService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return gorm.ErrInvalidData
	}
	return s.TemplateRepo.DeleteByID(ctx, id)
}

// BatchClone 执行批量克隆，最多复制 50 份，允许部分失败。
func (s *TemplateService) BatchClone(ctx context.Context, sourceIDs []uint64, copies int, opts BatchCloneOptions) (*BatchCloneResult, error) {
	if s == nil || s.TemplateRepo == nil {
		return nil, gorm.ErrInvalidDB
	}
	if len(sourceIDs) == 0 {
		return nil, gorm.ErrInvalidData
	}
	if copies <= 0 {
		copies = 1
	}
	if copies > 50 {
		copies = 50
	}
	result := &BatchCloneResult{
		CreatedIDs: make([]uint64, 0, len(sourceIDs)*copies),
		Failed:     make([]BatchCloneFailure, 0),
	}
	namePrefix := strings.TrimSpace(opts.NamePrefix)
	descPrefix := strings.TrimSpace(opts.DescriptionPrefix)

	for _, srcID := range sourceIDs {
		tpl, err := s.GetByID(ctx, srcID)
		if err != nil {
			result.Failed = append(result.Failed, BatchCloneFailure{SourceID: srcID, Reason: err.Error()})
			continue
		}
		for i := 0; i < copies; i++ {
			clone := &dbm.Template{
				BaseModel:    entmodels.BaseModel{TenantUuid: tpl.TenantUuid},
				Name:         buildCloneName(namePrefix, tpl.Name, i, copies),
				Description:  buildCloneDescription(descPrefix, tpl.Description),
				Content:      tpl.Content,
				Status:       TemplateStatusDraft,
				ReviewStatus: TemplateReviewPending,
			}
			saved, err := s.TemplateRepo.Create(ctx, clone)
			if err != nil {
				result.Failed = append(result.Failed, BatchCloneFailure{SourceID: srcID, Reason: err.Error()})
				continue
			}
			result.CreatedIDs = append(result.CreatedIDs, saved.ID)
		}
	}

	if result.Failed == nil {
		result.Failed = []BatchCloneFailure{}
	}
	return result, nil
}

func (s *TemplateService) MarkReviewed(ctx context.Context, id uint64, reviewer, comment string, approved bool) (*dbm.Template, error) {
	if s == nil || s.TemplateRepo == nil {
		return nil, gorm.ErrInvalidDB
	}
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	status := TemplateReviewRejected
	if approved {
		status = TemplateReviewApproved
	}
	fields := map[string]interface{}{
		"review_status":  status,
		"review_comment": strings.TrimSpace(comment),
		"reviewed_by":    strings.TrimSpace(reviewer),
		"reviewed_at":    time.Now().UTC(),
	}
	return s.TemplateRepo.UpdateByID(ctx, id, fields)
}

func (s *TemplateService) Publish(ctx context.Context, id uint64, channel string) (*dbm.Template, error) {
	if s == nil || s.TemplateRepo == nil {
		return nil, gorm.ErrInvalidDB
	}
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	fields := map[string]interface{}{
		"status":          TemplateStatusPublished,
		"publish_channel": strings.TrimSpace(channel),
		"published_at":    time.Now().UTC(),
	}
	return s.TemplateRepo.UpdateByID(ctx, id, fields)
}

func (s *TemplateService) Cleanup(ctx context.Context, id uint64, reason string) (*dbm.Template, error) {
	if s == nil || s.TemplateRepo == nil {
		return nil, gorm.ErrInvalidDB
	}
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	fields := map[string]interface{}{
		"status":         TemplateStatusArchived,
		"cleanup_reason": strings.TrimSpace(reason),
		"cleaned_at":     time.Now().UTC(),
	}
	return s.TemplateRepo.UpdateByID(ctx, id, fields)
}

// Validate 根据规则检查模板内容。
func (s *TemplateService) Validate(ctx context.Context, id uint64, rules []string, strict bool) (*ValidationResult, error) {
	if s == nil || s.TemplateRepo == nil {
		return nil, gorm.ErrInvalidDB
	}
	if id == 0 {
		return nil, gorm.ErrInvalidData
	}
	tpl, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	normalized := normalizeRules(rules)
	if len(normalized) == 0 {
		normalized = []string{"name_not_empty", "content_min_length", "description_not_empty"}
	}

	res := &ValidationResult{
		TemplateID: id,
		Valid:      true,
		Violations: make([]ValidationViolation, 0),
	}

	for _, rule := range normalized {
		switch rule {
		case "name_not_empty":
			if strings.TrimSpace(tpl.Name) == "" {
				res.Violations = append(res.Violations, ValidationViolation{
					Rule:     rule,
					Field:    "name",
					Severity: severity(strict, "error"),
					Message:  "模板名称不能为空",
				})
			}
		case "content_min_length":
			if len(strings.TrimSpace(tpl.Content)) < 10 {
				res.Violations = append(res.Violations, ValidationViolation{
					Rule:     rule,
					Field:    "content",
					Severity: severity(strict, "warning"),
					Message:  "模板内容应不少于 10 个字符",
				})
			}
		case "description_not_empty":
			if strings.TrimSpace(tpl.Description) == "" {
				res.Violations = append(res.Violations, ValidationViolation{
					Rule:     rule,
					Field:    "description",
					Severity: severity(strict, "warning"),
					Message:  "建议补充模板描述方便检索",
				})
			}
		default:
			// 未知规则，记录提示便于前端调试
			res.Violations = append(res.Violations, ValidationViolation{
				Rule:     rule,
				Field:    "*",
				Severity: "info",
				Message:  fmt.Sprintf("rule %s 未实现，已忽略", rule),
			})
		}
	}

	if len(res.Violations) > 0 {
		// 过滤掉 info 级别以外的违规才判定 false
		res.Valid = true
		for _, v := range res.Violations {
			if v.Severity == "error" || v.Severity == "warning" {
				res.Valid = false
				break
			}
		}
	}

	return res, nil
}

func buildCloneName(prefix, base string, idx, copies int) string {
	name := strings.TrimSpace(base)
	if name == "" {
		name = "Template"
	}
	if prefix = strings.TrimSpace(prefix); prefix != "" {
		name = prefix + " " + name
	}
	if copies > 1 {
		return fmt.Sprintf("%s #%d", name, idx+1)
	}
	return name
}

func buildCloneDescription(prefix, base string) string {
	base = strings.TrimSpace(base)
	if prefix = strings.TrimSpace(prefix); prefix != "" {
		if base == "" {
			return prefix
		}
		return prefix + " " + base
	}
	return base
}

func normalizeRules(rules []string) []string {
	if len(rules) == 0 {
		return nil
	}
	uniq := make([]string, 0, len(rules))
	seen := map[string]struct{}{}
	for _, rule := range rules {
		if r := strings.TrimSpace(strings.ToLower(rule)); r != "" {
			if _, ok := seen[r]; ok {
				continue
			}
			seen[r] = struct{}{}
			uniq = append(uniq, r)
		}
	}
	return uniq
}

func severity(strict bool, defaultLevel string) string {
	if strict && defaultLevel != "error" {
		return "error"
	}
	return defaultLevel
}
