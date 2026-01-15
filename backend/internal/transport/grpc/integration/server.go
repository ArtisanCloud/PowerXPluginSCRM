package integration

import (
	integrationService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Server 提供 integration gRPC 服务注册入口。
type Server struct {
	dispatch *integrationService.DispatchService
	logger   *logrus.Entry
}

// NewServer 构造 integration gRPC 服务。
func NewServer(dispatch *integrationService.DispatchService, logger *logrus.Entry) *Server {
	return &Server{
		dispatch: dispatch,
		logger:   logger,
	}
}

// Register 将 integration 服务注册到 gRPC Server。
func (s *Server) Register(registrar grpc.ServiceRegistrar) {
	if registrar == nil {
		return
	}
	if s.dispatch == nil {
		s.log().Warn("integration gRPC dispatch service not configured")
		return
	}
	// TODO: 注册实际的 integration proto 服务，一旦 gRPC 接口稳定。
	s.log().Info("integration gRPC dispatch server registered (placeholder)")
}

func (s *Server) log() *logrus.Entry {
	if s.logger != nil {
		return s.logger
	}
	return logrus.WithField("component", "integration.grpc")
}
