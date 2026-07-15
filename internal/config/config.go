package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port     int    `yaml:"port"`
	MusicDir string `yaml:"music_dir"`
}

type StationConfig struct {
	Name    string   `yaml:"name"`
	Mount   string   `yaml:"mount"`
	Sources []string `yaml:"sources"`
}

type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Stations []StationConfig `yaml:"stations"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
