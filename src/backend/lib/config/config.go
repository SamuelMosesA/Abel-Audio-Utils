package config

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)


type AILanguage struct {
	Code string `yaml:"code" json:"code"`
	Name string `yaml:"name" json:"name"`
}

type Config struct {
	Port               string  `yaml:"port"`
	SampleRate         int     `yaml:"sample_rate"`
	BufferSize         int     `yaml:"buffer_size"`
	StorageLocation    string  `yaml:"storage_location"`
	CloudDriveLocation string  `yaml:"cloud_drive_location"`
	DefaultChL         int     `yaml:"default_ch_l"`
	DefaultChR         int     `yaml:"default_ch_r"`
	DefaultBoost       float64 `yaml:"default_boost"`
	AdminUserCredentials  string  `yaml:"admin_user_credentials"`
	OpenAIAPIKey          string       `yaml:"openai_api_key"`
	OpenAITranslateModel  string       `yaml:"openai_translate_model"`
	OpenAITranscribeModel string       `yaml:"openai_transcribe_model"`
	OpenAIVoice           string       `yaml:"openai_voice"`
	AILanguages           []AILanguage `yaml:"ai_languages"`
	AIOriginalLanguage    string       `yaml:"ai_original_language"`
	OTLPEndpoint          string       `yaml:"otlp_endpoint"`

	// Loaded from credentials file
	Credentials map[string]string `yaml:"-"`

	// Guards fields that ApplyReload changes while the server is running.
	mu sync.RWMutex
}

// CheckCredentials reports whether username/password match the loaded credentials file.
func (cfg *Config) CheckCredentials(username, password string) bool {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	p, ok := cfg.Credentials[username]
	return ok && p == password
}

// ReloadChanges describes what ApplyReload did with a freshly loaded config.
type ReloadChanges struct {
	DefaultChL   bool
	DefaultChR   bool
	DefaultBoost bool
	// Settings that changed on disk but only take effect after a full restart of Abel.
	NeedsFullRestart []string
}

// ApplyReload copies the settings that are safe to change while Abel is running
// (credentials, default routing/gain, sample rate, buffer size) from fresh into cfg,
// and lists changed settings that still need a full restart.
// The audio engine must be stopped first: it reads SampleRate and BufferSize without locking.
func (cfg *Config) ApplyReload(fresh *Config) ReloadChanges {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	changes := ReloadChanges{
		DefaultChL:   cfg.DefaultChL != fresh.DefaultChL,
		DefaultChR:   cfg.DefaultChR != fresh.DefaultChR,
		DefaultBoost: cfg.DefaultBoost != fresh.DefaultBoost,
	}

	cfg.DefaultChL = fresh.DefaultChL
	cfg.DefaultChR = fresh.DefaultChR
	cfg.DefaultBoost = fresh.DefaultBoost
	cfg.SampleRate = fresh.SampleRate
	cfg.BufferSize = fresh.BufferSize
	cfg.AdminUserCredentials = fresh.AdminUserCredentials
	cfg.Credentials = fresh.Credentials

	fullRestart := []struct {
		name    string
		changed bool
	}{
		{"port", cfg.Port != fresh.Port},
		{"storage_location", cfg.StorageLocation != fresh.StorageLocation},
		{"cloud_drive_location", cfg.CloudDriveLocation != fresh.CloudDriveLocation},
		{"openai_api_key", cfg.OpenAIAPIKey != fresh.OpenAIAPIKey},
		{"openai_translate_model", cfg.OpenAITranslateModel != fresh.OpenAITranslateModel},
		{"openai_transcribe_model", cfg.OpenAITranscribeModel != fresh.OpenAITranscribeModel},
		{"openai_voice", cfg.OpenAIVoice != fresh.OpenAIVoice},
		{"ai_languages", !slices.Equal(cfg.AILanguages, fresh.AILanguages)},
		{"ai_original_language", cfg.AIOriginalLanguage != fresh.AIOriginalLanguage},
		{"otlp_endpoint", cfg.OTLPEndpoint != fresh.OTLPEndpoint},
	}
	for _, f := range fullRestart {
		if f.changed {
			changes.NeedsFullRestart = append(changes.NeedsFullRestart, f.name)
		}
	}

	return changes
}

func (cfg *Config) ResolveLanguageName(code string) string {
	lowerCode := strings.ToLower(code)
	for _, l := range cfg.AILanguages {
		if strings.ToLower(l.Code) == lowerCode {
			return l.Name
		}
	}
	// Fallback to capitalizing the code if not found
	if len(code) > 0 {
		return strings.ToUpper(code[:1]) + code[1:]
	}
	return code
}

func (cfg *Config) ResolveLanguageCode(name string) string {
	lowerName := strings.ToLower(name)
	for _, l := range cfg.AILanguages {
		if strings.ToLower(l.Name) == lowerName {
			return l.Code
		}
	}
	return name
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	// Load credentials if configured
	cfg.Credentials = make(map[string]string)
	if cfg.AdminUserCredentials != "" {
		credsData, err := os.ReadFile(cfg.AdminUserCredentials)
		if err == nil {
			var credList []struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.Unmarshal(credsData, &credList); err == nil {
				for _, c := range credList {
					cfg.Credentials[c.Username] = c.Password
				}
			} else {
				// Try map[string]string format as fallback
				json.Unmarshal(credsData, &cfg.Credentials)
			}
		}
	}

	if cfg.OpenAIAPIKey == "" {
		cfg.OpenAIAPIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	cfg.OpenAIAPIKey = strings.TrimSpace(cfg.OpenAIAPIKey)
	if cfg.OpenAITranslateModel == "" {
		cfg.OpenAITranslateModel = "gpt-4o-realtime-preview"
	}
	if cfg.OpenAITranscribeModel == "" {
		cfg.OpenAITranscribeModel = "gpt-4o-realtime-preview"
	}
	if cfg.OpenAIVoice == "" {
		cfg.OpenAIVoice = "alloy"
	}

	if cfg.AIOriginalLanguage == "" {
		cfg.AIOriginalLanguage = "en"
	}
	if len(cfg.AILanguages) == 0 {
		cfg.AILanguages = []AILanguage{
			{Code: "en", Name: "English"},
			{Code: "nl", Name: "Dutch"},
			{Code: "pt", Name: "Portuguese"},
			{Code: "es", Name: "Spanish"},
			{Code: "fr", Name: "French"},
			{Code: "de", Name: "German"},
			{Code: "ru", Name: "Russian"},
			{Code: "tr", Name: "Turkish"},
			{Code: "pl", Name: "Polish"},
			{Code: "id", Name: "Indonesian"},
		}
	}

	return &cfg, nil
}
