package bootstrap

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
)

const defaultLocalIAMSecretFile = "tmp/local-iam-secret.key"

// EnsureLocalIAMSecret makes sure local provider mode reuses a stable HMAC secret across restarts.
// When context.hmac_secret is empty, the helper loads it from PLUGIN_LOCAL_IAM_SECRET_FILE or
// generates a random 32-byte hex string and persists it to disk.
func EnsureLocalIAMSecret(cfg *config.Config) error {
	if cfg == nil || cfg.Context == nil {
		return nil
	}
	current := strings.TrimSpace(cfg.Context.HMACSecret)
	if current != "" {
		cfg.Context.HMACSecret = current
		return nil
	}

	secretFile := strings.TrimSpace(os.Getenv("PLUGIN_LOCAL_IAM_SECRET_FILE"))
	if secretFile == "" {
		secretFile = defaultLocalIAMSecretFile
	}
	secret, err := os.ReadFile(secretFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.WithError(err).Warn("failed to read local IAM secret file")
		}
	}
	trimmed := strings.TrimSpace(string(secret))
	if trimmed == "" {
		trimmed, err = generateLocalIAMSecret()
		if err != nil {
			return err
		}
		if writeErr := writeLocalIAMSecret(secretFile, trimmed); writeErr != nil {
			return writeErr
		}
		logger.WithField("path", secretFile).Info("generated local IAM HMAC secret")
	}
	cfg.Context.HMACSecret = trimmed
	return nil
}

func generateLocalIAMSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func writeLocalIAMSecret(path string, secret string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("secret path empty")
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(secret+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
