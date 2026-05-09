package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfigFiles(t *testing.T, cfgMap map[string]string, envPerm os.FileMode) string {
	t.Helper()
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "config.toml")
	envPath := filepath.Join(dir, "pkgstash.env")

	// set env_file path
	cfgMap["env_file"] = fmt.Sprintf("%q", envPath)

	// build the content strings
	var tb strings.Builder
	var eb strings.Builder
	for k, v := range cfgMap {
		if k == "PKGSTASH_TOKEN" {
			fmt.Fprintf(&eb, "%s=%s\n", k, v)
		} else {
			fmt.Fprintf(&tb, "%s = %s\n", k, v)
		}
	}

	//write config.toml
	if err := os.WriteFile(tomlPath, []byte(tb.String()), 0644); err != nil {
		t.Fatalf("failed write test config: %v", err)
	}

	//write env
	if err := os.WriteFile(envPath, []byte(eb.String()), envPerm); err != nil {
		t.Fatalf("failed write test env: %v", err)
	}
	return tomlPath
}

func defaultCfgMap() map[string]string {
	return map[string]string{
		"env_file":       `""`,
		"cache_root":     `"srv/cache"`,
		"mirror_urls":    `["https://mirror.example.com"]`,
		"mirrored_repos": `["core", "extra"]`,
		"port":           `"8090"`,
		"PKGSTASH_TOKEN": "testtoken",
	}
}

func TestReadConfig(t *testing.T) {
	cfgMap := defaultCfgMap()
	path := writeConfigFiles(t, cfgMap, 0640)

	cfg, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("expected no err on read got: %v", err)
	}
	if cfg.Port != "8090" {
		t.Errorf("expected port 8090 got %s", cfg.Port)
	}
}

func TestMissingCacheRoot(t *testing.T) {
	cfgMap := defaultCfgMap()
	cfgMap["cache_root"] = ""
	path := writeConfigFiles(t, cfgMap, 0600)
	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingMirrorUrls(t *testing.T) {
	cfgMap := defaultCfgMap()
	delete(cfgMap, "mirror_urls")
	path := writeConfigFiles(t, cfgMap, 0600)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingMirroredRepos(t *testing.T) {
	cfgMap := defaultCfgMap()
	delete(cfgMap, "mirrored_repos")
	path := writeConfigFiles(t, cfgMap, 0600)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingPort(t *testing.T) {
	cfgMap := defaultCfgMap()
	cfgMap["port"] = ""
	path := writeConfigFiles(t, cfgMap, 0600)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingAuthToken(t *testing.T) {
	cfgMap := defaultCfgMap()
	cfgMap["PKGSTASH_TOKEN"] = ""
	path := writeConfigFiles(t, cfgMap, 0600)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistant.toml")

	_, err := ReadConfig(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expected err got nil")
	}
}

func TestInvalidToml(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("cache_root = [srv/cache]"), 0644); err != nil {
		t.Fatalf("failed write test config: %v", err)
	}

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}

}

func TestWrongPermissionsEnv(t *testing.T) {
	cfgMap := defaultCfgMap()
	path := writeConfigFiles(t, cfgMap, 0644)
	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestEmptyEnv(t *testing.T) {
	cfgMap := defaultCfgMap()
	delete(cfgMap, "PKGSTASH_TOKEN")
	path := writeConfigFiles(t, cfgMap, 0600)
	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestEmptyKeyEnv(t *testing.T) {
	cfgMap := defaultCfgMap()
	cfgMap["PKGSTASH_TOKEN"] = ""
	path := writeConfigFiles(t, cfgMap, 0600)
	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestDefaultToken(t *testing.T) {
	cfgMap := defaultCfgMap()
	cfgMap["PKGSTASH_TOKEN"] = "changeme"
	path := writeConfigFiles(t, cfgMap, 0600)
	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}
