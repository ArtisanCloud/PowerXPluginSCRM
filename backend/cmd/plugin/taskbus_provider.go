package main

import (
	"context"
	"errors"
	"os"
	"strings"

	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	fwtaskbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskbus"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	powerxclient "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/grpc/client"
	runtimeswitch "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/runtime/switches"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/sirupsen/logrus"
)

type notConfiguredTaskBusProvider struct {
	reason string
}

func (p notConfiguredTaskBusProvider) NewEmitter() (fweventbridge.Emitter, error) {
	reason := strings.TrimSpace(p.reason)
	if reason == "" {
		reason = "host taskbus provider is not configured"
	}
	return nil, errors.New(reason)
}

func resolveTaskBusProvider(cfg *config.Config, log *logrus.Entry) fweventbridge.TaskBusProvider {
	if !runtimeswitch.IsHostDriver(runtimeswitch.Resolve(cfg).TaskBus) {
		return notConfiguredTaskBusProvider{reason: "runtime taskbus driver is not host"}
	}

	if cfg == nil || cfg.Gateway == nil {
		return notConfiguredTaskBusProvider{reason: "gateway config is missing"}
	}

	baseURL := strings.TrimSpace(cfg.Gateway.BaseURL)
	authScheme := strings.ToLower(strings.TrimSpace(cfg.Gateway.AuthScheme))
	if authScheme == "" {
		authScheme = "bearer"
	}
	hasSTS := cfg.GRPCUpstream != nil &&
		strings.TrimSpace(cfg.GRPCUpstream.STSClientID) != "" &&
		strings.TrimSpace(cfg.GRPCUpstream.STSClientSecret) != ""
	credentialOK := authScheme == "bearer" && hasSTS
	if baseURL == "" || !credentialOK {
		if log != nil {
			log.WithFields(logrus.Fields{
				"gateway_base_url": baseURL,
				"gateway_auth":     authScheme,
				"has_sts":          hasSTS,
			}).Warn("TaskBus host provider unavailable; missing gateway credentials")
		}
		return notConfiguredTaskBusProvider{reason: "gateway base_url and STS credential are required"}
	}

	sourcePlugin := app.PluginID
	if cfg.EventBridge != nil && strings.TrimSpace(cfg.EventBridge.SourcePlugin) != "" {
		sourcePlugin = strings.TrimSpace(cfg.EventBridge.SourcePlugin)
	}

	payloadVersion := "v1"
	if cfg.EventBridge != nil && strings.TrimSpace(cfg.EventBridge.PayloadVersion) != "" {
		payloadVersion = strings.TrimSpace(cfg.EventBridge.PayloadVersion)
	}

	tenantUUID := strings.TrimSpace(cfg.Gateway.TenantUUID)
	if strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" {
		tenantUUID = ""
	}

	return fwtaskbus.NewHostProvider(fwtaskbus.HostProviderConfig{
		BaseURL:        baseURL,
		APIPrefix:      strings.TrimSpace(cfg.Gateway.APIPrefix),
		TokenProvider:  hostTaskBusTokenProvider(cfg, log),
		TenantUUID:     tenantUUID,
		UserAgent:      strings.TrimSpace(cfg.Gateway.UserAgent),
		Timeout:        cfg.Gateway.Timeout,
		PayloadVersion: payloadVersion,
		SourcePlugin:   sourcePlugin,
	})
}

func hostTaskBusTokenProvider(cfg *config.Config, log *logrus.Entry) func(context.Context) (string, error) {
	if cfg == nil || cfg.GRPCUpstream == nil {
		return nil
	}
	return func(ctx context.Context) (string, error) {
		return exchangeHostBearerForTaskBus(ctx, cfg, log)
	}
}

func exchangeHostBearerForTaskBus(ctx context.Context, cfg *config.Config, log *logrus.Entry) (string, error) {
	if cfg == nil || cfg.GRPCUpstream == nil {
		return "", errors.New("grpc upstream is not configured")
	}
	client, err := powerxclient.NewPowerXServiceClient(ctx, cfg.GRPCUpstream)
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("TaskBus host provider unavailable; STS client init failed")
		}
		return "", err
	}
	token, err := client.AccessToken(ctx)
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("TaskBus host provider unavailable; STS exchange failed")
		}
		return "", err
	}
	return strings.TrimSpace(token), nil
}
