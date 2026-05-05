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
	return c.cfg.mirrorURLs[idx%uint32(len(c.cfg.mirrorURLs))]
}

func downloadToDisk(url, destPath string, c http.Client) error {
	slog.Info("fetching", "url", url)

	// set the user agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("failed create request", "err", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.Do(req)
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
	os.MkdirAll(filepath.Dir(destPath), 0755)

	// use a tmp file for the initial fetch in case it fails
	tempPath := destPath + ".tmp"
	tmpFile, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tempPath)
		return err
	}

	// mv file to final location
	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return err
	}
	return nil
}
