// Command vp stages semver bump intent in Git.
package main

import (
	"os"

	"github.com/ThomasK33/vp/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
