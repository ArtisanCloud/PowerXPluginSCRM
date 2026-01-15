package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/plugin"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/crypto"
	"github.com/google/uuid"
)

// CredentialService 负责插件凭证的加密与持久化
type CredentialService struct {
	cfg  *config.Config
	repo *repo.CredentialsRepository
}

func NewCredentialService(cfg *config.Config, repo *repo.CredentialsRepository) *CredentialService {
	return &CredentialService{cfg: cfg, repo: repo}
}

// SavePlainCredentials 接收明文 client_secret，进行加密并保存
func (s *CredentialService) SavePlainCredentials(ctx context.Context, tenantUUID string, pluginID, clientID, clientSecret string) error {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return errors.New("invalid tenant_uuid")
	}
	if _, err := uuid.Parse(tenantUUID); err != nil {
		return errors.New("invalid tenant_uuid")
	}
	if pluginID == "" || clientID == "" || clientSecret == "" {
		return errors.New("missing required fields")
	}

	keyMaterial := s.cfg.Server.SecretKey
	if keyMaterial == "" {
		if s.cfg.IsProduction() {
			return errors.New("server.secret_key not configured")
		}
		logger.Warn("server.secret_key is empty; using DEV-ONLY fallback key. Do NOT use in production.")
		keyMaterial = "dev-only-change-me"
	}
	key := crypto.DeriveKey32(keyMaterial)

	// AAD 绑定：租户与插件维度，避免密文移植
	aad := []byte(fmt.Sprintf("tenant:%s|plugin:%s|cid:%s", tenantUUID, pluginID, clientID))

	ct, iv, err := crypto.EncryptAESGCM(key, []byte(clientSecret), aad)
	if err != nil {
		return err
	}

	pc := &models.PluginCredential{
		TenantUUID:       tenantUUID,
		PluginID:         pluginID,
		ClientID:         clientID,
		SecretCiphertext: ct,
		IVNonce:          iv,
		KeyVersion:       1,
		UpdatedAt:        time.Now(),
	}
	return s.repo.Upsert(ctx, pc)
}

// LoadDecryptedCredentials 读取并解密，返回 (clientID, clientSecret)
func (s *CredentialService) LoadDecryptedCredentials(ctx context.Context, tenantUUID string, pluginID string) (string, string, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return "", "", errors.New("invalid tenant_uuid")
	}
	if _, err := uuid.Parse(tenantUUID); err != nil {
		return "", "", errors.New("invalid tenant_uuid")
	}
	pc, err := s.repo.GetByTenantPlugin(ctx, tenantUUID, pluginID)
	if err != nil {
		return "", "", err
	}

	keyMaterial := s.cfg.Server.SecretKey
	if keyMaterial == "" {
		if s.cfg.IsProduction() {
			return "", "", errors.New("server.secret_key not configured")
		}
		keyMaterial = "dev-only-change-me"
	}
	key := crypto.DeriveKey32(keyMaterial)
	aad := []byte(fmt.Sprintf("tenant:%s|plugin:%s|cid:%s", pc.TenantUUID, pc.PluginID, pc.ClientID))

	pt, err := crypto.DecryptAESGCM(key, pc.SecretCiphertext, pc.IVNonce, aad)
	if err != nil {
		return "", "", err
	}
	return pc.ClientID, string(pt), nil
}
