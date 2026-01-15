package security

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	secmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
	secrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/security"
	secobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const DefaultAuditCommand = "make"
const DefaultAuditTarget = "security-audit"

// BaselineService orchestrates baseline checklists and audit runs.
type BaselineService struct {
	repo   *secrepo.Repository
	cfg    *config.Config
	logger *logrus.Entry
	runner func(ctx context.Context) error
}

// NewBaselineService constructs the service with the provided dependencies.
func NewBaselineService(db *gorm.DB, cfg *config.Config, logger *logrus.Entry) *BaselineService {
	svc := &BaselineService{
		repo:   secrepo.NewRepository(db),
		cfg:    cfg,
		logger: logger,
	}
	svc.runner = svc.defaultRunner
	return svc
}

// WithRunner overrides the execution runner (useful for tests).
func (s *BaselineService) WithRunner(runner func(ctx context.Context) error) {
	if runner != nil {
		s.runner = runner
	}
}

func (s *BaselineService) defaultRunner(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, DefaultAuditCommand, DefaultAuditTarget)
	return cmd.Run()
}

// UpsertChecklist persists a checklist manifest.
func (s *BaselineService) UpsertChecklist(ctx context.Context, version string, controls datatypes.JSONMap) (*secmodel.BaselineChecklist, error) {
	checklist := &secmodel.BaselineChecklist{
		Version:  version,
		Controls: controls,
	}
	return s.repo.UpsertChecklist(ctx, checklist)
}

// ListChecklists fetches existing baseline manifests.
func (s *BaselineService) ListChecklists(ctx context.Context) ([]*secmodel.BaselineChecklist, error) {
	return s.repo.ListChecklists(ctx)
}

// RunAudit executes the pipeline and records a report placeholder.
func (s *BaselineService) RunAudit(ctx context.Context, initiatedBy, checklistVersion string, baselineID string) (*secmodel.AuditReport, error) {
	if initiatedBy == "" {
		initiatedBy = "admin"
	}
	if checklistVersion == "" {
		checklistVersion = "unknown"
	}

	report, err := s.repo.CreateAuditReport(ctx, &secmodel.AuditReport{
		BaselineID:       baselineID,
		InitiatedBy:      initiatedBy,
		Status:           "RUNNING",
		ChecklistVersion: checklistVersion,
	})
	if err != nil {
		return nil, err
	}

	runErr := s.runner(ctx)
	status := "PASSED"
	findings := datatypes.JSONMap{"severity_counts": map[string]int{"critical": 0, "high": 0}}
	if runErr != nil {
		status = "FAILED"
		findings["error"] = runErr.Error()
		if s.logger != nil {
			s.logger.WithError(runErr).Warn("security audit command failed")
		}
	}
	if err := s.repo.UpdateAuditReportStatus(ctx, report.ID, status, findings); err != nil {
		return nil, err
	}
	report.Status = status
	report.Findings = findings
	report.CreatedAt = time.Now().UTC()
	secobs.RecordAuditReport(status)
	return report, nil
}

// RecordAuditResult updates an audit report with final details (artifact paths, findings map).
func (s *BaselineService) RecordAuditResult(ctx context.Context, id string, status string, artifactPath, sarifPath, hash string, findings map[string]interface{}) error {
	payload := datatypes.JSONMap(findings)
	if err := s.repo.UpdateAuditReportStatus(ctx, id, status, payload); err != nil {
		return err
	}
	secobs.RecordAuditReport(status)
	updates := map[string]interface{}{}
	if artifactPath != "" {
		updates["artifact_path"] = artifactPath
	}
	if sarifPath != "" {
		updates["sarif_path"] = sarifPath
	}
	if hash != "" {
		updates["report_hash"] = hash
	}
	return s.repo.UpdateAuditReportMetadata(ctx, id, updates)
}

// ListAuditReports returns reports sorted by time.
func (s *BaselineService) ListAuditReports(ctx context.Context, limit int) ([]*secmodel.AuditReport, error) {
	return s.repo.ListAuditReports(ctx, limit)
}

// MarshalFindings helper for writing to files.
func MarshalFindings(findings map[string]interface{}) ([]byte, error) {
	if findings == nil {
		findings = map[string]interface{}{}
	}
	return json.MarshalIndent(findings, "", "  ")
}
