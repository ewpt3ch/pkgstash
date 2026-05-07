package cache

import (
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
)

func (c *Cache) nextMirror() string {
	idx := c.mirrorIdx.Add(1) - 1
	mirrorCount := uint64(len(c.cfg.mirrorURLs))
	return c.cfg.mirrorURLs[idx%mirrorCount]
}

func (c *Cache) downloadToDisk(url, relPath string) error {
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
	defer resp.Body.Close()

	// make sure the dir structure exists
	err = c.cr.MkdirAll(filepath.Dir(relPath), 0750)
	if err != nil {
		return err
	}

	// use a tmp file for the initial fetch in case it fails
	tmpPath := relPath + ".tmp"
	tmpFile, err := c.cr.Create(tmpPath)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		removeErr := c.cr.Remove(tmpPath)
		if removeErr != nil {
			slog.Warn("failed to remove temp file", "path", tmpPath, "err", removeErr)
		}
		return err
	}

	// mv file to final location
	err = c.cr.Rename(tmpPath, relPath)
	if err != nil {
		removeErr := c.cr.Remove(tmpPath)
		if removeErr != nil {
			slog.Warn("failed to remove temp file", "path", tmpPath, "err", removeErr)
		}
		return err
	}
	return nil
}
