package output

import (
	"encoding/json"
	"io"
)

// ApplyReport is the JSON shape emitted by `vp apply --json` and
// `vp apply --dry-run --json`. The two modes share a single shape;
// dry-run emits Consumed as an empty array.
type ApplyReport struct {
	Changes  []Change `json:"changes"`
	Consumed []string `json:"consumed"`
}

// Change is one component's planned or applied version-file write.
type Change struct {
	Component string `json:"component"`
	Current   string `json:"current"`
	Next      string `json:"next"`
	Bump      string `json:"bump"`
	File      string `json:"file"`
	Tag       string `json:"tag,omitempty"`
}

// WriteApply encodes r to w as pretty-printed JSON with a trailing newline.
// Nil slices in r are written as empty arrays so consumers see a stable shape.
func WriteApply(w io.Writer, r *ApplyReport) error {
	if r.Changes == nil {
		r.Changes = []Change{}
	}
	if r.Consumed == nil {
		r.Consumed = []string{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
