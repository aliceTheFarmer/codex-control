package yolo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"codex-control/internal/logger"
)

// Mode represents the Codex command variant to run.
type Mode string

const (
	ModeDefault Mode = "default"
	ModeResume  Mode = "resume"
)

// Runner executes Codex with the bypass flags.
type Runner struct {
	Binary string
	Mode   Mode
	Log    *logger.Logger
}

// Result describes the proxied command run.
type Result struct {
	Command  []string `json:"command"`
	ExitCode int      `json:"exit_code"`
}

// Run executes the Codex command and returns its exit status.
func (r Runner) Run(ctx context.Context, args []string) (Result, error) {
	binary := r.Binary
	if binary == "" {
		binary = "codex"
	}
	cmdArgs := r.buildArgs(args)
	if r.Log != nil {
		r.Log.Printf(logger.PrefixCodex, "Executing %s", strings.Join(append([]string{binary}, cmdArgs...), " "))
	}
	cmd := exec.CommandContext(ctx, binary, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	exit := exitCode(err)
	return Result{Command: append([]string{binary}, cmdArgs...), ExitCode: exit}, err
}

func (r Runner) buildArgs(args []string) []string {
	base := []string{"--dangerously-bypass-approvals-and-sandbox"}
	if r.Mode == ModeResume {
		base = append([]string{"resume"}, base...)
	}
	return append(base, args...)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}
