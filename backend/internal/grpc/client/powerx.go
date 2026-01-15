package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	cfgpkg "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	v1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	protojson "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	commonv1 "github.com/ArtisanCloud/PowerX/api/grpc/gen/go/common/v1"
	stsv1 "github.com/ArtisanCloud/PowerX/api/grpc/gen/go/powerx/auth/sts/v1"
)

// 通用请求和响应结构（与 PowerX proto 兼容）
type RequestContext struct {
	TenantUUID  string `json:"tenant_uuid"`
	AccessToken string `json:"access_token"`
}

type PageRequest struct {
	PageIndex int32 `json:"page_index"`
	PageSize  int32 `json:"page_size"`
}

// PowerX PowerX gRPC 客户端封装
type PowerXServiceClient struct {
	conn *grpc.ClientConn
	// 基于 proto 的强类型客户端
	sts stsv1.STSServiceClient

	token          string
	tenantUUID     string
	tenantLegacyID int64
	cfg            *cfgpkg.GRPCUpstream
	tm             *TokenManager
}

// NewPowerX 根据配置拨号 PowerX gRPC
func NewPowerXServiceClient(ctx context.Context, c *cfgpkg.GRPCUpstream) (*PowerXServiceClient, error) {
	if c.Address == "" {
		return nil, fmt.Errorf("grpc upstream address is required")
	}

	tenantUUID := strings.TrimSpace(strings.ToLower(c.TenantUUID))
	p := &PowerXServiceClient{
		conn:           nil,
		token:          c.Token,
		tenantUUID:     tenantUUID,
		tenantLegacyID: parseLegacyTenantUuid(tenantUUID),
		cfg:            c,
	}

	// eager 模式：启动即连接；lazy 模式：首次调用再连接
	if c.ConnectMode == "" || c.ConnectMode == "eager" {
		if err := p.ensureConn(ctx); err != nil {
			if c.Optional {
				logger.WithError(err).Warn("gRPC upstream not available; continue in optional mode")
			} else {
				return nil, err
			}
		}
	}

	// 初始化 STS TokenManager（若提供了 client_id/secret）
	if c.STSClientID != "" && c.STSClientSecret != "" {
		p.tm = NewTokenManager(c.STSClientID, c.STSClientSecret, c.STSAudience, c.STSScope, c.STSTTL, func(ctx context.Context, req *STSExchangeRequest) (*STSExchangeResponse, error) {
			// 使用 proto 强类型客户端调用 STS Exchange
			if p.conn == nil {
				if err := p.ensureConn(ctx); err != nil {
					return nil, err
				}
			}
			// STS Exchange 通常不带原有 Authorization
			// 但仍可携带租户信息到 Ctx
			ctxPayload := &commonv1.RequestContext{}
			if p.tenantLegacyID > 0 {
				ctxPayload.TenantId = p.tenantLegacyID
			}
			if p.tenantUUID != "" {
				if ctxPayload.Attributes == nil {
					ctxPayload.Attributes = map[string]string{}
				}
				ctxPayload.Attributes["tenant_uuid"] = p.tenantUUID
			}
			pr := &stsv1.ExchangeRequest{
				Ctx:          ctxPayload,
				ClientId:     req.ClientID,
				ClientSecret: req.ClientSecret,
				Audience:     req.Audience,
				Scope:        req.Scope,
				TtlSeconds:   req.TTL,
			}
			// 直接用底层连接构造 client（或使用 p.sts）
			stscli := p.sts
			if stscli == nil {
				stscli = stsv1.NewSTSServiceClient(p.conn)
			}
			resp, err := stscli.Exchange(ctx, pr)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Data == nil {
				return nil, fmt.Errorf("empty response from STS Exchange")
			}
			return &STSExchangeResponse{AccessToken: resp.Data.GetAccessToken(), ExpiresIn: int32(resp.Data.GetExpiresIn())}, nil
		})
	}

	return p, nil
}

