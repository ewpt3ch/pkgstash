package repomaint

import (
	"os"

	"github.com/ewpt3ch/pkgstash/internal/cache"
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
	// create map of pkgname to filenames from current db
	// call cache db fetch
	if err := r.c.FetchDB(); err != nil {
		return err
	}
	// create map from pkgnames in new db
	// compare and fetch
	// call cache cleanup
	return nil
}
