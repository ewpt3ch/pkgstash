package repomaint

import (
	"fmt"
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
	FetchDB(repo string) error
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
		slog.Info("getting list of cached pkgs", "repo", repo)
		cachedPkgs, err := r.buildMap(repo)
		if err != nil {
			// pass through to cover initialization of repos
			slog.Warn("failed to read current db", "err", err)
		}
		fmt.Printf("cached pkgs: %v", cachedPkgs)

		// call cache db fetch
		slog.Info("refreshing databases")
		if err := r.c.FetchDB(repo); err != nil {
			return err
		}

		// pkgsToUpdate slice contains relative paths like
		// <repo>/os/<arch>/filename
		slog.Info("checking for updates")
		pkgsToUpdate, err := r.updatablePkgs(repo, cachedPkgs)
		if err != nil {
			slog.Warn("failed to get updatable pkgs list", "err", err)
			return err
		}
		fmt.Printf("updatable pkgs: %v", pkgsToUpdate)

		for _, fileName := range pkgsToUpdate {
			slog.Debug("fetching pkgs", "pkg", fileName)
			path := filepath.Join(repo, repoArch, fileName)
			_, err := r.c.Fetch(path)
			if err != nil {
				slog.Warn("failed to update pkg", "pkg", fileName, "err", err)
			}

		}
		slog.Info("finished pkg update")

		// call cache cleanup
	}
	return nil
}
