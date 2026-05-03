package cache

import (
	"io"
	"log"
	"net/http"
	"os"
)

func (c *Cache) nextMirror() string {
	idx := c.mirrorIdx.Add(1) - 1
	return c.cfg.mirrorURLs[idx%uint32(len(c.cfg.mirrorURLs))]
}

func downloadToDisk(url, destPath string, c http.Client) error {
	// #log
	log.Printf("fetching %v", url)

	// set the user agent
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// #log
		log.Printf("failed to create request: %v", err)
		return &UpstreamError{StatusCode: http.StatusInternalServerError}
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.Do(req)
	if err != nil {
		// #log
		log.Printf("error fetching %s: %v", url, err)
		return err
	}
	if resp.StatusCode != 200 {
		// #log
		log.Printf("GET %s returned %d", url, resp.StatusCode)
		return &UpstreamError{StatusCode: resp.StatusCode}
	}
	defer resp.Body.Close()

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
