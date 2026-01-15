package powerx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/sirupsen/logrus"
)

// CapabilityClient implements capabilities.HostSyncClient against PowerX REST endpoints.
type CapabilityClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *logrus.Entry
}

// NewCapabilityClient creates a client with explicit base URL/token.
func NewCapabilityClient(baseURL, token string, log *logrus.Entry) capabilities.HostSyncClient {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil
	}
	if log == nil {
		log = logger.WithField("component", "capability_client")
	} else {
		log = log.WithField("component", "capability_client")
	}
	return &CapabilityClient{
		baseURL: base,
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: log,
	}
}

// NewCapabilityClientFromEnv builds a client using standard environment variables.
func NewCapabilityClientFromEnv(log *logrus.Entry) capabilities.HostSyncClient {
	base := fallbackEnv(
		"POWERX_CAPABILITY_SYNC_ENDPOINT",
		"PX_DEV_API_BASEURL",
		"PX_DEV_API_BASE",
	)
	token := fallbackEnv(
		"POWERX_CAPABILITY_SYNC_TOKEN",
		"PX_DEV_API_TOKEN",
	)
	return NewCapabilityClient(base, token, log)
}

func fallbackEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

// RegisterCatalog pushes the catalog snapshot and protocol assets to PowerX host.
func (c *CapabilityClient) RegisterCatalog(ctx context.Context, catalog *capabilities.CatalogSnapshot, assets []capabilities.ProtocolAsset) error {
	if catalog == nil {
		return fmt.Errorf("catalog snapshot is nil")
	}
	payload, err := c.buildRequestPayload(catalog, assets)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode catalog payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/plugins/capabilities/catalog", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build capability catalog request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("capability sync request failed: %w", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode/100 != 2 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("capability sync failed: %s %s", resp.Status, buf.String())
	}

	result := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.WithError(err).Warn("capability sync response decode failed")
	}

	c.logger.WithFields(logrus.Fields{
		"plugin_id": catalog.PluginID,
		"duration":  elapsed,
		"assets":    len(payload.Assets),
	}).Info("capability catalog synchronized with host")

	return nil
}

func (c *CapabilityClient) buildRequestPayload(catalog *capabilities.CatalogSnapshot, assets []capabilities.ProtocolAsset) (*catalogSyncRequest, error) {
	payload := &catalogSyncRequest{
		Catalog: catalog,
		Assets:  make([]catalogSyncAsset, 0, len(assets)),
	}
	for _, asset := range assets {
		if strings.TrimSpace(asset.Path) == "" {
			continue
		}
		content, err := os.ReadFile(asset.Path)
		if err != nil {
			c.logger.WithError(err).Warnf("skip asset %s: unable to read file", asset.Path)
			continue
		}
		info, err := os.Stat(asset.Path)
		if err != nil {
			c.logger.WithError(err).Warnf("skip asset %s: stat failed", asset.Path)
			continue
		}
		checksum := sha256.Sum256(content)
		payload.Assets = append(payload.Assets, catalogSyncAsset{
			Type:     asset.Type,
			Path:     filepath.ToSlash(asset.Path),
			Size:     info.Size(),
			Checksum: fmt.Sprintf("%x", checksum[:]),
			Content:  base64.StdEncoding.EncodeToString(content),
		})
	}
	return payload, nil
}

type catalogSyncRequest struct {
	Catalog *capabilities.CatalogSnapshot `json:"catalog"`
	Assets  []catalogSyncAsset            `json:"assets"`
}

type catalogSyncAsset struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Content  string `json:"content"`
}
