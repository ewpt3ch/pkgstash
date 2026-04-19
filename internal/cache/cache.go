package cache

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type Cache struct {
	cacheRoot     string
	mirrorURL     string
	mirroredRepos []string
	sf            singleflight.Group //prevents duplicate downloads
	mu            sync.Mutex
	client        http.Client
}

func NewCache(cacheRoot string, mirrorURL string) *Cache {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	return &Cache{
		cacheRoot:     cacheRoot,
		mirrorURL:     mirrorURL,
		mirroredRepos: []string{"core", "extra", "multilib"},
		client: http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Cache) Fetch(pkgPath string) error {
	_, err, _ := c.sf.Do(pkgPath, func() (any, error) {
		return nil, c.fetch(pkgPath)
	})
	return err
}

type UpstreamError struct {
	StatusCode int
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream returned %d", e.StatusCode)
}

func (c *Cache) fetch(pkgName string) error {
	// pkgName is relative to the localRoot
	// ie pkgName includes /{repo}/os/{arch}/ and the actual name linux-x.x.x.pkg.tar.zst
	tempPkgName := pkgName + ".tmp"
	tempPkgPath := filepath.Join(c.cacheRoot, tempPkgName) //full tmp write path
	outPkg := filepath.Join(c.cacheRoot, pkgName)
	pkgURL := c.mirrorURL + pkgName

	resp, err := c.client.Get(pkgURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &UpstreamError{StatusCode: resp.StatusCode}
	}

	outFile, err := os.Create(tempPkgPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(tempPkgPath)
		return err
	}

	if err := os.Rename(tempPkgPath, outPkg); err != nil {
		os.Remove(tempPkgPath)
		return err
	}
	return nil
}
