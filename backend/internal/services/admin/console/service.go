package console

import "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"

// Service aggregates shared dependencies for admin console workflows.
type Service struct {
	deps         *app.Deps
	jobService   *JobService
	safeOps      *SafeOpsService
	troubleshoot *TroubleshootService
	help         *HelpService
}

// NewService creates a new console Service placeholder.
func NewService(deps *app.Deps) *Service {
	return &Service{deps: deps}
}

// Jobs returns the JobService singleton.
func (s *Service) Jobs() *JobService {
	if s == nil {
		return nil
	}
	if s.jobService == nil {
		s.jobService = NewJobService(s.deps)
	}
	return s.jobService
}

// SafeOps returns the SafeOpsService singleton.
func (s *Service) SafeOps() *SafeOpsService {
	if s == nil {
		return nil
	}
	if s.safeOps == nil {
		s.safeOps = NewSafeOpsService(s.deps, s.Jobs())
	}
	return s.safeOps
}

// Troubleshoot returns the TroubleshootService singleton.
func (s *Service) Troubleshoot() *TroubleshootService {
	if s == nil {
		return nil
	}
	if s.troubleshoot == nil {
		help := s.Help()
		troubleshoot := NewTroubleshootService(s.deps, WithGuidanceSource(help))
		s.troubleshoot = troubleshoot
	}
	return s.troubleshoot
}

// Help returns contextual help service.
func (s *Service) Help() *HelpService {
	if s == nil {
		return nil
	}
	if s.help == nil {
		s.help = NewHelpService()
	}
	return s.help
}
