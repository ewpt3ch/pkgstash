package repomaint

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ewpt3ch/pkgstash/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// create a CacheClient to mock cache.Fetch and cache.FetchDB to fulfill
// expected interface in RepoSync struct
type mockCache struct {
	fetched       []string
	fetchDBCalled bool
}

func (m *mockCache) FetchDB(repo string) error {
	m.fetchDBCalled = true
	return nil
}
func (m *mockCache) Fetch(relPath string) (*cache.CacheFile, error) {
	m.fetched = append(m.fetched, relPath)
	return nil, nil
}

func newTestRepo(t *testing.T, repoPath string, rs *RepoSync) {
	// create newTestRepo with db file
	t.Helper()

	src, err := os.Open(filepath.Join("testdata", "pre.core.db.tar.gz"))
	if err != nil {
		t.Fatalf("unable to open src db for copy: %v", err)
	}
	defer src.Close()
	rs.root.MkdirAll(repoPath, 0750)
	dst, err := rs.root.Create(filepath.Join(repoPath, "core.db.tar.gz"))
	if err != nil {
		t.Fatalf("unable to open dst db for copy: %s", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("failed to copy db: %v", err)
	}
}

func TestParseDesc(t *testing.T) {
	t.Run("parse good desc file", func(t *testing.T) {
		r := strings.NewReader("%NAME%\nlinux\n\n\n%FAKEKEY%\nFAKEINFO\n\n%FILENAME%\nlinux-7.0.9\n")
		pkgName, fileName, err := parseDesc(r)
		require.NoError(t, err)
		assert.Equal(t, "linux", pkgName)
		assert.Equal(t, "linux-7.0.9", fileName)
	})

	t.Run("parse bad desc file", func(t *testing.T) {
		r := strings.NewReader("%NAME\nlinux\n\n%FAKEKEY%\nwhat\n")
		pkgName, fileName, err := parseDesc(r)
		assert.Error(t, err)
		assert.Equal(t, "", pkgName)
		assert.Equal(t, "", fileName)
	})
}

func TestCachedFiles(t *testing.T) {
	testFiles := []string{
		"linux-x86_64.pkg.tar.zst",
		"gcc-rust-x86_64.pkg.tar.zst",
		"linux-api-headers-x86_64.pkg.tar.zst",
	}
	c := &mockCache{
		fetched: make([]string, 5),
	}
	rs, err := NewRepoSync(c, t.TempDir(), []string{"core"})
	if err != nil {
		t.Fatalf("failed to create RepoMaint: %v", err)
	}
	repoPath := filepath.Join("core", repoArch)
	newTestRepo(t, repoPath, rs)
	for _, f := range testFiles {
		rs.root.WriteFile(filepath.Join(repoPath, f), []byte{}, 0644)
	}

	cachedFiles, err := rs.getCachedPkgs("core")
	require.NoError(t, err)
	assert.Contains(t, cachedFiles, "linux")
	assert.Contains(t, cachedFiles, "gcc-rust")
	assert.Contains(t, cachedFiles, "linux-api-headers")
}

func TestBuildMap(t *testing.T) {
	testFiles := make(map[string]string)
	testFiles["linux"] = "linux-7.0.9.arch1-1-x86_64.pkg.tar.zst"
	testFiles["linux-api-headers"] = "linux-api-headers-6.19-1-x86_64.pkg.tar.zst"
	testFiles["gcc-rust"] = "gcc-rust-16.1.1+r12+g301eb08fa2c5-1-x86_64.pkg.tar.zst"

	t.Run("build map from real db file", func(t *testing.T) {
		// create test structs
		c := &mockCache{
			fetched: make([]string, 5),
		}
		rs, err := NewRepoSync(c, t.TempDir(), []string{"core"})
		if err != nil {
			t.Fatalf("failed to create RepoMaint: %v", err)
		}
		repoPath := filepath.Join(rs.repos[0], repoArch)
		newTestRepo(t, repoPath, rs)
		for _, f := range testFiles {
			rs.root.WriteFile(filepath.Join(repoPath, f), []byte{}, 0644)
		}

		// run tests

		pkgMap, err := rs.buildMap(rs.repos[0])
		if err != nil {
			t.Fatalf("failed to create map: %v", err)
		}
		for key, val := range testFiles {
			assert.Equal(t, val, pkgMap[key])
		}
	})

}

func TestUpdatablePkgs(t *testing.T) {
	repo := "core"
	testMap := make(map[string]string)
	testMap["linux"] = "linux-7.0.9.arch1-1"
	testMap["linux-api-headers"] = "linux-a"
	testMap["gcc-rust"] = "gcc-rust-16.1.1+r12+g301eb08fa2c5-1-x86_64.pkg.tar.zst"
	expectedFileName1 := "linux-7.0.9.arch1-1-x86_64.pkg.tar.zst"
	expectedFileName2 := "linux-api-headers-6.19-1-x86_64.pkg.tar.zst"
	unexpectedFileName := "gcc-rust-16.1.1+r12+g301eb08fa2c5-1-x86_64.pkg.tar.zst"

	// create test structs
	c := &mockCache{
		fetched: make([]string, 5),
	}
	rs, err := NewRepoSync(c, t.TempDir(), []string{"core"})
	if err != nil {
		t.Fatalf("failed to create RepoMaint: %v", err)
	}
	repoPath := filepath.Join(rs.repos[0], repoArch)
	newTestRepo(t, repoPath, rs)

	files, err := rs.updatablePkgs(repo, testMap)
	require.NoError(t, err)
	assert.Contains(t, files, expectedFileName1)
	assert.Contains(t, files, expectedFileName2)
	assert.NotContains(t, files, unexpectedFileName)

}

func TestSyncNoDB(t *testing.T) {
	// can't create db and test initialization behavior without
	// going out of scope. We test fall through to FetchDB and
	// second attempt to read db file fails
	c := &mockCache{
		fetched: make([]string, 5),
	}
	rs, err := NewRepoSync(c, t.TempDir(), []string{"core"})
	if err != nil {
		t.Fatalf("failed to create RepoSync: %v", err)
	}

	err = rs.Sync()
	// err here is correct behavior as no db exists
	assert.Error(t, err)
	assert.True(t, c.fetchDBCalled)

}
