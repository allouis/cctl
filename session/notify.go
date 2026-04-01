package session

import (
	"net"
	"os"
	"path/filepath"
)

// NotifySocketPath returns the path to the notification socket for a config dir.
func NotifySocketPath(dir string) string {
	return filepath.Join(dir, "notify.sock")
}

// ListenNotify creates a Unix socket at path and returns a channel that
// receives a value each time a hook process connects. The returned func
// stops the listener and cleans up the socket file.
func ListenNotify(path string) (<-chan struct{}, func(), error) {
	// Remove stale socket from a previous run.
	os.Remove(path)

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan struct{}, 1)
	done := make(chan struct{})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			conn.Close()

			// Non-blocking send — if a signal is already pending, skip.
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}()

	cleanup := func() {
		close(done)
		ln.Close()
		os.Remove(path)
	}

	return ch, cleanup, nil
}

// SignalNotify connects to the Unix socket at path and immediately closes
// the connection. This is fire-and-forget; errors are silently ignored
// (the server may not be running).
func SignalNotify(path string) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return
	}
	conn.Close()
}
