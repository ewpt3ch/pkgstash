package cache

import (
	"fmt"
	"path/filepath"
)

func (c *Cache) nextMirror() string {
	idx := c.mirrorIdx.Add(1) - 1
	mirrorCount := uint64(len(c.cfg.mirrorURLs))
	return c.cfg.mirrorURLs[idx%mirrorCount]
}

func (c *Cache) getCachedFile(relPath string) (*CacheFile, error) {
	info, err := c.cr.Stat(relPath)
	if err != nil {
		return nil, err
	}

	f, err := c.cr.Open(relPath)
	if err != nil {
		return nil, err
	}

	return &CacheFile{
		Reader:   f,
		Size:     info.Size(),
		Filename: filepath.Base(relPath),
	}, nil
}

type UpstreamError struct {
	StatusCode int
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream returned %d", e.StatusCode)
}

func (c *Cache) Close() error {
	return c.cr.Close()
}
