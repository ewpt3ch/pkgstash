package cache

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
)

func (c *Cache) downloadWrangle(relPath string, flight *inFlight, tmpFile *os.File) {
	// defer map cleanup and signal client handle done
	defer c.cleanupFlight(relPath, flight)

	// declare vars outside loop
	var err error
	// fetch pkgs from mirror with retry logic
	for range len(c.cfg.mirrorURLs) {
		url := c.nextMirror() + relPath
		err = c.downloadToDisk(url, tmpFile)
		if err == nil {
			break
		}
		var upstreamErr *UpstreamError
		if !errors.As(err, &upstreamErr) {
			// network error, transfer interupted, bail
			break
		}
		slog.Warn("mirror failed", "url", url, "status", upstreamErr.StatusCode)
	}
	if err != nil {
		slog.Warn("download error", "err", err)
		flight.err = err
		removeErr := c.cr.Remove(flight.tmpPath)
		if removeErr != nil {
			slog.Warn("failed to remove temp file", "path", flight.tmpPath, "err", removeErr)
		}
		return
	}

	// mv file to final location
	err = c.cr.Rename(flight.tmpPath, relPath)
	if err != nil {
		removeErr := c.cr.Remove(flight.tmpPath)
		if removeErr != nil {
			slog.Warn("failed to remove temp file", "path", flight.tmpPath, "err", removeErr)
		}
		return
	}
	return
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

func (c *Cache) cleanupFlight(key string, f *inFlight) {
	c.inFlightMu.Lock()
	delete(c.inFlight, key)
	c.inFlightMu.Unlock()
	close(f.done)
}
