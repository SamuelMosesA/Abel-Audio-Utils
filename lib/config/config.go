package config

import (
	"encoding/json"
	"os"
	"strings"

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
	GeminiAPIKey          string  `yaml:"gemini_api_key"`
	GeminiModel           string  `yaml:"gemini_model"`
	GeminiChunkMultiplier int     `yaml:"gemini_chunk_multiplier"`
	GeminiVoice           string  `yaml:"gemini_voice"`
	AILanguages           []AILanguage `yaml:"ai_languages"`
	AIOriginalLanguage    string       `yaml:"ai_original_language"`

	// Loaded from credentials file
	Credentials map[string]string `yaml:"-"`
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

	if cfg.GeminiAPIKey == "" {
		cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = "models/gemini-2.5-flash-native-audio-preview-12-2025"
	}
	if cfg.GeminiVoice == "" {
		cfg.GeminiVoice = "Zephyr"
	}
	if cfg.GeminiChunkMultiplier <= 0 {
		cfg.GeminiChunkMultiplier = 1
	}

	if cfg.AIOriginalLanguage == "" {
		cfg.AIOriginalLanguage = "English"
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
