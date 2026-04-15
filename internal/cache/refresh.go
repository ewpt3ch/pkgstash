package cache

import "path/filepath"

func (c *Cache) Refresh() error {
	if !c.mu.TryLock() {
		return nil
	}
	defer c.mu.Unlock()

	for _, repo := range c.mirroredRepos {
		if err := c.refreshDB(repo); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cache) refreshDB(repo string) error {
	dbFile := repo + ".db.tar.gz"
	dbPath := filepath.Join(repo, "os/x86_64", dbFile)
	err := c.Fetch(dbPath)
	if err != nil {
		return err
	}
	return nil
}
