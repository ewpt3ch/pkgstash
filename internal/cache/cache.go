package cache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/sync/singleflight"
)

type Cache struct {
	repo      string
	mirrorURL string
	sf        singleflight.Group
}

func NewCache(repo string, mirrorURL string) *Cache {
	return &Cache{
		repo:      repo,
		mirrorURL: mirrorURL,
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

func (c *Cache) fetch(pkgPath string) error {
	// pkgPath is relative to the repo or mirror root
	tempPkgPath := pkgPath + ".tmp"
	tempPkgName := filepath.Join(c.repo, tempPkgPath) //full tmp write path
	outPkg := filepath.Join(c.repo, pkgPath)
	pkgURL := c.mirrorURL + pkgPath

	resp, err := http.Get(pkgURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &UpstreamError{StatusCode: resp.StatusCode}
	}

	outFile, err := os.Create(tempPkgName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(tempPkgName)
		return err
	}

	if err := os.Rename(tempPkgName, outPkg); err != nil {
		os.Remove(tempPkgName)
		return err
	}
	return nil
}
