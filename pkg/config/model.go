package config

type ProxyType string

var (
	ProxyTypeApache ProxyType = "apache"
)

// Site represents a single WordPress site managed by this app.
type Site struct {
	Name       string `yaml:"name"`
	DomainName string `yaml:"domain_name"`
	Root       string `yaml:"root"`
}

type Proxy struct {
	Type ProxyType `yaml:"type"` // e.g. "apache"
}

// Config is the root configuration structure loaded from YAML.
type Config struct {
	Sites []Site `yaml:"sites"`
	Proxy Proxy  `yaml:"proxy"`
}
