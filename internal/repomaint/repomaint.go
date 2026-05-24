package repomaint

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ewpt3ch/pkgstash/internal/cache"
)

const (
	repoArch  = "os/x86_64"
	dbSuffix  = ".db.tar.gz"
	pkgSuffix = "-x86_64.pkg.tar.zst"
)

type CacheClient interface {
	FetchDB() error
	Fetch(relpath string) (*cache.CacheFile, error)
}

type RepoSync struct {
	c     CacheClient
	root  *os.Root
	repos []string
}

func NewRepoSync(c CacheClient, path string, repos []string) (*RepoSync, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	rs := RepoSync{
		c:     c,
		root:  root,
		repos: repos,
	}
	return &rs, nil
}

func (r *RepoSync) Sync() error {

	for _, repo := range r.repos {
		// create map of pkgname to filenames from current db
		cachedPkgs, err := r.buildMap(repo)
		if err != nil {
			slog.Warn("failed to read current db", "err", err)
			return err
		}

		// call cache db fetch
		if err := r.c.FetchDB(); err != nil {
			return err
		}

		// pkgsToUpdate slice contains relative paths like
		// <repo>/os/<arch>/filename
		pkgsToUpdate, err := r.updatablePkgs(repo, cachedPkgs)
		if err != nil {
			slog.Warn("failed to get updatable pkgs list", "err", err)
			return err
		}

		for _, fileName := range pkgsToUpdate {
			path := filepath.Join(repo, repoArch, fileName)
			_, err := r.c.Fetch(path)
			if err != nil {
				slog.Warn("failed to update pkg", "pkg", fileName, "err", err)
			}

		}

		// call cache cleanup
	}
	return nil
}
