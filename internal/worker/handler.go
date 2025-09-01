package worker

import (
	"context"
	"log"

	cfgpkg "github.com/eryalito/multi-wordpress-file-manager/pkg/config"
)

// Handle is the default worker function. Replace its body with real work.
func Handle(ctx context.Context, cfg *cfgpkg.Config) error {
	if cfg == nil {
		log.Printf("worker: no config loaded yet; skipping run")
		return nil
	}

	return nil
}
