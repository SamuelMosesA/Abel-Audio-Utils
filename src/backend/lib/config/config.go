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
	OpenAIAPIKey          string       `yaml:"openai_api_key"`
	OpenAITranslateModel  string       `yaml:"openai_translate_model"`
	OpenAITranscribeModel string       `yaml:"openai_transcribe_model"`
	OpenAIVoice           string       `yaml:"openai_voice"`
	AILanguages           []AILanguage `yaml:"ai_languages"`
	AIOriginalLanguage    string       `yaml:"ai_original_language"`
	OTLPEndpoint          string       `yaml:"otlp_endpoint"`

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
