package cache

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const userAgent = "pacman/7.1.0 (Linux x86_64) libalpm/16.0.1"

type Cache struct {
	cfg        CacheConfig
	cr         *os.Root
	mirrorIdx  atomic.Uint64
	refreshMu  sync.Mutex
	client     http.Client
	inFlight   map[string]*inFlight
	inFlightMu sync.Mutex
}

type CacheConfig struct {
	mirrorURLs            []string
	mirroredRepos         []string
	DialTimeout           time.Duration
	ResponseHeaderTimeout time.Duration
	ClientTimeout         time.Duration
}

type inFlight struct {
	tmpPath string
	done    chan struct{}
	err     error
}

type CacheFile struct {
	Reader   io.ReadCloser
	Size     int64
	Filename string
}

func NewCache(cacheRoot string, mirrorURLs []string, mirroredRepos []string) (*Cache, error) {
	cfg := CacheConfig{
		mirrorURLs:            mirrorURLs,
		mirroredRepos:         mirroredRepos,
		DialTimeout:           5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ClientTimeout:         0 * time.Second,
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: cfg.DialTimeout,
		}).DialContext,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
	}

	cr, err := os.OpenRoot(cacheRoot)
	if err != nil {
		return nil, err
	}

	if err := checkSymLinks(cr, mirroredRepos); err != nil {
		return nil, err
	}

	return &Cache{
		cfg: cfg,
		cr:  cr,
		client: http.Client{
			Timeout:   cfg.ClientTimeout,
			Transport: transport,
		},
		inFlight: make(map[string]*inFlight),
	}, nil
}

func (c *Cache) Close() error {
	return c.cr.Close()
}

type UpstreamError struct {
	StatusCode int
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream returned %d", e.StatusCode)
}

func checkSymLinks(cr *os.Root, repos []string) error {
	for _, repo := range repos {
		dirPath := filepath.Join(repo, "os/x86_64")
		if err := cr.MkdirAll(dirPath, 0750); err != nil {
			return err
		}
		lnPath := filepath.Join(dirPath, repo+".db")
		srcPath := repo + ".db.tar.gz"
		if _, err := cr.Lstat(lnPath); err == nil {
			continue // link exists
		}
		if err := cr.Symlink(srcPath, lnPath); err != nil {
			return err
		}
	}
	return nil
}
