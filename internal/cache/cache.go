package cache

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sync/singleflight"
)

type Cache struct {
	localRoot     string
	mirrorURL     string
	mirroredRepos []string
	sf            singleflight.Group //prevents duplicate downloads
	mu            sync.Mutex
}

func NewCache(localRoot string, mirrorURL string) *Cache {
	return &Cache{
		localRoot:     localRoot,
		mirrorURL:     mirrorURL,
		mirroredRepos: []string{"core", "extra"},
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
	tempPkgName := pkgName + ".tmp"
	tempPkgPath := filepath.Join(c.localRoot, tempPkgName) //full tmp write path
	outPkg := filepath.Join(c.localRoot, pkgName)
	pkgURL := c.mirrorURL + pkgName

	resp, err := http.Get(pkgURL)
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
