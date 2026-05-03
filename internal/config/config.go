package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Filename is the canonical name of the vp config file.
const Filename = "vp.yaml"

// ErrNotFound is returned by FindUpwards (and Load) when no vp.yaml exists in
// startDir or any of its ancestors.
var ErrNotFound = errors.New("vp.yaml not found")

// Config is a parsed vp.yaml. After Load, all path-shaped fields
// (Plans.Dir, Plans.ArchiveDir, Components[*].Version.File) are absolute paths
// rooted at Dir. Components[*].Paths are kept as relative globs — they are
// matched against repo-relative file paths by future Affected logic.
type Config struct {
	Plans      PlansConfig          `yaml:"plans"`
	Components map[string]Component `yaml:"components"`

	// Dir is the directory that contained the loaded vp.yaml.
	Dir string `yaml:"-"`
}

// PlansConfig configures the plans directory and how plans are consumed.
type PlansConfig struct {
	Dir        string `yaml:"dir"`
	Consumed   string `yaml:"consumed"`
	ArchiveDir string `yaml:"archive_dir"`
}

// Component is one independently-versioned unit declared in vp.yaml.
type Component struct {
	Paths   []string      `yaml:"paths"`
	Version VersionTarget `yaml:"version"`
	Tag     string        `yaml:"tag"`
}

// VersionTarget points at the file owning a component's canonical version.
type VersionTarget struct {
	File   string `yaml:"file"`
	Format string `yaml:"format"`
	Path   string `yaml:"path"`
}

// FindUpwards searches startDir and its ancestors for a vp.yaml file, returning
// the absolute path to the first match. ErrNotFound is returned if no ancestor
// contains one.
func FindUpwards(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, Filename)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotFound
		}
		dir = parent
	}
}

// Load finds vp.yaml by searching upward from startDir, parses it, validates
// required fields, and resolves all relative paths against the directory that
// contained the file.
func Load(startDir string) (*Config, error) {
	path, err := FindUpwards(startDir)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.Dir = filepath.Dir(path)
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	resolvePaths(cfg)
	return cfg, nil
}

func resolvePaths(cfg *Config) {
	cfg.Plans.Dir = absJoin(cfg.Dir, cfg.Plans.Dir)
	cfg.Plans.ArchiveDir = absJoin(cfg.Dir, cfg.Plans.ArchiveDir)
	for name, comp := range cfg.Components {
		comp.Version.File = absJoin(cfg.Dir, comp.Version.File)
		cfg.Components[name] = comp
	}
}

func absJoin(base, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
