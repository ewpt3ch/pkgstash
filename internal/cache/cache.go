package cache

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const userAgent = "pacman/7.1.0 (Linux x86_64) libalpm/16.0.1"

type Cache struct {
	cfg       CacheConfig
	mirrorIdx atomic.Uint32
	sf        singleflight.Group //prevents duplicate downloads
	mu        sync.Mutex
	client    http.Client
}

type CacheConfig struct {
	cacheRoot             string
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

func NewCache(cacheRoot string, mirrorURLs []string, mirroredRepos []string) *Cache {
	cfg := CacheConfig{
		cacheRoot:             cacheRoot,
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

	return &Cache{
		cfg: cfg,
		client: http.Client{
			Timeout:   cfg.ClientTimeout,
			Transport: transport,
		},
	}
}

type UpstreamError struct {
	StatusCode int
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream returned %d", e.StatusCode)
}
