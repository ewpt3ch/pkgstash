package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed write test config: %v", err)
	}
	return path
}

func TestReadConfig(t *testing.T) {
	path := writeConfigFile(t, `
cache_root = "srv/cache"
mirror_urls = ["https://mirror.example.com"]
mirrored_repos = ["core", "extra"]
port = "8090"

[auth]
token = "testtoken"
`)

	cfg, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("expected no err on read got: %v", err)
	}
	if cfg.Port != "8090" {
		t.Errorf("expected port 8090 got %s", cfg.Port)
	}
}

func TestMissingCacheRoot(t *testing.T) {
	path := writeConfigFile(t, `
mirror_urls = ["https://mirror.example.com"]
mirrored_repos = ["core", "extra"]
port = "8090"

[auth]
token = "testtoken"
`)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingMirrorUrls(t *testing.T) {
	path := writeConfigFile(t, `
cache_root = "srv/cache"
mirrored_repos = ["core", "extra"]
port = "8090"

[auth]
token = "testtoken"
`)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingMirroredRepos(t *testing.T) {
	path := writeConfigFile(t, `
cache_root = "srv/cache"
mirror_urls = ["https://mirror.example.com"]
port = "8090"

[auth]
token = "testtoken"
`)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingPort(t *testing.T) {
	path := writeConfigFile(t, `
cache_root = "srv/cache"
mirror_urls = ["https://mirror.example.com"]
mirrored_repos = ["core", "extra"]

[auth]
token = "testtoken"
`)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}

func TestMissingAuthToken(t *testing.T) {
	path := writeConfigFile(t, `
cache_root = "srv/cache"
mirror_urls = ["https://mirror.example.com"]
mirrored_repos = ["core", "extra"]
port = "8090"

[auth]
`)

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
	path := writeConfigFile(t, `
cache_root = [srv/cache]
`)

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected err got nil")
	}
}
