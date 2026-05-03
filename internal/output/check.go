package output

import (
	"encoding/json"
	"io"
)

// CheckReport is the JSON shape emitted by `vp check --json`.
type CheckReport struct {
	Affected []string `json:"affected"`
	Planned  []string `json:"planned"`
	Missing  []string `json:"missing"`
}

// WriteCheck encodes r to w as pretty-printed JSON with a trailing newline.
// Nil slices in r are written as empty arrays so consumers see a stable shape.
func WriteCheck(w io.Writer, r *CheckReport) error {
	r.Affected = nilToEmpty(r.Affected)
	r.Planned = nilToEmpty(r.Planned)
	r.Missing = nilToEmpty(r.Missing)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func nilToEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
