package cache

import (
	"log"
	"os"
	"path/filepath"
)

func (c *Cache) Fetch(relPath string) (*CacheFile, error) {
	// return file directly if exists in cache
	cf, err := getCachedFile(c.cfg.cacheRoot, relPath)
	if err == nil {
		return cf, nil
	}

	// fetch file from upstream
	_, err, _ = c.sf.Do(relPath, func() (any, error) {
		// #log info
		log.Print("calling fetch")
		return nil, c.fetch(relPath)
	})
	if err != nil {
		return nil, err
	}

	cf, err = getCachedFile(c.cfg.cacheRoot, relPath)
	if err != nil {
		return nil, err
	}
	return cf, nil
}

func (c *Cache) fetch(relPath string) error {
	// relPath is relative to the localRoot
	// ie relPath includes /{repo}/os/{arch}/ and the actual name linux-x.x.x.pkg.tar.zst

	// final file name and path
	destPath := filepath.Join(c.cfg.cacheRoot, relPath)

	// declare vars outside loop
	var err error
	// fetch pkgs from mirror with retry logic
	for range len(c.cfg.mirrorURLs) {
		url := c.nextMirror() + relPath
		err = downloadToDisk(url, destPath, c.client)
		if err == nil {
			break
		}
		// #log warn or info
		log.Printf("mirror %s returned %v", url, err)
	}
	if err != nil {
		return err
	}
	return nil
}

func getCachedFile(cacheRoot, relPath string) (*CacheFile, error) {
	filePath := filepath.Join(cacheRoot, relPath)
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	return &CacheFile{
		Reader:   f,
		Size:     info.Size(),
		Filename: filepath.Base(filePath),
	}, nil
}
