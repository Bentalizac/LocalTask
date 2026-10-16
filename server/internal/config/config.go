// Package config loads the central YAML file that tells the taskboard
// application which local project directories to serve.
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"localtask/internal/storage"
)

// Storage says where a project's taskboard.json lives.
type Storage string

const (
	// StorageExternal keeps taskboard.json outside the project directory,
	// under BoardsDirName next to the config file. LocalTask then writes
	// nothing into the project's repo. This is the default.
	StorageExternal Storage = "external"
	// StorageRepo keeps taskboard.json at the project's root, so it can be
	// committed alongside the code.
	StorageRepo Storage = "repo"
)

// BoardsDirName is the directory, next to the config file, that holds one
// subdirectory per externally stored project board.
const BoardsDirName = "boards"

// ProjectConfig points the application at one local project directory.
type ProjectConfig struct {
	Name    string  `yaml:"name"`
	Path    string  `yaml:"path"`
	Storage Storage `yaml:"storage,omitempty"`
	// JiraBaseURL overrides Config.JiraBaseURL for this project.
	JiraBaseURL string `yaml:"jira_base_url,omitempty"`
}

// EffectiveStorage returns the project's storage mode, applying the
// external default when none is set.
func (pc ProjectConfig) EffectiveStorage() Storage {
	if pc.Storage == "" {
		return StorageExternal
	}
	return pc.Storage
}

// Config is the top-level shape of the taskboard config file.
type Config struct {
	// JiraBaseURL is the Jira site that tickets' Jira keys link to, e.g.
	// "https://example.atlassian.net". Optional.
	JiraBaseURL string          `yaml:"jira_base_url,omitempty"`
	Projects    []ProjectConfig `yaml:"projects"`

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

// Path returns the file the config was loaded from (or will be saved to).
// It is empty for a Config built without one.
func (c *Config) Path() string {
	return c.path
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
	if err := pc.Validate(); err != nil {
		return err
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
	if err := validateJiraBaseURL(c.JiraBaseURL); err != nil {
		issues = append(issues, err.Error())
	}
	seen := make(map[string]bool, len(c.Projects))
	for i, p := range c.Projects {
		if p.Name == "" {
			issues = append(issues, fmt.Sprintf("projects[%d]: missing name", i))
		}
		if p.Path == "" {
			issues = append(issues, fmt.Sprintf("projects[%d]: missing path", i))
		}
		if err := p.Validate(); err != nil {
			issues = append(issues, fmt.Sprintf("projects[%d]: %v", i, err))
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

// Validate checks pc's fields beyond presence: the storage mode, and that
// the name is usable as a directory name for an external board.
func (pc ProjectConfig) Validate() error {
	switch pc.Storage {
	case "", StorageExternal, StorageRepo:
	default:
		return fmt.Errorf("invalid storage %q (want %q or %q)", pc.Storage, StorageExternal, StorageRepo)
	}
	if pc.Name == "." || pc.Name == ".." || strings.ContainsAny(pc.Name, `/\`) {
		return fmt.Errorf("invalid project name %q: must not be \".\", \"..\", or contain a path separator", pc.Name)
	}
	return validateJiraBaseURL(pc.JiraBaseURL)
}

// validateJiraBaseURL accepts an empty value or an absolute http(s) URL.
func validateJiraBaseURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("invalid jira_base_url %q: want an absolute http(s) URL like https://example.atlassian.net", raw)
	}
	return nil
}

// JiraBaseURLFor returns the Jira site pc's tickets link to: its own
// jira_base_url, else the top-level one, without a trailing slash.
func (c *Config) JiraBaseURLFor(pc ProjectConfig) string {
	base := pc.JiraBaseURL
	if base == "" {
		base = c.JiraBaseURL
	}
	return strings.TrimRight(base, "/")
}

// BoardDir returns the directory holding pc's taskboard.json: the project
// root for StorageRepo, or a per-project directory under BoardsDirName next
// to the config file for StorageExternal.
func (c *Config) BoardDir(pc ProjectConfig) (string, error) {
	if pc.EffectiveStorage() == StorageRepo {
		return pc.Path, nil
	}
	if c.path == "" {
		return "", fmt.Errorf("project %q: external storage needs a config file location to store boards next to", pc.Name)
	}
	return filepath.Join(filepath.Dir(c.path), BoardsDirName, pc.Name), nil
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
