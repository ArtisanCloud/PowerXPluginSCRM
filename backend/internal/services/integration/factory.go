package integration

import (
	"time"

	idrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/mcp/stream"
	srvtemplates "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/templates"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/sirupsen/logrus"
)

// BuildDispatchService 根据应用依赖构造 DispatchService。
func BuildDispatchService(deps *app.Deps, logger *logrus.Entry) *DispatchService {
	if deps == nil {
		return nil
	}
	if logger == nil {
		logger = logrus.WithField("component", "integration.dispatch_factory")
	}

	loader := NewGrantMatrixLoader(
		deps.DB,
		logger.WithField("component", "integration.grant_matrix_loader"),
		LoaderOptions{},
	)
	grantService := NewGrantMatrixService(loader, logger.WithField("component", "integration.grant_matrix_service"))

	var fallbackStore idrepo.IdempotencyProvider
	if deps.DB != nil {
		ttl := 24 * time.Hour
		if deps.Config != nil {
			ttl = deps.Config.IntegrationIdempotencyTTL()
		}
		fallbackStore = idrepo.NewPostgresIdempotencyProvider(deps.DB, ttl)
	}
	repository := idrepo.NewIdempotencyRepository(deps.DB, nil, fallbackStore, logger.WithField("component", "integration.idempotency_repository"))

	templateService := srvtemplates.NewTemplateService(deps.DB)
	hostLogger := logger.WithField("component", "integration.host_invoker")
	noopInvoker := NewNoopInvoker(hostLogger)
	invoker := NewCapabilityInvoker(templateService, stream.DefaultBroker(), hostLogger, noopInvoker)

	return NewDispatchService(deps.Config, grantService, repository, invoker, logger.WithField("component", "integration.dispatch_service"))
}
