package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port               string  `yaml:"port"`
	SampleRate         int     `yaml:"sample_rate"`
	BufferSize         int     `yaml:"buffer_size"`
	StorageLocation    string  `yaml:"storage_location"`
	CloudDriveLocation string  `yaml:"cloud_drive_location"`
	DefaultChL         int     `yaml:"default_ch_l"`
	DefaultChR         int     `yaml:"default_ch_r"`
	DefaultBoost       float64 `yaml:"default_boost"`
	AdminPassword      string  `yaml:"admin_password"`
	GeminiAPIKey          string  `yaml:"gemini_api_key"`
	GeminiModel           string  `yaml:"gemini_model"`
	GeminiChunkMultiplier int     `yaml:"gemini_chunk_multiplier"`
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

	if cfg.GeminiAPIKey == "" {
		cfg.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = "models/gemini-2.0-flash-exp" // Default model
	}
	if cfg.GeminiChunkMultiplier <= 0 {
		cfg.GeminiChunkMultiplier = 1
	}

	return &cfg, nil
}
