package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configFileName = ".testlens.yml"

type Package struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

type Config struct {
	Packages []Package `yaml:"packages"`
}

// Load attempts to find and parse a .testlens.yml config file.
// It walks up from startDir to the filesystem root looking for the file.
// If configPath is non-empty, it uses that path directly.
// Returns nil (no error) if no config file is found.
func Load(startDir, configPath string) (*Config, error) {
	path := configPath
	if path == "" {
		path = findConfigFile(startDir)
	}
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, nil
}

// FindPackage returns the package with the given name, or an error if not found.
func (c *Config) FindPackage(name string) (*Package, error) {
	for i := range c.Packages {
		if c.Packages[i].Name == name {
			return &c.Packages[i], nil
		}
	}
	return nil, fmt.Errorf("package %q not found in config (available: %s)", name, c.packageNames())
}

func (c *Config) packageNames() string {
	names := ""
	for i, p := range c.Packages {
		if i > 0 {
			names += ", "
		}
		names += p.Name
	}
	return names
}

func findConfigFile(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}

	for {
		candidate := filepath.Join(dir, configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
