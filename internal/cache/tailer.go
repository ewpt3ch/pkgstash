package cache

import (
	"io"
	"log/slog"
	"os"
	"time"
)

type tailer struct {
	f      *os.File
	flight *inFlight
}

func (t *tailer) Read(p []byte) (int, error) {
	for {
		n, err := t.f.Read(p)
		slog.Debug("tailer read", "n", n, "err", err)
		if n > 0 {
			return n, nil
		}
		if err == io.EOF {
			select {
			case <-t.flight.done:
				if t.flight.err != nil {
					return 0, t.flight.err
				}
				return t.f.Read(p) // send remainiing bytes
			case <-time.After(50 * time.Millisecond):
				slog.Debug("tailer waiting for more data")
				continue
			}
		}
		return n, err
	}
}

func (t *tailer) Close() error {
	return t.f.Close()
}
