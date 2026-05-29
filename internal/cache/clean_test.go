package cache

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCachedPkgs(t *testing.T, prefix, name string, num, size int64, age time.Time, cr *os.Root) {
	t.Helper()
	for range num {
		diffStamp := time.Now().UnixNano()
		tmpFileName := fmt.Sprintf("%s%s-pkg-%d.pkg.tar.zst", prefix, name, diffStamp)
		fileSize := size / num
		contents := bytes.Repeat([]byte("a"), int(fileSize))
		fmt.Printf("creating file: %s\n", tmpFileName)
		if err := cr.WriteFile(tmpFileName, []byte(contents), 0644); err != nil {
			t.Fatalf("failed to create testPkgs: %v", err)
		}
		if err := cr.Chtimes(tmpFileName, age, age); err != nil {
			t.Fatalf("failed to set time stamp: %v", err)
		}
	}
}

func TestGetCachedPkgs(t *testing.T) {
	t.Run("get only files ending in .pkg.tar.zst", func(t *testing.T) {
		c := newTestCache(t, []string{"http//example.com"})
		// populate cache and store names of expected files
		prefix := filepath.Join("core", repoArch)
		contents := "testfile"
		expectedFiles := []string{}
		expectedSize := int64(0)
		for i := range 5 {
			testFilePath := filepath.Join(prefix, fmt.Sprintf("pkg-%d.pkg.tar.zst", i))
			expectedFiles = append(expectedFiles, testFilePath)
			expectedSize += int64(len(contents))
			if err := c.cr.WriteFile(testFilePath, []byte(contents), 0644); err != nil {
				t.Fatalf("failed to create testfiles")
			}
		}
		//
		testFilePath := filepath.Join(prefix, "core.db.tar.gz")
		if err := c.cr.WriteFile(testFilePath, []byte("testfile"), 0644); err != nil {
			t.Fatalf("failed to create testfile")
		}
		size, files, err := c.getCachedFiles()
		require.NoError(t, err)
		assert.Equal(t, expectedSize, size)
		for _, f := range files {
			assert.Contains(t, expectedFiles, f.path)
		}
	})
}

func TestClean(t *testing.T) {
	t.Run("remove based on age", func(t *testing.T) {
		c := newTestCache(t, []string{"http://example.com"})
		//populate cacheA
		numFiles := int64(5)
		prefix := filepath.Join("core", repoArch)
		evictTime := time.Now().Add(-c.cfg.MaxCacheAge * 2)
		newTestCachedPkgs(t, prefix, "/evict", numFiles, int64(10), evictTime, c.cr)
		newTestCachedPkgs(t, prefix, "/keep", numFiles, c.cfg.MaxCacheSize/2, time.Now(), c.cr)

		err := c.Clean()
		require.NoError(t, err)
		size, files, err := c.getCachedFiles()
		if err != nil {
			t.Fatal("failed to get cached files")
		}
		assert.LessOrEqual(t, size, c.cfg.MaxCacheSize)
		assert.Len(t, files, int(numFiles))
		for _, f := range files {
			fmt.Printf("checking file %s\n", f.path)
			assert.Contains(t, f.path, "keep")
			assert.NotContains(t, f.path, "evict")
		}
	})

	t.Run("evict based on size", func(t *testing.T) {
		c := newTestCache(t, []string{"http://example.com"})
		//populate cache
		numFiles := int64(5)
		prefix := filepath.Join("core", repoArch)
		evictTime := time.Now().Add(-time.Millisecond * 100)
		newTestCachedPkgs(t, prefix, "/evict", numFiles, c.cfg.MaxCacheSize, evictTime, c.cr)
		newTestCachedPkgs(t, prefix, "/keep", numFiles, c.cfg.MaxCacheSize*3/4, time.Now(), c.cr)

		err := c.Clean()
		require.NoError(t, err)
		size, files, err := c.getCachedFiles()
		if err != nil {
			t.Fatal("failed to get cached files")
		}
		assert.LessOrEqual(t, size, c.cfg.MaxCacheSize)
		assert.Len(t, files, int(numFiles))
		for _, f := range files {
			fmt.Printf("checking file %s\n", f.path)
			assert.Contains(t, f.path, "keep")
			assert.NotContains(t, f.path, "evict")
		}
	})

	t.Run("evict based on age and size", func(t *testing.T) {
		c := newTestCache(t, []string{"http://example.com"})
		//populate cache
		numFiles := int64(5)
		prefix := filepath.Join("core", repoArch)
		evictTime := time.Now().Add(-c.cfg.MaxCacheAge * 2)
		newTestCachedPkgs(t, prefix, "/evict-age", numFiles, c.cfg.MaxCacheSize, evictTime, c.cr)
		newTestCachedPkgs(t, prefix, "/evict-size", numFiles, c.cfg.MaxCacheSize, time.Now(), c.cr)
		newTestCachedPkgs(t, prefix, "/keep", numFiles, c.cfg.MaxCacheSize*3/4, time.Now(), c.cr)

		err := c.Clean()
		require.NoError(t, err)
		size, files, err := c.getCachedFiles()
		if err != nil {
			t.Fatal("failed to get cached files")
		}
		assert.LessOrEqual(t, size, c.cfg.MaxCacheSize)
		assert.Len(t, files, int(numFiles))
		for _, f := range files {
			fmt.Printf("checking file %s\n", f.path)
			assert.Contains(t, f.path, "keep")
			assert.NotContains(t, f.path, "evict")
		}
	})
}
