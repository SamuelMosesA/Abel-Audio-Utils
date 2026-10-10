package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCredentialsDefaultAdmin(t *testing.T) {
	// When no credentials file is specified or file is missing, admin:admin is default
	creds := LoadCredentials("/path/to/config.yaml", "")
	assert.Equal(t, "admin", creds["admin"])

	credsMissing := LoadCredentials("/path/to/config.yaml", "nonexistent.json")
	assert.Equal(t, "admin", credsMissing["admin"])
}

func TestLoadCredentialsRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create relative credentials file in the same directory as configPath
	credsContent := `[{"username": "admin", "password": "secretpassword"}, {"username": "operator", "password": "op123"}]`
	credsFile := filepath.Join(tmpDir, "admin_user_credentials.json")
	err := os.WriteFile(credsFile, []byte(credsContent), 0644)
	require.NoError(t, err)

	// Pass relative path "./admin_user_credentials.json"
	creds := LoadCredentials(configPath, "./admin_user_credentials.json")
	assert.Equal(t, "secretpassword", creds["admin"])
	assert.Equal(t, "op123", creds["operator"])
}

func TestLoadCredentialsMapFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sub", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))

	credsContent := `{"user1": "pass1"}`
	credsFile := filepath.Join(filepath.Dir(configPath), "creds.json")
	require.NoError(t, os.WriteFile(credsFile, []byte(credsContent), 0644))

	creds := LoadCredentials(configPath, "creds.json")
	assert.Equal(t, "pass1", creds["user1"])
	// Default admin is still present
	assert.Equal(t, "admin", creds["admin"])
}

func TestLoadConfigEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `
port: "9090"
sample_rate: 48000
admin_user_credentials: "./my_creds.json"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, 48000, cfg.SampleRate)
	assert.Equal(t, "admin", cfg.Credentials["admin"])
}

func TestResolveSessionSecret(t *testing.T) {
	t.Run("Config Override", func(t *testing.T) {
		cfg := &Config{SessionSecret: "explicit-secret-key-12345"}
		secret, err := ResolveSessionSecret(cfg)
		require.NoError(t, err)
		assert.Equal(t, []byte("explicit-secret-key-12345"), secret)
	})

	t.Run("Dynamic Ephemeral Generation per run", func(t *testing.T) {
		cfg := &Config{}

		// First run generates random secret
		secret1, err := ResolveSessionSecret(cfg)
		require.NoError(t, err)
		assert.Len(t, secret1, 32)

		// Subsequent run generates distinct fresh secret to invalidate previous sessions
		secret2, err := ResolveSessionSecret(cfg)
		require.NoError(t, err)
		assert.Len(t, secret2, 32)
		assert.NotEqual(t, secret1, secret2)
	})

	t.Run("Nil config fallback", func(t *testing.T) {
		secret, err := ResolveSessionSecret(nil)
		require.NoError(t, err)
		assert.Len(t, secret, 32)
	})
}
