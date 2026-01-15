package grpc

import (
	integrationtransport "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/grpc/integration"
	marketplacegrpc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/grpc/marketplace"
	templategrpc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/grpc/template"
	"google.golang.org/grpc"
)

// Registrar 聚合各模块的 gRPC 服务注册器。
type Registrar struct {
	Integration *integrationtransport.Server
	Marketplace marketplacegrpc.LicenseServiceServer
	Template    templategrpc.TemplateServiceServer
}

// Register 将可用的 gRPC 服务注册到给定 server。
func Register(server *grpc.Server, registrar Registrar) {
	if server == nil {
		return
	}
	if registrar.Integration != nil {
		registrar.Integration.Register(server)
	}
	if registrar.Marketplace != nil {
		marketplacegrpc.RegisterLicenseServiceServer(server, registrar.Marketplace)
	}
	if registrar.Template != nil {
		templategrpc.RegisterTemplateServiceServer(server, registrar.Template)
	}
}
