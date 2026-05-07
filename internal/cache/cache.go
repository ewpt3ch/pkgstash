package cache

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const userAgent = "pacman/7.1.0 (Linux x86_64) libalpm/16.0.1"

type Cache struct {
	cfg       CacheConfig
	cr        *os.Root
	mirrorIdx atomic.Uint64
	sf        singleflight.Group //prevents duplicate downloads
	mu        sync.Mutex
	client    http.Client
}

type CacheConfig struct {
	mirrorURLs            []string
	mirroredRepos         []string
	DialTimeout           time.Duration
	ResponseHeaderTimeout time.Duration
	ClientTimeout         time.Duration
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

	return &Cache{
		cfg: cfg,
		cr:  cr,
		client: http.Client{
			Timeout:   cfg.ClientTimeout,
			Transport: transport,
		},
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
