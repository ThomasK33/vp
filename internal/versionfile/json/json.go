// Package json reads and surgically writes JSON version files using the
// validate-then-splice approach from ADR-0002: walk the dotted path with
// encoding/json, capture the leaf string value's byte span, and splice
// only those bytes — never round-tripping the document. The rest of the
// file is preserved byte-for-byte: key order, indentation, line endings,
// and trailing whitespace.
package json

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Read returns the version string at jsonPath (a dotted path like
// "version" or "package.version"). Errors if the path is missing or
// resolves to a non-string value.
func Read(file, jsonPath string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	_, _, value, err := locate(data, jsonPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", file, err)
	}
	return value, nil
}

// Write replaces the string value at jsonPath with version. Only the
// value bytes (including the surrounding quotes) are rewritten; the
// rest of the file is preserved byte-for-byte. Errors before writing
// if the path is missing or resolves to a non-string value.
func Write(file, jsonPath, version string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	start, end, _, err := locate(data, jsonPath)
	if err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	encoded, err := stdjson.Marshal(version)
	if err != nil {
		return fmt.Errorf("write %s: encode value: %w", file, err)
	}
	out := make([]byte, 0, len(data)-(end-start)+len(encoded))
	out = append(out, data[:start]...)
	out = append(out, encoded...)
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

func locate(data []byte, jsonPath string) (start, end int, value string, err error) {
	if jsonPath == "" {
		return 0, 0, "", fmt.Errorf("empty path")
	}
	segments := strings.Split(jsonPath, ".")
	if slices.Contains(segments, "") {
		return 0, 0, "", fmt.Errorf("path %q: empty segment", jsonPath)
	}
	dec := stdjson.NewDecoder(bytes.NewReader(data))
	return walk(dec, data, jsonPath, segments)
}

func walk(dec *stdjson.Decoder, data []byte, jsonPath string, segments []string) (start, end int, value string, err error) {
	tok, err := dec.Token()
	if err != nil {
		return 0, 0, "", fmt.Errorf("path %q: parse: %w", jsonPath, err)
	}
	d, ok := tok.(stdjson.Delim)
	if !ok || d != '{' {
		return 0, 0, "", fmt.Errorf("path %q: expected object, got %v", jsonPath, tok)
	}
	want := segments[0]
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return 0, 0, "", fmt.Errorf("path %q: parse: %w", jsonPath, err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return 0, 0, "", fmt.Errorf("path %q: expected string key, got %T", jsonPath, keyTok)
		}
		if key != want {
			if err := skipValue(dec); err != nil {
				return 0, 0, "", fmt.Errorf("path %q: parse: %w", jsonPath, err)
			}
			continue
		}
		if len(segments) > 1 {
			return walk(dec, data, jsonPath, segments[1:])
		}
		pre := dec.InputOffset()
		valTok, err := dec.Token()
		if err != nil {
			return 0, 0, "", fmt.Errorf("path %q: parse: %w", jsonPath, err)
		}
		s, ok := valTok.(string)
		if !ok {
			return 0, 0, "", fmt.Errorf("path %q: value is not a string (got %T)", jsonPath, valTok)
		}
		post := dec.InputOffset()
		valueStart := scanToValue(data, int(pre))
		return valueStart, int(post), s, nil
	}
	return 0, 0, "", fmt.Errorf("path %q: not found", jsonPath)
}

func skipValue(dec *stdjson.Decoder) error {
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(stdjson.Delim); ok {
			switch d {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
		if depth == 0 {
			return nil
		}
	}
}

func scanToValue(data []byte, from int) int {
	for i := from; i < len(data); i++ {
		switch data[i] {
		case ' ', '\t', '\n', '\r', ':':
			continue
		}
		return i
	}
	return from
}
