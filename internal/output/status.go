package output

import (
	"encoding/json"
	"io"
)

// StatusReport is the JSON shape emitted by `vp status --json`.
type StatusReport struct {
	Pending  []PendingPlan       `json:"pending"`
	Resolved []ResolvedComponent `json:"resolved"`
}

// PendingPlan describes one plan file currently sitting in the plans dir.
type PendingPlan struct {
	File     string            `json:"file"`
	Releases map[string]string `json:"releases"`
	Message  string            `json:"message,omitempty"`
}

// ResolvedComponent is one component's collapsed bump and (when readable)
// the current and next version strings.
type ResolvedComponent struct {
	Component string `json:"component"`
	Bump      string `json:"bump"`
	Current   string `json:"current,omitempty"`
	Next      string `json:"next,omitempty"`
}

// WriteStatus encodes r to w as pretty-printed JSON with a trailing newline.
// Nil slices in r are written as empty arrays so consumers see a stable shape.
func WriteStatus(w io.Writer, r *StatusReport) error {
	if r.Pending == nil {
		r.Pending = []PendingPlan{}
	}
	if r.Resolved == nil {
		r.Resolved = []ResolvedComponent{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
