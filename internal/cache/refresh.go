package cache

import (
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
)

func (c *Cache) FetchDB(repo string) error {
	dbFile := repo + ".db.tar.gz"
	dbPath := filepath.Join(repo, "os/x86_64", dbFile)
	url := c.nextMirror() + dbPath

	// set the user agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("failed create request", "err", err)
		return err
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
	//nolint:errcheck //best effort
	defer resp.Body.Close()

	tmpPath := dbPath + ".tmp"
	tmpFile, err := c.cr.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		if err := tmpFile.Close(); err != nil {
			slog.Error("failed to close file", "file", tmpPath, "err", err)
		}
		if err := c.cr.Remove(tmpPath); err != nil {
			slog.Error("failed to remove file", "file", tmpPath, "err", err)
		}
		return err
	}

	if err := tmpFile.Close(); err != nil {
		if err := c.cr.Remove(tmpPath); err != nil {
			slog.Error("failed to remove file", "file", tmpPath, "err", err)
		}
		return err
	}
	if err := c.cr.Rename(tmpPath, dbPath); err != nil {
		if err := c.cr.Remove(tmpPath); err != nil {
			slog.Error("failed to remove file", "file", tmpPath, "err", err)
		}
		slog.Error("failed to move tmpfile to permanent file", "file", tmpFile, "err", err)
		return err
	}

	return nil
}
