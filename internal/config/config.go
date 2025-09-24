package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	App struct {
		UserInactiveTimeout time.Duration `yaml:"user_inactive_timeout_minutes"`
		UsernameCooldown    time.Duration `yaml:"username_cooldown_minutes"`
	} `yaml:"app"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return nil, err
	}

	// Convert minutes to time.Duration
	cfg.App.UserInactiveTimeout *= time.Minute
	cfg.App.UsernameCooldown *= time.Minute

	return &cfg, nil
}

