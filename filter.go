package main

import (
	"os/exec"
	"strings"
)

// filter runs real fzf in its non-interactive filter mode against items and
// returns the matches in fzf's relevance order. An empty or whitespace-only
// query is not sent to fzf at all; it matches everything, same as fzf itself
// shows for an empty query box.
func filter(realFzf string, items []string, query string, exact bool) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return items, nil
	}

	args := []string{"-f", query}
	if exact {
		args = append(args, "--exact")
	}

	cmd := exec.Command(realFzf, args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))

	out, err := cmd.Output()
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		// Exit status 1 means the query matched nothing; not an error.
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	text := strings.TrimRight(string(out), "\n")
	if text == "" {
		return []string{}, nil
	}
	return strings.Split(text, "\n"), nil
}
