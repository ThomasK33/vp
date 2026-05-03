// Package text reads and writes whole-file plain-text version files.
//
// The version is the file body with surrounding whitespace stripped. Write
// preserves the trailing-newline convention of the existing file: if the
// previous content ended with a newline, the rewrite ends with one too.
package text

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// Read returns the version string at path. Surrounding whitespace
// (including any trailing newline) is stripped.
func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "", fmt.Errorf("read %s: empty version file", path)
	}
	return v, nil
}

// Write replaces the contents of path with version, preserving whether the
// original file ended with a newline.
func Write(path, version string) error {
	prev, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	body := version
	if bytes.HasSuffix(prev, []byte("\n")) {
		body += "\n"
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
