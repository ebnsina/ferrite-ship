package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/executor"
)

// Level classifies a log line for the UI.
type Level string

const (
	LevelCommand Level = "command"
	LevelOutput  Level = "output"
	LevelInfo    Level = "info"
	LevelChanged Level = "changed"
	LevelSkipped Level = "skipped"
	LevelError   Level = "error"
)

// LogFunc receives every line a run produces. It must not block for long —
// the runner calls it inline while the playbook executes.
type LogFunc func(level Level, message string)

// Session is what steps are handed: a way to run commands on the target and a
// way to say what they are doing.
type Session struct {
	exec executor.Executor
	log  LogFunc
}

func NewSession(exec executor.Executor, log LogFunc) *Session {
	if log == nil {
		log = func(Level, string) {}
	}
	return &Session{exec: exec, log: log}
}

func (s *Session) Log(level Level, message string) { s.log(level, message) }

func (s *Session) Logf(level Level, format string, args ...any) {
	s.log(level, fmt.Sprintf(format, args...))
}

// Run executes cmd, streaming the command and its output into the job log.
func (s *Session) Run(ctx context.Context, cmd string) (executor.Result, error) {
	s.log(LevelCommand, cmd)

	result, err := s.exec.Run(ctx, cmd)
	if err != nil {
		s.log(LevelError, err.Error())
		return result, err
	}

	for _, line := range splitLines(result.Stdout) {
		s.log(LevelOutput, line)
	}
	for _, line := range splitLines(result.Stderr) {
		s.log(LevelOutput, line)
	}

	return result, nil
}

// Test reports whether cmd exited zero. Used by Check, where a non-zero exit
// is the expected way of saying "not done yet".
func (s *Session) Test(ctx context.Context, cmd string) (bool, error) {
	result, err := s.exec.Run(ctx, cmd)
	if err != nil {
		return false, err
	}
	return result.OK(), nil
}

// Capture runs cmd and returns its trimmed stdout, failing if it exits non-zero.
func (s *Session) Capture(ctx context.Context, cmd string) (string, error) {
	result, err := s.exec.Run(ctx, cmd)
	if err != nil {
		return "", err
	}
	if !result.OK() {
		return "", fmt.Errorf("command %q exited %d: %s", cmd, result.ExitCode, result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
