package cache

import (
	"os"
	"path/filepath"
)

func (c *Cache) getStream(relPath string) (*inFlight, *os.File, error) {

	// lock the map
	c.inFlightMu.Lock()
	defer c.inFlightMu.Unlock()

	// download in progress connect new stream to existing download
	if flight, ok := c.inFlight[relPath]; ok {
		f, err := c.cr.Open(flight.tmpPath)
		if err != nil {
			return nil, nil, err
		}
		return flight, f, nil
	}

	// download not in progress, setup new download and stream

	// make sure the dir structure exists
	if err := c.cr.MkdirAll(filepath.Dir(relPath), 0750); err != nil {
		return nil, nil, err
	}
	// use a tmp file for the initial fetch in case it fails
	tmpPath := relPath + ".tmp"
	tmpFile, err := c.cr.Create(tmpPath)
	if err != nil {
		return nil, nil, err
	}

	flight := &inFlight{
		tmpPath: tmpPath,
	}

	c.inFlight[relPath] = flight

	// spin off a downloading func so we can return the file handler
	// downloadWrangle takes ownership of tmpFile and closes it.
	go c.downloadWrangle(relPath, flight, tmpFile)
	f, err := c.cr.Open(flight.tmpPath)
	if err != nil {
		return nil, nil, err
	}
	return flight, f, nil
}
