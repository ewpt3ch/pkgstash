package cache

import "path/filepath"

func (c *Cache) Refresh() error {
	if !c.refreshMu.TryLock() {
		return nil
	}
	defer c.refreshMu.Unlock()

	for _, repo := range c.cfg.mirroredRepos {
		if err := c.refreshDB(repo); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cache) refreshDB(repo string) error {
	dbFile := repo + ".db.tar.gz"
	dbPath := filepath.Join(repo, "os/x86_64", dbFile)
	_, _, err := c.getStream(dbPath)
	if err != nil {
		return err
	}
	return nil
}
