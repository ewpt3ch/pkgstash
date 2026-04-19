package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	CacheRoot string     `toml:"cache_root"`
	MirrorURL string     `toml:"mirror_url"`
	Port      string     `toml:"port"`
	Auth      AuthConfig `toml:"auth"`
}

type AuthConfig struct {
	Token string `toml:"token"`
}

/* Function kept for reference for future logic
func NewConfig() *Config {
	return &Config{
		CacheRoot: "/home/ewpt3ch/dev/pacman-cache-server/tmprepo",
		MirrorURL: "https://us.mirrors.cicku.me/archlinux/",
		Port:      "8090",
		Auth:      AuthConfig{Token: "FakeToken"},
	}
}
*/

func ReadConfig(path string) (*Config, error) {

	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error loading config from %s: %w", path, err)
	}

	if err = cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.CacheRoot == "" {
		return fmt.Errorf("cache root is required")
	}
	if c.MirrorURL == "" {
		return fmt.Errorf("mirror url is required")
	}
	if c.Port == "" {
		c.Port = "8090"
	}
	if c.Auth.Token == "" {
		return fmt.Errorf("auth token is required")
	}
	return nil
}
