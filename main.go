// fzfweb is a drop-in replacement for the fzf binary. Non-interactive calls
// (anything passing -f/--filter) are forwarded to the real fzf unchanged.
// Interactive calls are instead served as a local web page of checkboxes,
// with the real fzf still doing the matching for every query typed there.
//
// It is deliberately not specific to any one caller: it knows nothing about
// mach, tryselect, or Firefox. It just impersonates fzf.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]

	// Anything other than a genuine interactive selection call passes
	// through untouched. -f/--filter is the non-interactive filter mode
	// tryselect uses for -q/--query. --version and --help never provide
	// a candidate list on stdin, so treating them as interactive would
	// hang forever waiting for input that isn't coming.
	hasFilter := false
	passthroughOnly := false
	exact := false
	for _, a := range args {
		switch {
		case a == "-f" || a == "--filter":
			hasFilter = true
		case strings.HasPrefix(a, "--filter="):
			hasFilter = true
		case a == "--version" || a == "--help" || a == "-h":
			passthroughOnly = true
		case a == "--exact":
			exact = true
		}
	}

	realFzf, err := resolveRealFzf()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fzfweb: "+err.Error())
		os.Exit(1)
	}

	if hasFilter || passthroughOnly {
		os.Exit(passthrough(realFzf, args))
	}

	os.Exit(runWeb(realFzf, exact))
}

// resolveRealFzf finds the fzf binary this shim delegates all real matching
// to. REAL_FZF overrides the default location fzf_bootstrap() installs to.
func resolveRealFzf() (string, error) {
	real := os.Getenv("REAL_FZF")
	if real == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		real = filepath.Join(home, ".mozbuild", "fzf", "fzf.exe")
	}

	abs, err := filepath.Abs(real)
	if err != nil {
		return "", fmt.Errorf("could not resolve %q: %w", real, err)
	}

	if self, err := os.Executable(); err == nil {
		if selfAbs, err := filepath.Abs(self); err == nil && strings.EqualFold(selfAbs, abs) {
			return "", fmt.Errorf(
				"REAL_FZF resolves to this shim itself (%s); refusing to recurse", abs,
			)
		}
	}

	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf(
			"real fzf not found at %s (set REAL_FZF to override): %w", abs, err,
		)
	}

	return abs, nil
}

// passthrough hands the call straight to real fzf, unchanged, and returns
// its exit code.
func passthrough(realFzf string, args []string) int {
	cmd := exec.Command(realFzf, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err == nil {
		return 0
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}

	fmt.Fprintln(os.Stderr, "fzfweb: "+err.Error())
	return 1
}
