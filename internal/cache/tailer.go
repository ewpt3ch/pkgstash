package cache

import (
	"io"
	"os"
	"time"
)

type tailer struct {
	f      *os.File
	done   <-chan struct{}
	flight *inFlight
}

func (t *tailer) Read(p []byte) (int, error) {
	for {
		n, err := t.f.Read(p)
		if n > 0 {
			return n, nil
		}
		if err == io.EOF {
			select {
			case <-t.done:
				if t.flight.err != nil {
					return 0, t.flight.err
				}
				return t.f.Read(p) // send remainiing bytes
			default:
				time.Sleep(50 * time.Millisecond)
				continue
			}
		}
		return n, err
	}
}

func (t *tailer) Close() error {
	return t.f.Close()
}
