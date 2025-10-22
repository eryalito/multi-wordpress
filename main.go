package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	internalCfg "github.com/eryalito/multi-wordpress-file-manager/internal/config"
	"github.com/eryalito/multi-wordpress-file-manager/internal/lock"
	"github.com/eryalito/multi-wordpress-file-manager/internal/worker"
	publicCfg "github.com/eryalito/multi-wordpress-file-manager/pkg/config"
)

func parseFlags() (cfgPath string, lockPath string, member string, lockTimeout *time.Duration, interval *time.Duration) {
	cfg := flag.String("config", "config.yaml", "Path to YAML configuration file")
	lockP := flag.String("lock", "", "Path to lock file on shared filesystem (optional; defaults next to config)")
	mem := flag.String("member", "", "Identifier for this instance (defaults to hostname)")
	lto := flag.Duration("lock-timeout", 0, "Max time to wait to acquire the lock (0=wait forever)")
	iv := flag.Duration("interval", 3*time.Minute, "Worker interval (e.g. 3m, 30s)")
	flag.Parse()
	return *cfg, *lockP, *mem, lto, iv
}

func setupContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func acquireLock(ctx context.Context, cfgPath, lockPath, member string, lockTimeout *time.Duration) (context.Context, func(), error) {
	lp := lockPath
	if lp == "" {
		lp = filepath.Join(filepath.Dir(cfgPath), ".multi-wordpress-file-manager.lock")
	}
	m := member
	if m == "" {
		if h, err := os.Hostname(); err == nil {
			m = h
		}
	}
	lockCtx := ctx
	if *lockTimeout > 0 {
		var cancel context.CancelFunc
		lockCtx, cancel = context.WithTimeout(ctx, *lockTimeout)
		defer cancel()
	}
	rel, err := lock.Acquire(lockCtx, lp, m)
	if err != nil {
		return ctx, nil, err
	}
	cleanup := func() {
		if err := rel(); err != nil {
			log.Printf("lock release error: %v", err)
		}
	}
	log.Printf("acquired lock: %s", lp)
	return ctx, cleanup, nil
}

func loadInitialConfig(cfgPath string) *publicCfg.Config {
	cfg, err := internalCfg.Load(cfgPath)
	if err != nil {
		log.Printf("config load: %v", err)
		return nil
	}
	log.Printf("config loaded: %d site(s)", len(cfg.Sites))
	return cfg
}

func startWorker(ctx context.Context, currentCfg func() *publicCfg.Config, interval *time.Duration) *worker.Worker {
	w := worker.New(worker.Handle, currentCfg, *interval, log.Printf)
	go w.Start(ctx)
	return w
}

func startWatcher(ctx context.Context, cfgPath string, onReload func(*publicCfg.Config)) error {
	_, err := internalCfg.Watch(ctx, cfgPath, func(cfg *publicCfg.Config, err error) {
		if err != nil {
			log.Printf("config reload error: %v", err)
			return
		}
		onReload(cfg)
		log.Printf("config reloaded: %d site(s)", len(cfg.Sites))
	})
	return err
}

func main() {
	cfgPath, lockPath, member, lockTimeout, interval := parseFlags()

	ctx, cancel := setupContext()
	defer cancel()

	if _, cleanup, err := acquireLock(ctx, cfgPath, lockPath, member, lockTimeout); err != nil {
		log.Fatalf("failed to acquire lock: %v", err)
	} else {
		defer cleanup()
	}

	var cfgVal atomic.Value
	cfg := loadInitialConfig(cfgPath)
	cfgVal.Store(cfg)

	w := startWorker(ctx, func() *publicCfg.Config {
		v := cfgVal.Load()
		if v == nil {
			return nil
		}
		return v.(*publicCfg.Config)
	}, interval)

	if err := startWatcher(ctx, cfgPath, func(c *publicCfg.Config) {
		cfgVal.Store(c)
		w.Trigger()
	}); err != nil {
		log.Fatalf("watch start: %v", err)
	}

	log.Printf("watching %s for changes...", cfgPath)
	<-ctx.Done()
	log.Printf("shutting down")
}
