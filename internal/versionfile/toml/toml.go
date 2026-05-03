// Package toml reads and surgically writes TOML version files using the
// validate-then-splice approach from ADR-0002: parse with the
// pelletier/go-toml/v2/unstable position-aware parser, walk top-level
// expressions to the target dotted key path, and splice only the leaf
// string's inner byte span — never round-tripping the document. The rest
// of the file is preserved byte-for-byte: comments, key order, table
// headers, and surrounding whitespace.
package toml

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2/unstable"
)

// Read returns the version string at tomlPath (a dotted path like
// "version" or "package.version"). Errors if the path is missing or
// resolves to a non-string value.
func Read(file, tomlPath string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	_, _, value, err := locate(data, tomlPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	return value, nil
}

// Write replaces the string value at tomlPath with version. Only the
// inner value bytes are rewritten; surrounding quotes are kept and the
// rest of the file is preserved byte-for-byte. Errors before writing if
// the path is missing or resolves to anything other than a basic or
// literal single-line string.
func Write(file, tomlPath, version string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	start, end, _, err := locate(data, tomlPath)
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

func locate(data []byte, tomlPath string) (start, end int, value string, err error) {
	if tomlPath == "" {
		return 0, 0, "", fmt.Errorf("empty path")
	}
	targets := strings.Split(tomlPath, ".")
	if slices.Contains(targets, "") {
		return 0, 0, "", fmt.Errorf("path %q: empty segment", tomlPath)
	}
	var p unstable.Parser
	p.Reset(data)
	var currentTable []string
	inArrayTable := false
	for p.NextExpression() {
		e := p.Expression()
		switch e.Kind {
		case unstable.Table:
			currentTable = keySegments(e)
			inArrayTable = false
			continue
		case unstable.ArrayTable:
			inArrayTable = true
			continue
		case unstable.KeyValue:
			if inArrayTable {
				continue
			}
		default:
			continue
		}
		segs := append(append([]string{}, currentTable...), keySegments(e)...)
		if !slices.Equal(segs, targets) {
			if isPrefix(segs, targets) {
				return 0, 0, "", fmt.Errorf("path %q: expected table at segment %q", tomlPath, targets[len(segs)])
			}
			continue
		}
		v := e.Value()
		if v.Kind != unstable.String {
			return 0, 0, "", fmt.Errorf("path %q: value is not a string (got %s)", tomlPath, v.Kind)
		}
		tokenStart := int(v.Raw.Offset)
		tokenEnd := tokenStart + int(v.Raw.Length)
		token := data[tokenStart:tokenEnd]
		if len(token) >= 6 && token[0] == token[1] && token[1] == token[2] {
			return 0, 0, "", fmt.Errorf("path %q: value must be a basic or literal string", tomlPath)
		}
		return tokenStart + 1, tokenEnd - 1, string(token[1 : len(token)-1]), nil
	}
	if perr := p.Error(); perr != nil {
		return 0, 0, "", fmt.Errorf("path %q: parse: %w", tomlPath, perr)
	}
	return 0, 0, "", fmt.Errorf("path %q: not found", tomlPath)
}

func isPrefix(prefix, full []string) bool {
	return len(prefix) < len(full) && slices.Equal(prefix, full[:len(prefix)])
}

func keySegments(n *unstable.Node) []string {
	var segs []string
	it := n.Key()
	for it.Next() {
		segs = append(segs, string(it.Node().Data))
	}
	return segs
}
