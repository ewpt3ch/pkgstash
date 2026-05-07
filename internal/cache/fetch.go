package cache

import (
	"errors"
	"log/slog"
	"path/filepath"
)

func (c *Cache) Fetch(relPath string) (*CacheFile, error) {
	// return file directly if exists in cache
	cf, err := c.getCachedFile(relPath)
	if err == nil {
		return cf, nil
	}

	// fetch file from upstream
	_, err, _ = c.sf.Do(relPath, func() (any, error) {
		slog.Debug("calling fetch", "file", relPath)
		return nil, c.fetch(relPath)
	})
	if err != nil {
		return nil, err
	}

	cf, err = c.getCachedFile(relPath)
	if err != nil {
		return nil, err
	}
	return cf, nil
}

func (c *Cache) fetch(relPath string) error {
	// relPath is relative to the localRoot
	// ie relPath includes /{repo}/os/{arch}/ and the actual name linux-x.x.x.pkg.tar.zst

	// declare vars outside loop
	var err error
	// fetch pkgs from mirror with retry logic
	for range len(c.cfg.mirrorURLs) {
		url := c.nextMirror() + relPath
		err = c.downloadToDisk(url, relPath)
		if err == nil {
			break
		}
		if upstreamErr, ok := errors.AsType[*UpstreamError](err); ok {
			slog.Warn("mirror failed", "url", url, "status", upstreamErr.StatusCode)
		} else {
			slog.Warn("mirror unreachable", "url", url, "err", err)
		}
	}
	if err != nil {
		return err
	}
	return nil
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
