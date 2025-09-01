package worker

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	cfgpkg "github.com/eryalito/multi-wordpress-file-manager/pkg/config"
)

// Handle is the default worker function. Replace its body with real work.
func Handle(ctx context.Context, cfg *cfgpkg.Config) error {
	if cfg == nil {
		log.Printf("worker: no config loaded yet; skipping run")
		return nil
	}

	log.Println("worker: starting wordpress deployment check")

	// Path for the downloaded wordpress zip
	zipPath := "/tmp/wordpress.zip"

	// Check if wordpress zip is already downloaded
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		log.Printf("worker: wordpress not found at %s, downloading from %s", zipPath, cfg.WordpressGlobal.ZipURL)
		if err := downloadFile(zipPath, cfg.WordpressGlobal.ZipURL); err != nil {
			return fmt.Errorf("worker: failed to download wordpress: %w", err)
		}
		log.Printf("worker: wordpress downloaded successfully to %s", zipPath)
	} else {
		log.Printf("worker: found existing wordpress zip at %s", zipPath)
	}

	// Ensure the BasePath directory exists
	if err := os.MkdirAll(cfg.WordpressGlobal.BasePath, os.ModePerm); err != nil {
		return fmt.Errorf("worker: failed to create base path directory %s: %w", cfg.WordpressGlobal.BasePath, err)
	}

	for _, site := range cfg.Sites {
		sitePath := filepath.Join(cfg.WordpressGlobal.BasePath, site.DomainName)
		log.Printf("worker: processing site %s at path %s", site.DomainName, sitePath)

		wpSettingsPath := filepath.Join(sitePath, "wp-settings.php")
		if _, err := os.Stat(wpSettingsPath); os.IsNotExist(err) {
			log.Printf("worker: wordpress not installed for site %s, installing now", site.DomainName)

			// Create site directory if it doesn't exist
			if err := os.MkdirAll(sitePath, os.ModePerm); err != nil {
				return fmt.Errorf("worker: failed to create site directory %s: %w", sitePath, err)
			}

			// Unzip wordpress
			if err := unzip(zipPath, sitePath); err != nil {
				return fmt.Errorf("worker: failed to unzip wordpress for site %s: %w", site.DomainName, err)
			}
			log.Printf("worker: successfully unzipped wordpress for site %s", site.DomainName)
		}

		// Ensure wp-config.php is present and correct
		if err := ensureWPConfig(sitePath, site); err != nil {
			return fmt.Errorf("worker: failed to ensure wp-config.php for site %s: %w", site.DomainName, err)
		}
	}

	log.Println("worker: finished wordpress deployment check")
	return nil
}

func ensureWPConfig(sitePath string, site cfgpkg.Site) error {
	wpConfigPath := filepath.Join(sitePath, "wp-config.php")
	wpConfig := site.Wordpress

	if _, err := os.Stat(wpConfigPath); os.IsNotExist(err) {
		log.Printf("worker: wp-config.php not found for site %s, creating it", site.DomainName)
		return createWPConfig(sitePath, wpConfig)
	}

	// File exists, check if an update is needed.
	content, err := os.ReadFile(wpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read existing wp-config.php: %w", err)
	}

	// Extract current config from file content
	currentConfig := parseWPConfig(content)

	// Compare with new config from yaml
	newDBHost := fmt.Sprintf("%s:%d", wpConfig.Database.Host, wpConfig.Database.Port)
	if currentConfig["DB_NAME"] == wpConfig.Database.Name &&
		currentConfig["DB_USER"] == wpConfig.Database.User &&
		currentConfig["DB_PASSWORD"] == wpConfig.Database.Password &&
		currentConfig["DB_HOST"] == newDBHost {
		log.Printf("worker: wp-config.php for site %s is up to date", site.DomainName)
		return nil
	}

	log.Printf("worker: database configuration for site %s has changed, updating wp-config.php", site.DomainName)

	// Config has changed, regenerate the file preserving salts.
	salts := extractSalts(content)
	if salts == "" {
		log.Printf("worker: could not find salts in existing wp-config.php for site %s, fetching new ones.", site.DomainName)
		salts, err = getSalts()
		if err != nil {
			return fmt.Errorf("failed to get new salts: %w", err)
		}
	}

	return writeWPConfig(sitePath, wpConfig, salts)
}

// parseWPConfig extracts database credentials from wp-config.php content.
func parseWPConfig(content []byte) map[string]string {
	config := make(map[string]string)
	// Regex to find define('KEY', 'VALUE');
	re := regexp.MustCompile(`define\(\s*'([^']*)'\s*,\s*'([^']*)'\s*\);`)
	matches := re.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		if len(match) == 3 {
			config[match[1]] = match[2]
		}
	}
	return config
}

// extractSalts pulls the block of salt definitions from wp-config.php content.
func extractSalts(content []byte) string {
	re := regexp.MustCompile(`(?s)define\(\s*'AUTH_KEY'.*require_once`)
	match := re.Find(content)
	if match != nil {
		// Trim the require_once from the end
		matchStr := string(match)
		lastIndex := strings.LastIndex(matchStr, "require_once")
		if lastIndex != -1 {
			return strings.TrimSpace(matchStr[:lastIndex])
		}
	}
	return ""
}

// createWPConfig creates a new wp-config.php file, fetching new salts.
func createWPConfig(dest string, wpConfig cfgpkg.Wordpress) error {
	salts, err := getSalts()
	if err != nil {
		return fmt.Errorf("failed to get salts: %w", err)
	}
	return writeWPConfig(dest, wpConfig, salts)
}

// writeWPConfig writes the wp-config.php file with the given credentials and salts.
func writeWPConfig(dest string, wpConfig cfgpkg.Wordpress, salts string) error {
	wpConfigPath := filepath.Join(dest, "wp-config.php")
	configContent := fmt.Sprintf(`<?php
define( 'DB_NAME', '%s' );
define( 'DB_USER', '%s' );
define( 'DB_PASSWORD', '%s' );
define( 'DB_HOST', '%s' );
define( 'DB_CHARSET', 'utf8' );
define( 'DB_COLLATE', '' );

%s

$table_prefix = 'wp_';

define( 'WP_DEBUG', false );

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}

require_once ABSPATH . 'wp-settings.php';
`, wpConfig.Database.Name, wpConfig.Database.User, wpConfig.Database.Password, fmt.Sprintf("%s:%d", wpConfig.Database.Host, wpConfig.Database.Port), salts)

	return os.WriteFile(wpConfigPath, []byte(configContent), 0644)
}

// getSalts fetches unique keys and salts from the WordPress.org API.
func getSalts() (string, error) {
	resp, err := http.Get("https://api.wordpress.org/secret-key/1.1/salt/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// downloadFile will download a url to a file. It's not efficient for large files.
func downloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// This is to handle the case where the zip file contains a single folder
	// with the content, like "wordpress-6.0.2/..."
	var basePath string
	if len(r.File) > 0 {
		parts := strings.Split(r.File[0].Name, "/")
		if len(parts) > 1 && strings.HasSuffix(r.File[0].Name, "/") {
			basePath = parts[0] + "/"
		}
	}

	for _, f := range r.File {
		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, strings.TrimPrefix(f.Name, basePath))

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		cleanDest := filepath.Clean(dest)
		if fpath != cleanDest && !strings.HasPrefix(fpath, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
