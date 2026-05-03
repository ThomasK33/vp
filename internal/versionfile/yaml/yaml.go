// Package yaml reads and surgically writes YAML version files using the
// validate-then-splice approach from ADR-0002: parse the document into a
// yaml.Node tree, walk the dotted path, and splice the leaf scalar's
// byte span — never round-tripping the document through yaml.Marshal.
// The rest of the file is preserved byte-for-byte: comments, anchors
// elsewhere, key order, indentation, and line endings.
package yaml

import (
	"fmt"
	"os"
	"slices"
	"strings"

	goyaml "go.yaml.in/yaml/v3"
)

// Read returns the version string at yamlPath (a dotted path like
// "version" or "package.version"). Errors if the path is missing or
// resolves to a non-string scalar.
func Read(file, yamlPath string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	_, _, value, err := locate(data, yamlPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	return value, nil
}

// Write replaces the string value at yamlPath with version. Only the
// inner value bytes are rewritten; surrounding quotes (if any) are kept
// and the rest of the file is preserved byte-for-byte. Errors before
// writing if the path is missing or resolves to a non-string scalar.
func Write(file, yamlPath, version string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	start, end, _, err := locate(data, yamlPath)
	if err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	out := make([]byte, 0, len(data)-(end-start)+len(version))
	out = append(out, data[:start]...)
	out = append(out, version...)
	out = append(out, data[end:]...)
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	if err := os.Rename(tmp, file); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write %s: %w", file, err)
	}
	return nil
}

// locate returns the byte span of the leaf's inner value (between the
// surrounding quotes for quoted styles) plus the decoded string value.
func locate(data []byte, yamlPath string) (start, end int, value string, err error) {
	if yamlPath == "" {
		return 0, 0, "", fmt.Errorf("empty path")
	}
	segments := strings.Split(yamlPath, ".")
	if slices.Contains(segments, "") {
		return 0, 0, "", fmt.Errorf("path %q: empty segment", yamlPath)
	}
	var root goyaml.Node
	if err := goyaml.Unmarshal(data, &root); err != nil {
		return 0, 0, "", fmt.Errorf("path %q: parse: %w", yamlPath, err)
	}
	if root.Kind != goyaml.DocumentNode || len(root.Content) == 0 {
		return 0, 0, "", fmt.Errorf("path %q: empty document", yamlPath)
	}
	node := root.Content[0]
	for _, seg := range segments {
		if node.Kind != goyaml.MappingNode {
			return 0, 0, "", fmt.Errorf("path %q: expected mapping at segment %q", yamlPath, seg)
		}
		var found *goyaml.Node
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == seg {
				found = node.Content[i+1]
				break
			}
		}
		if found == nil {
			return 0, 0, "", fmt.Errorf("path %q: not found", yamlPath)
		}
		node = found
	}
	if tag := node.ShortTag(); node.Kind != goyaml.ScalarNode || tag != "!!str" {
		return 0, 0, "", fmt.Errorf("path %q: value is not a string (got %s)", yamlPath, tag)
	}
	off := offsetOf(data, node.Line, node.Column)
	switch node.Style {
	case 0:
		return off, off + len(node.Value), node.Value, nil
	case goyaml.DoubleQuotedStyle:
		return off + 1, scanDoubleQuotedClose(data, off), node.Value, nil
	case goyaml.SingleQuotedStyle:
		return off + 1, scanSingleQuotedClose(data, off), node.Value, nil
	default:
		return 0, 0, "", fmt.Errorf("path %q: value must be a plain or quoted scalar", yamlPath)
	}
}

func offsetOf(data []byte, line, column int) int {
	off, curLine := 0, 1
	for off < len(data) && curLine < line {
		if data[off] == '\n' {
			curLine++
		}
		off++
	}
	return off + column - 1
}

func scanDoubleQuotedClose(data []byte, off int) int {
	for i := off + 1; ; i++ {
		switch data[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
}

func scanSingleQuotedClose(data []byte, off int) int {
	for i := off + 1; ; i++ {
		if data[i] != '\'' {
			continue
		}
		if i+1 < len(data) && data[i+1] == '\'' {
			i++
			continue
		}
		return i
	}
}
