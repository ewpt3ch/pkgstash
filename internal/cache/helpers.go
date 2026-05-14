package cache

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func (c *Cache) nextMirror() string {
	idx := c.mirrorIdx.Add(1) - 1
	mirrorCount := uint64(len(c.cfg.mirrorURLs))
	return c.cfg.mirrorURLs[idx%mirrorCount]
}

func (c *Cache) downloadToDisk(url string, tmpFile *os.File) error {
	slog.Info("fetching", "url", url)

	// set the user agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("failed create request", "err", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		slog.Warn("fetch failed", "url", url, "err", err)
		return err
	}
	if resp.StatusCode != 200 {
		slog.Info("fetch returned", "url", url, "status", resp.StatusCode)
		return &UpstreamError{StatusCode: resp.StatusCode}
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(tmpFile, resp.Body)
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
