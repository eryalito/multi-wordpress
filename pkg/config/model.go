package config

type ProxyType string

var (
	ProxyTypeApache ProxyType = "apache"
)

type Proxy struct {
	Type ProxyType `yaml:"type"` // e.g. "apache"
}

type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

type Wordpress struct {
	Database   Database `yaml:"database"`
	ForceHTTPS *bool    `yaml:"force_https"`
}

type WordpressGlobal struct {
	ZipURL   string `yaml:"zip_url"`
	BasePath string `yaml:"base_path"`
}

type Site struct {
	DomainName string    `yaml:"domain_name"`
	Wordpress  Wordpress `yaml:"wordpress"`
}

type Config struct {
	Sites           []Site          `yaml:"sites"`
	Proxy           Proxy           `yaml:"proxy"`
	WordpressGlobal WordpressGlobal `yaml:"wordpress_global"`
}
