package worker

import (
	"context"
	"log"

	cfgpkg "multi-wordpress-file-manager/internal/config"
)

// Handle is the default worker function. Replace its body with real work.
func Handle(ctx context.Context, cfg *cfgpkg.Config) error {
	if cfg == nil {
		log.Printf("worker: no config loaded yet; skipping run")
		return nil
	}
	// TODO: implement your actual periodic work here.
	log.Printf("worker: processing %d site(s)", len(cfg.Sites))
	return nil
}
