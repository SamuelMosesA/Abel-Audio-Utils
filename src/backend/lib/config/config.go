package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type AILanguage struct {
	Code string `yaml:"code" json:"code"`
	Name string `yaml:"name" json:"name"`
}

// Config holds global server, audio interface, and AI streaming runtime configuration.
type Config struct {
	Port                  string       `yaml:"port"`
	SampleRate            int          `yaml:"sample_rate"`
	BufferSize            int          `yaml:"buffer_size"`
	StorageLocation       string       `yaml:"storage_location"`
	CloudDriveLocation    string       `yaml:"cloud_drive_location"`
	DefaultChL            int          `yaml:"default_ch_l"`
	DefaultChR            int          `yaml:"default_ch_r"`
	DefaultBoost          float64      `yaml:"default_boost"`
	AdminUserCredentials  string       `yaml:"admin_user_credentials"`
	SessionSecret         string       `yaml:"session_secret"`
	OpenAIAPIKey          string       `yaml:"openai_api_key"`
	OpenAITranslateModel  string       `yaml:"openai_translate_model"`
	OpenAITranscribeModel string       `yaml:"openai_transcribe_model"`
	OpenAIVoice           string       `yaml:"openai_voice"`
	AILanguages           []AILanguage `yaml:"ai_languages"`
	AIOriginalLanguage    string       `yaml:"ai_original_language"`
	OTLPEndpoint          string       `yaml:"otlp_endpoint"`

	// Loaded from credentials file
	Credentials map[string]string `yaml:"-"`

	// File the config was loaded from, used to reload it on engine restart
	Path string `yaml:"-"`
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

// ResolveRelativePath resolves target relative to the directory containing baseFile if target is relative.
// If target exists relative to baseFile's directory, that path is returned.
// Otherwise, it checks if target exists relative to working directory, and finally returns candidate or target.
func ResolveRelativePath(baseFile, target string) string {
	if target == "" || filepath.IsAbs(target) {
		return target
	}
	baseDir := filepath.Dir(baseFile)
	candidate := filepath.Join(baseDir, target)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	if _, err := os.Stat(target); err == nil {
		return target
	}
	return candidate
}

// LoadCredentials extracts and parses credentials from a JSON file, resolving relative filepaths
// against configPath. It supports list format [{"username": "...", "password": "..."}] and map format.
// Always ensures username "admin" with password "admin" is present as default if not explicitly defined.
func LoadCredentials(configPath, credsPath string) map[string]string {
	creds := make(map[string]string)

	if credsPath != "" {
		resolved := ResolveRelativePath(configPath, credsPath)
		if credsData, err := os.ReadFile(resolved); err == nil {
			var credList []struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.Unmarshal(credsData, &credList); err == nil && len(credList) > 0 {
				for _, c := range credList {
					if c.Username != "" {
						creds[c.Username] = c.Password
					}
				}
			} else {
				// Try map[string]string format as fallback
				var credMap map[string]string
				if err := json.Unmarshal(credsData, &credMap); err == nil {
					for k, v := range credMap {
						creds[k] = v
					}
				}
			}
		}
	}

	// Always ensure default admin:admin exists if not already set
	if _, exists := creds["admin"]; !exists {
		creds["admin"] = "admin"
	}

	return creds
}

// ResolveSessionSecret determines the cryptographic secret key used for session cookie signing.
// If cfg.SessionSecret is configured, it is converted to bytes and returned.
// Otherwise, a cryptographically secure 32-byte random key is generated dynamically in memory
// for every run, guaranteeing that restarting the server invalidates prior sessions.
func ResolveSessionSecret(cfg *Config) ([]byte, error) {
	if cfg != nil && cfg.SessionSecret != "" {
		return []byte(cfg.SessionSecret), nil
	}

	// Generate fresh 32 cryptographically secure random bytes per run
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate random session secret: %w", err)
	}
	return secret, nil
}

// LoadConfig parses application settings from YAML or environment variables.
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
	cfg.Path = path

	// Load credentials using dedicated function with relative path resolution
	cfg.Credentials = LoadCredentials(path, cfg.AdminUserCredentials)

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
