package plan

import (
	"bytes"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Plan is the in-memory representation of a .version-plans/*.yaml file.
type Plan struct {
	Releases map[string]string `yaml:"releases"`
	Message  string            `yaml:"message,omitempty"`
}

// Bump levels.
const (
	BumpMajor = "major"
	BumpMinor = "minor"
	BumpPatch = "patch"
	BumpNone  = "none"
)

func validBump(b string) bool {
	switch b {
	case BumpMajor, BumpMinor, BumpPatch, BumpNone:
		return true
	}
	return false
}

// New constructs and validates a Plan from the given releases and message.
func New(releases map[string]string, message string) (*Plan, error) {
	if len(releases) == 0 {
		return nil, fmt.Errorf("releases: at least one component bump is required")
	}
	for name, bump := range releases {
		if name == "" {
			return nil, fmt.Errorf("releases: component name must be non-empty")
		}
		if bump == "" {
			return nil, fmt.Errorf("releases.%s: bump must be non-empty", name)
		}
		if !validBump(bump) {
			return nil, fmt.Errorf("releases.%s: %q is not one of [%s, %s, %s, %s]",
				name, bump, BumpMajor, BumpMinor, BumpPatch, BumpNone)
		}
	}
	return &Plan{Releases: releases, Message: message}, nil
}

// Save writes p to path as YAML with 2-space indent. It refuses to overwrite
// an existing file. Callers must ensure the parent directory exists.
func Save(p *Plan, path string) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(p); err != nil {
		return fmt.Errorf("encode plan: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode plan: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// Load parses a plan file at path.
func Load(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	p := &Plan{}
	if err := yaml.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return p, nil
}
