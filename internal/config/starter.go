package config

import _ "embed"

//go:embed starter.yaml
var starter []byte

// Starter returns the bytes of the starter vp.yaml that `vp init` writes.
func Starter() []byte { return starter }
