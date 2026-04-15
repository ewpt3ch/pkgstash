package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	MirrorRoot string
	MirrorURL  string
	Port       string
	Auth       AuthConfig
}

type AuthConfig struct {
	Token string
}

func NewConfig() *Config {
	return &Config{
		MirrorRoot: "/home/ewpt3ch/dev/pacman-cache-server/tmprepo",
		MirrorURL:  "https://us.mirrors.cicku.me/archlinux/",
		Port:       "8090",
		Auth:       AuthConfig{Token: "FakeToken"},
	}
}

func ReadConfig(path string) (*Config, error) {

	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("Error loading config from %s: %w", path, err)
	}

	if err = cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.MirrorRoot == "" {
		return fmt.Errorf("cache root is required")
	}
	if c.MirrorURL == "" {
		return fmt.Errorf("mirror url is required")
	}
	if c.Port == "" {
		c.Port = "8090"
	}
	return nil
}
