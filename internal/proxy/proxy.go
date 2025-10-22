package proxy

import (
	cfg "github.com/eryalito/multi-wordpress-file-manager/pkg/config"
)

// Manager is an interface for proxy managers.
type Manager interface {
	Configure(site cfg.Site, sitePath string) error
	Enable(site cfg.Site) error
}
