package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type rawConfig struct {
	Config
	EnvFile        string `toml:"env_file"`
	MaxCacheSize   string `toml:"max_cache_size"`
	MaxCacheAgeStr string `toml:"max_cache_age"`
}

type Config struct {
	CacheRoot     string   `toml:"cache_root"`
	MirrorURLs    []string `toml:"mirror_urls"`
	MirroredRepos []string `toml:"mirrored_repos"`
	Port          string   `toml:"port"`
	Token         string
	MaxCacheSize  int64
	MaxCacheAge   time.Duration
}

func ReadConfig(path string) (*Config, error) {

	var rawcfg rawConfig
	_, err := toml.DecodeFile(path, &rawcfg)
	if err != nil {
		return nil, fmt.Errorf("error loading config from %s: %w", path, err)
	}

	cfg := rawcfg.Config

	err = cfg.loadToken(rawcfg.EnvFile)
	if err != nil {
		return nil, fmt.Errorf("error getting token: %v", err)
	}

	if rawcfg.MaxCacheSize == "" {
		return nil, fmt.Errorf("max_cache_size must be set")
	}
	sizeBytes, err := strconv.ParseInt(strings.TrimSuffix(rawcfg.MaxCacheSize, "GB"), 10, 64)
	sizeBytes = sizeBytes * 1024 * 1024 * 1024
	if err != nil {
		return nil, fmt.Errorf("invalid cache size string")
	}
	cfg.MaxCacheSize = sizeBytes

	cfg.MaxCacheAge, err = time.ParseDuration(rawcfg.MaxCacheAgeStr)
	if err != nil {
		return nil, fmt.Errorf("error getting max_cache_age: %v", err)
	}

	if err = cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) loadToken(path string) error {

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat env file: %v", err)
	}

	if info.Mode().Perm() != 0o640 {
		return fmt.Errorf("env file perms not secure, expected 0640 got %o", info.Mode().Perm())
	}
	//#nosec G304 -- config is known path
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read env file: %v", err)
	}

	key, val, ok := strings.Cut(strings.TrimSpace(string(data)), "=")
	if !ok || key != "PKGSTASH_TOKEN" {
		return fmt.Errorf("invalid env file format")
	}

	c.Token = val
	return nil

}

func (c *Config) validate() error {
	if c.CacheRoot == "" {
		return fmt.Errorf("cache root is required")
	}
	if len(c.MirrorURLs) == 0 {
		return fmt.Errorf("at least one mirror is required")
	}
	if len(c.MirroredRepos) == 0 {
		return fmt.Errorf("at least one repo is required")
	}
	if c.Port == "" {
		return fmt.Errorf("port required")
	}
	if c.Token == "" || c.Token == "changeme" {
		return fmt.Errorf("auth token is required")
	}
	return nil
}
