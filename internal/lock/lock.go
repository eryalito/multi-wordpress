package lock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// Info describes the process that currently holds the lock.
type Info struct {
	Member   string    `json:"member"`
	Hostname string    `json:"hostname"`
	PID      int       `json:"pid"`
	Acquired time.Time `json:"acquired"`
}

// Acquire tries to acquire an exclusive lock on the given file path. The lock
// is kept until the returned release function is called, or the process exits.
// If the context is canceled before the lock is acquired, it returns context error.
func Acquire(ctx context.Context, path string, member string) (release func() error, _ error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("ensure lock dir: %w", err)
	}
	f := flock.New(path)

	// Attempt to acquire with polling so we can respect context.
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		locked, err := f.TryLock()
		if err != nil {
			return nil, fmt.Errorf("try lock: %w", err)
		}
		if locked {
			// Write sidecar info for visibility (best-effort).
			_ = writeInfo(path+".json", member)
			return func() error {
				_ = os.Remove(path + ".json")
				return f.Unlock()
			}, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func writeInfo(infoPath, member string) error {
	host, _ := os.Hostname()
	b, _ := json.MarshalIndent(Info{
		Member:   member,
		Hostname: host,
		PID:      os.Getpid(),
		Acquired: time.Now(),
	}, "", "  ")
	if len(b) == 0 {
		return errors.New("empty info")
	}
	// Best-effort write; ignore errors to avoid blocking lock acquisition.
	_ = os.WriteFile(infoPath, b, 0o644)
	return nil
}
