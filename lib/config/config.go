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

	return &cfg, nil
}
