// Package config loads the central YAML file that tells the taskboard
// application which local project directories to serve.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"localtask/internal/storage"
)

// ProjectConfig points the application at one local project directory.
type ProjectConfig struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// Config is the top-level shape of the taskboard config file.
type Config struct {
	Projects []ProjectConfig `yaml:"projects"`

	path string // set by Load; used by Save/AddProject
}

// New returns an empty Config for when no config file exists yet at path
// (e.g. a fresh install with no projects configured), but Save/AddProject
// should still write one there once a project is added.
func New(path string) *Config {
	return &Config{path: path}
}

// Load reads and validates a config file from path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.path = path
	return &cfg, nil
}

// Save writes the config back to the file it was loaded from, atomically.
func (c *Config) Save() error {
	if c.path == "" {
		return fmt.Errorf("config has no source path to save to")
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(c.path, data, 0o644)
}

// AddProject appends pc and persists the config, so a project added at
// runtime (see the HTTP API's add-project endpoint) survives a restart. On
// failure the in-memory config is left unchanged.
func (c *Config) AddProject(pc ProjectConfig) error {
	if pc.Name == "" || pc.Path == "" {
		return fmt.Errorf("project name and path are required")
	}
	for _, existing := range c.Projects {
		if existing.Name == pc.Name {
			return fmt.Errorf("project %q already configured", pc.Name)
		}
	}
	c.Projects = append(c.Projects, pc)
	if err := c.Save(); err != nil {
		c.Projects = c.Projects[:len(c.Projects)-1]
		return err
	}
	return nil
}

// Validate checks that every configured project has a name and path, and
// that names are unique.
func (c *Config) Validate() error {
	var issues []string
	seen := make(map[string]bool, len(c.Projects))
	for i, p := range c.Projects {
		if p.Name == "" {
			issues = append(issues, fmt.Sprintf("projects[%d]: missing name", i))
		}
		if p.Path == "" {
			issues = append(issues, fmt.Sprintf("projects[%d]: missing path", i))
		}
		if p.Name != "" && seen[p.Name] {
			issues = append(issues, fmt.Sprintf("duplicate project name %q", p.Name))
		}
		seen[p.Name] = true
	}
	if len(issues) > 0 {
		return fmt.Errorf("config validation failed:\n- %s", strings.Join(issues, "\n- "))
	}
	return nil
}

// FindProject returns the configured project with the given name.
func (c *Config) FindProject(name string) (*ProjectConfig, error) {
	for i := range c.Projects {
		if c.Projects[i].Name == name {
			return &c.Projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %q not configured", name)
}
