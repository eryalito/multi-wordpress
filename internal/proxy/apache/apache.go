package apache

import (
	"fmt"
	"os"

	cfg "github.com/eryalito/multi-wordpress-file-manager/pkg/config"
)

// ApacheManager configures Apache virtual hosts.
type ApacheManager struct{}

// Configure creates a virtual host configuration file for a site.
func (m *ApacheManager) Configure(site cfg.Site, sitePath string) error {
	vhostConfig := fmt.Sprintf(`
<VirtualHost *>
    ServerName %s
    DocumentRoot %s

    <Directory %s>
        Options Indexes SymLinksIfOwnerMatch
        AllowOverride All
        Require all granted
    </Directory>

    ErrorLog ${APACHE_LOG_DIR}/%s_error.log
    CustomLog ${APACHE_LOG_DIR}/%s_access.log combined
</VirtualHost>
`, site.DomainName, sitePath, sitePath, site.DomainName, site.DomainName)

	configPath := fmt.Sprintf("/etc/apache2/sites-available/%s.conf", site.DomainName)
	return os.WriteFile(configPath, []byte(vhostConfig), 0644)
}

// Enable enables the site by creating a symlink.
func (m *ApacheManager) Enable(site cfg.Site) error {
	src := fmt.Sprintf("/etc/apache2/sites-available/%s.conf", site.DomainName)
	dest := fmt.Sprintf("/etc/apache2/sites-enabled/%s.conf", site.DomainName)

	// a2ensite command is just a symlink, so we can do it directly
	if _, err := os.Lstat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	if err := os.Symlink(src, dest); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}
