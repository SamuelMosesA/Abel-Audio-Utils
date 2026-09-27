package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCredentials(t *testing.T) {
	cfg := &Config{Credentials: map[string]string{"admin": "secret"}}

	assert.True(t, cfg.CheckCredentials("admin", "secret"))
	assert.False(t, cfg.CheckCredentials("admin", "wrong"))
	assert.False(t, cfg.CheckCredentials("nobody", ""))
}

func TestApplyReloadAppliesLiveSettingsAndReportsTheRest(t *testing.T) {
	cfg := &Config{
		Port: "8080", SampleRate: 48000, BufferSize: 1024,
		DefaultChL: 1, DefaultChR: 0, DefaultBoost: 1.0,
		StorageLocation: "/rec", OpenAIVoice: "alloy",
		AILanguages: []AILanguage{{Code: "en", Name: "English"}},
		Credentials: map[string]string{"admin": "old"},
	}
	fresh := &Config{
		Port: "9090", SampleRate: 44100, BufferSize: 512,
		DefaultChL: 1, DefaultChR: 2, DefaultBoost: 1.0,
		StorageLocation: "/rec", OpenAIVoice: "verse",
		AILanguages: []AILanguage{{Code: "en", Name: "English"}, {Code: "nl", Name: "Dutch"}},
		Credentials: map[string]string{"admin": "new"},
	}

	changes := cfg.ApplyReload(fresh)

	assert.False(t, changes.DefaultChL)
	assert.True(t, changes.DefaultChR)
	assert.False(t, changes.DefaultBoost)
	assert.Equal(t, []string{"port", "openai_voice", "ai_languages"}, changes.NeedsFullRestart)

	// Applied live
	assert.Equal(t, 44100, cfg.SampleRate)
	assert.Equal(t, 512, cfg.BufferSize)
	assert.Equal(t, 2, cfg.DefaultChR)
	assert.True(t, cfg.CheckCredentials("admin", "new"))
	// Left for a full restart
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "alloy", cfg.OpenAIVoice)
	assert.Len(t, cfg.AILanguages, 1)
}

func TestApplyReloadNoChanges(t *testing.T) {
	cfg := &Config{Port: "8080", AILanguages: []AILanguage{{Code: "en", Name: "English"}}}
	fresh := &Config{Port: "8080", AILanguages: []AILanguage{{Code: "en", Name: "English"}}}

	assert.Empty(t, cfg.ApplyReload(fresh).NeedsFullRestart)
}

func TestLoadConfigReadsCredentialsFile(t *testing.T) {
	dir := t.TempDir()
	creds := filepath.Join(dir, "creds.json")
	require.NoError(t, os.WriteFile(creds, []byte(`[{"username":"admin","password":"pw"}]`), 0o600))
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("port: \"8080\"\nadmin_user_credentials: \""+filepath.ToSlash(creds)+"\"\n"), 0o600))

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.True(t, cfg.CheckCredentials("admin", "pw"))
}