// ensureConn: 若未连接或不可用，尝试拨号（短超时）。
func (p *PowerXServiceClient) ensureConn(ctx context.Context) error {
	if p.conn != nil {
		st := p.conn.GetState().String()
		if st == "READY" || st == "IDLE" || st == "CONNECTING" {
			return nil
		}
	}
	// 重新拨号
	dialOpts := []grpc.DialOption{}
	if p.cfg.UseTLS {
		var creds credentials.TransportCredentials
		if p.cfg.CACert != "" {
			pem, err := os.ReadFile(p.cfg.CACert)
			if err != nil {
				return fmt.Errorf("read ca cert: %w", err)
			}
			cp := x509.NewCertPool()
			if !cp.AppendCertsFromPEM(pem) {
				return fmt.Errorf("failed to append ca cert")
			}
			creds = credentials.NewTLS(&tls.Config{RootCAs: cp})
		} else {
			creds = credentials.NewTLS(&tls.Config{})
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	// 非阻塞 + 短超时拨号（尝试立即建立；失败返回错误）
	ctxDial, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	logger.WithField("address", p.cfg.Address).Info("Dialing PowerX gRPC (on-demand)")
	//nolint:staticcheck // grpc.DialContext remains necessary until upstream migrates to the new API.
	conn, err := grpc.DialContext(ctxDial, p.cfg.Address, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to dial PowerX gRPC: %w", err)
	}
	p.conn = conn
	// 初始化强类型客户端
	p.sts = stsv1.NewSTSServiceClient(conn)
	return nil
}

// Close 关闭 gRPC 连接
func (p *PowerXServiceClient) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *PowerXServiceClient) RC() *RequestContext {
	return &RequestContext{
		TenantUUID:  p.tenantUUID,
		AccessToken: p.token,
	}
}

// STSClient 返回 STS 服务客户端
func (p *PowerXServiceClient) STSClient(ctx context.Context) (stsv1.STSServiceClient, error) {
	if p.conn == nil {
		if err := p.ensureConn(ctx); err != nil {
			return nil, err
		}
	}
	if p.sts == nil {
		p.sts = stsv1.NewSTSServiceClient(p.conn)
	}
	return p.sts, nil
}

// Outgoing 基础 ctx：附带 auth 头（未来加拦截器时无缝）
func (p *PowerXServiceClient) Outgoing(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{})

	// 优先使用 STS token，其次使用静态 Token
	bearer := p.token
	if p.tm != nil {
		if tok, err := p.tm.GetToken(ctx); err == nil && tok != "" {
			bearer = tok
		}
	}
	if bearer != "" {
		head := bearer
		if len(head) > 40 {
			head = bearer[:40]
		}
		// 临时调试
		log.Printf("[PLUGIN-GATE-TOKEN] upstream=%s token.head=%s...", p.cfg.Address, head)
		md.Set("authorization", "Bearer "+bearer)
	}

	if p.tenantUUID != "" {
		md.Set("x-powerx-tenant-uuid", p.tenantUUID)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// GetToken 获取认证 token
func (p *PowerXServiceClient) GetToken() string {
	if p.token != "" {
		return p.token
	}
	if p.tm != nil && p.tm.HasValid() {
		// 为避免阻塞，这里不触发刷新，只返回空串代表未知
		// 上层可通过 HealthCheck/实际调用来驱动刷新
		return "sts"
	}
	return ""
}

// HasToken 是否配置/具备可用的访问凭据（静态或临时）
func (p *PowerXServiceClient) HasToken() bool {
	if p.token != "" {
		return true
	}
	if p.tm != nil && p.tm.HasValid() {
		return true
	}
	return false
}

// GetTenantUUID 获取租户 UUID
func (p *PowerXServiceClient) GetTenantUUID() string {
	return p.tenantUUID
}

// IsConnected 检查连接状态
func (p *PowerXServiceClient) IsConnected() bool {
	if p.conn == nil {
		return false
	}
	return p.conn.GetState().String() == "READY"
}

// invokeGRPC 通用 gRPC 调用方法
func (p *PowerXServiceClient) InvokeGRPC(ctx context.Context, service, method string, req, resp interface{}) error {
	if p.conn == nil {
		if err := p.ensureConn(ctx); err != nil {
			return err
		}
	}

	// STS Exchange 不附带 Authorization，其它服务使用 Outgoing 附带 token
	if service == "powerx.auth.sts.v1.STSService" {
		// no auth header
	} else {
		ctx = p.Outgoing(ctx)
	}

	logger.WithField("service", service).WithField("method", method).Info("Calling PowerX gRPC service via reflection")

	// 1) 通过反射拉取包含该符号的文件描述
	refClient := v1alpha.NewServerReflectionClient(p.conn)
	stream, err := refClient.ServerReflectionInfo(ctx)
	if err != nil {
		return fmt.Errorf("open reflection stream: %w", err)
	}
	//nolint:staticcheck // reflection APIs rely on v1alpha descriptors until upstream replaces them.
	if err := stream.Send(&v1alpha.ServerReflectionRequest{
		MessageRequest: &v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: service,
		},
	}); err != nil {
		return fmt.Errorf("send reflection request: %w", err)
	}
	respMsg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv reflection response: %w", err)
	}
	//nolint:staticcheck // reflection response helpers remain deprecated but required.
	fdResp := respMsg.GetFileDescriptorResponse()
	if fdResp == nil {
		return fmt.Errorf("no file descriptor in reflection response")
	}
	var files []*descriptorpb.FileDescriptorProto
	//nolint:staticcheck // access to FileDescriptorProto is required for dynamic invocation.
	for _, b := range fdResp.FileDescriptorProto {
		fdp := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(b, fdp); err != nil {
			return fmt.Errorf("unmarshal FileDescriptorProto: %w", err)
		}
		files = append(files, fdp)
	}
	fileSet := &descriptorpb.FileDescriptorSet{File: files}
	r, err := protodesc.NewFiles(fileSet)
	if err != nil {
		return fmt.Errorf("build file descriptors: %w", err)
	}
	d, err := r.FindDescriptorByName(protoreflect.FullName(service))
	if err != nil {
		return fmt.Errorf("find service descriptor: %w", err)
	}
	sd, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return fmt.Errorf("descriptor is not a service: %s", service)
	}
	var md protoreflect.MethodDescriptor
	for i := 0; i < sd.Methods().Len(); i++ {
		m := sd.Methods().Get(i)
		if string(m.Name()) == method {
			md = m
			break
		}
	}
	if md == nil {
		return fmt.Errorf("method not found: %s", method)
	}

	inDesc := md.Input()
	outDesc := md.Output()
	inMsg := dynamicpb.NewMessage(inDesc)
	outMsg := dynamicpb.NewMessage(outDesc)

	// 将调用方的 req（Go struct）转为 map，并按字段名/jsonName 逐一赋值，兼容 client_id/clientId
	jb, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal req: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(jb, &m); err != nil {
		return fmt.Errorf("unmarshal req to map: %w", err)
	}
	in := inMsg.ProtoReflect()
	fds := inDesc.Fields()
	for i := 0; i < fds.Len(); i++ {
		fd := fds.Get(i)
		// 两种命名都试一下
		var v interface{}
		if vv, ok := m[string(fd.Name())]; ok {
			v = vv
		} else if vv, ok := m[fd.JSONName()]; ok {
			v = vv
		} else {
			continue
		}
		switch fd.Kind() {
		case protoreflect.StringKind:
			if s, ok := v.(string); ok {
				in.Set(fd, protoreflect.ValueOfString(s))
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			switch t := v.(type) {
			case float64:
				in.Set(fd, protoreflect.ValueOfInt32(int32(t)))
			case int:
				in.Set(fd, protoreflect.ValueOfInt32(int32(t)))
			case int32:
				in.Set(fd, protoreflect.ValueOfInt32(t))
			case string:
				// 尝试解析
				var iv int
				if _, err := fmt.Sscanf(t, "%d", &iv); err == nil {
					in.Set(fd, protoreflect.ValueOfInt32(int32(iv)))
				}
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			switch t := v.(type) {
			case float64:
				in.Set(fd, protoreflect.ValueOfInt64(int64(t)))
			case int64:
				in.Set(fd, protoreflect.ValueOfInt64(t))
			case string:
				var iv int64
				if _, err := fmt.Sscanf(t, "%d", &iv); err == nil {
					in.Set(fd, protoreflect.ValueOfInt64(iv))
				}
			}
		case protoreflect.BoolKind:
			if b, ok := v.(bool); ok {
				in.Set(fd, protoreflect.ValueOfBool(b))
			}
		default:
			// 对于其他标量类型，可按需补齐
		}
	}

	fullMethod := "/" + service + "/" + method
	if err := p.conn.Invoke(ctx, fullMethod, inMsg, outMsg); err != nil {
		return fmt.Errorf("grpc invoke %s: %w", fullMethod, err)
	}

	// 动态响应转 JSON，再映射到调用方 resp（Go struct）
	ob, err := protojson.MarshalOptions{UseEnumNumbers: true}.Marshal(outMsg)
	if err != nil {
		return fmt.Errorf("marshal dynamic response: %w", err)
	}
	// 兼容包裹结构：若顶层包含 {"data": {...}}，优先解开 data 再反序列化
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(ob, &envelope); err == nil {
		if data, ok := envelope["data"]; ok && len(data) > 0 {
			ob = data
		}
	}
	if err := json.Unmarshal(ob, resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

// ExchangeSTS 立即触发 STS Exchange（调试用）
func (p *PowerXServiceClient) ExchangeSTS(ctx context.Context) (token string, expiresIn int32, err error) {
	if p.tm == nil {
		return "", 0, fmt.Errorf("sts not configured")
	}
	tok, exp, err := p.tm.ExchangeNow(ctx)
	if err != nil {
		return "", 0, err
	}
	sec := int32(time.Until(exp).Seconds())
	if sec < 0 {
		sec = 0
	}
	return tok, sec, nil
}

// InvalidateSTS 使当前 STS token 失效，便于轮换后强制刷新
func (p *PowerXServiceClient) InvalidateSTS() {
	if p.tm != nil {
		p.tm.Invalidate()
	}
}

// HealthCheck 健康检查方法
func (p *PowerXServiceClient) HealthCheck(ctx context.Context) error {
	if p.conn == nil {
		if err := p.ensureConn(ctx); err != nil {
			return err
		}
	}

	state := p.conn.GetState()
	if state.String() != "READY" && state.String() != "IDLE" {
		return fmt.Errorf("grpc connection state is %s", state.String())
	}

	return nil
}

// Reconnect 重新连接
func (p *PowerXServiceClient) Reconnect(ctx context.Context) error {
	if p.conn != nil {
		p.conn.Close()
	}

	newClient, err := NewPowerXServiceClient(ctx, p.cfg)
	if err != nil {
		return err
	}

	*p = *newClient
	return nil
}

func parseLegacyTenantUuid(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return v
	}
	return 0
}
