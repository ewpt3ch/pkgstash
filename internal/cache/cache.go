package cache

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type Cache struct {
	cacheRoot     string
	mirrorURLs    []string
	mirroredRepos []string
	mirrorIdx     atomic.Uint32
	sf            singleflight.Group //prevents duplicate downloads
	mu            sync.Mutex
	client        http.Client
}

func NewCache(cacheRoot string, mirrorURLs []string, mirroredRepos []string) *Cache {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	return &Cache{
		cacheRoot:     cacheRoot,
		mirrorURLs:    mirrorURLs,
		mirroredRepos: mirroredRepos,
		client: http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Cache) Fetch(pkgPath string) error {
	log.Printf("pkgPath from Fetch %v", pkgPath)
	_, err, _ := c.sf.Do(pkgPath, func() (any, error) {
		log.Print("calling fetch")
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
	pkgURL := c.nextMirror() + pkgName

	log.Printf("fetching %v", pkgURL)

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

func (c *Cache) nextMirror() string {
	idx := c.mirrorIdx.Add(1) - 1
	return c.mirrorURLs[idx%uint32(len(c.mirrorURLs))]
}
