package cache

import (
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
)

type fileEntry struct {
	path  string
	size  int64
	atime time.Time
}

func (c *Cache) Clean() error {
	// age always triggers eviction
	// if cache is still over max size after age eviction
	// evict until size is sizeThreshold % of max size
	sizeThreshold := c.cfg.MaxCacheSize * 3 / 4

	cacheSize, cachedFiles, err := c.getCachedFiles()
	if err != nil {
		return err
	}

	// remove any pkgs over max age, update cache size while doing
	evictBefore := time.Now().Add(-c.cfg.MaxCacheAge)
	i := 0
	rsize := int64(0)
	for _, f := range cachedFiles {
		if !f.atime.Before(evictBefore) {
			break
		}
		if err := c.cr.Remove(f.path); err != nil {
			slog.Error("failed to remove file: %v", "file", f.path, "err", err)
			continue
		}
		cacheSize -= f.size
		rsize += f.size
		i++
	}
	cachedFiles = cachedFiles[i:]
	slog.Info("evicted aged out pkgs", "num", i, "size", humanize.Bytes(uint64(rsize)))

	// if cache size < max size we're done
	if cacheSize < c.cfg.MaxCacheSize {
		return nil
	}

	// remove oldest files until cache size < threshold
	i = 0
	rsize = 0
	for i < len(cachedFiles) && cacheSize > sizeThreshold {
		f := cachedFiles[i]
		if err := c.cr.Remove(f.path); err != nil {
			slog.Error("failed to remove file: %v", "file", f.path, "err", err)
			continue
		}
		cacheSize -= f.size
		rsize += f.size
		i++
	}
	slog.Info("evicted pkgs over size limit", "num", i, "size", humanize.Bytes(uint64(rsize)))

	return nil
}

func (c *Cache) getCachedFiles() (int64, []fileEntry, error) {
	// returns total cache size and sorted slice, oldest first,
	// of all fileEntry where name is the relative path
	// from the cacheroot
	cacheSize := int64(0)
	cachedFiles := []fileEntry{}
	for _, repo := range c.cfg.mirroredRepos {
		relPath := filepath.Join(repo, "os/x86_64")
		f, err := c.cr.Open(relPath)
		if err != nil {
			return 0, nil, err
		}
		files, err := f.ReadDir(-1)
		if err != nil {
			return 0, nil, err
		}
		if err := f.Close(); err != nil {
			return 0, nil, err
		}
		for _, f := range files {
			fInfo, err := f.Info()
			if err != nil {
				return 0, nil, err
			}
			if strings.HasSuffix(f.Name(), ".gz") || strings.HasSuffix(f.Name(), ".db") {
				continue
			}
			name := filepath.Join(repo, repoArch, f.Name())
			size := fInfo.Size()
			stat_t := fInfo.Sys().(*syscall.Stat_t)
			atime := time.Unix(stat_t.Atim.Sec, stat_t.Atim.Nsec)
			entry := fileEntry{
				path:  name,
				size:  size,
				atime: atime,
			}
			cachedFiles = append(cachedFiles, entry)
			cacheSize += size

		}
	}
	slices.SortFunc(cachedFiles, func(a, b fileEntry) int {
		return a.atime.Compare(b.atime)
	})
	return cacheSize, cachedFiles, nil
}
