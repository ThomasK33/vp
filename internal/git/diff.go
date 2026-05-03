package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ChangedFiles returns the file paths changed between base and head as
// reported by `git diff --name-only --relative base...head` run from dir.
// Paths returned are relative to dir. (nil, nil) means no files changed.
func ChangedFiles(dir, base, head string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "diff", "--name-only", "--relative", base+"..."+head)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, fmt.Errorf("git diff %s...%s: %w", base, head, err)
		}
		return nil, fmt.Errorf("git diff %s...%s: %w: %s", base, head, err, msg)
	}
	var out []string
	for line := range strings.SplitSeq(stdout.String(), "\n") {
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}
