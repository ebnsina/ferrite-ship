package steps

import (
	"context"
	"errors"
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

// Privilege is how a session gets the rights the playbook needs.
type Privilege int

const (
	// PrivilegeRoot means commands run as-is.
	PrivilegeRoot Privilege = iota
	// PrivilegeSudo wraps each command, because the login account is not root.
	PrivilegeSudo
)

// Session is what steps are handed: a way to run commands on the target and a
// way to say what they are doing.
type Session struct {
	exec      executor.Executor
	log       LogFunc
	privilege Privilege
	redact    *strings.Replacer
}

func NewSession(exec executor.Executor, log LogFunc) *Session {
	if log == nil {
		log = func(Level, string) {}
	}
	return &Session{exec: exec, log: log}
}

// WithPrivilege returns a session that elevates commands when needed.
func (s *Session) WithPrivilege(privilege Privilege) *Session {
	clone := *s
	clone.privilege = privilege
	return &clone
}

// Mask is what a redacted secret is replaced with in the job log.
const Mask = "••••••••"

// WithSecrets returns a session that masks these values wherever they appear
// in a logged command or its output.
//
// Installing a database means writing a generated password to a file, and the
// command that does it is echoed into the job log — which is persisted and
// shown in the browser. Redacting at this boundary covers every step at once,
// rather than asking each one to remember. Note that this only affects what is
// *logged*: the command still runs with the real value.
func (s *Session) WithSecrets(values ...string) *Session {
	pairs := make([]string, 0, len(values)*2)
	for _, value := range values {
		// A short secret would mask half the log, and an empty one makes
		// Replacer insert the mask between every character.
		if len(value) < 8 {
			continue
		}
		pairs = append(pairs, value, Mask)
	}

	clone := *s
	if len(pairs) > 0 {
		clone.redact = strings.NewReplacer(pairs...)
	}
	return &clone
}

func (s *Session) safe(text string) string {
	if s.redact == nil {
		return text
	}
	return s.redact.Replace(text)
}

// elevate wraps a command so it runs with the rights the playbook needs.
//
// The whole command goes through `sh -c` rather than being prefixed directly:
// steps use pipes, redirects and `&&`, and a bare `sudo cmd | grep` would
// elevate only the first process in the pipeline.
func (s *Session) elevate(cmd string) string {
	if s.privilege == PrivilegeRoot {
		return cmd
	}
	return "sudo -n sh -c " + shellQuote(cmd)
}

// DetectPrivilege works out whether the login account can do the job.
//
// Failing here with a clear message is much kinder than letting every step
// fail one by one with "permission denied".
func DetectPrivilege(ctx context.Context, exec executor.Executor) (Privilege, error) {
	whoami, err := exec.Run(ctx, "whoami")
	if err != nil {
		return PrivilegeRoot, err
	}
	if strings.TrimSpace(whoami.Stdout) == "root" {
		return PrivilegeRoot, nil
	}

	// -n means "never prompt": if a password would be required, fail now
	// rather than hang waiting on a prompt nobody can answer.
	sudo, err := exec.Run(ctx, "sudo -n true")
	if err != nil {
		return PrivilegeRoot, err
	}
	if sudo.OK() {
		return PrivilegeSudo, nil
	}

	return PrivilegeRoot, errors.New(
		"this account cannot make changes: sign in as root, or use an account " +
			"that can run sudo without being asked for a password")
}

func (s *Session) Log(level Level, message string) { s.log(level, s.safe(message)) }

func (s *Session) Logf(level Level, format string, args ...any) {
	s.Log(level, fmt.Sprintf(format, args...))
}

// Redact masks any known secret in text.
//
// Exported because not every failure leaves through the log: a job's stored
// error is written straight to the database, and it must get the same
// treatment as the lines around it.
func (s *Session) Redact(text string) string { return s.safe(text) }

// Run executes cmd, streaming the command and its output into the job log.
func (s *Session) Run(ctx context.Context, cmd string) (executor.Result, error) {
	s.Log(LevelCommand, cmd)

	result, err := s.exec.Run(ctx, s.elevate(cmd))
	if err != nil {
		s.Log(LevelError, err.Error())
		return result, err
	}

	// Output is redacted as well as the command: plenty of tools echo their own
	// configuration back, and `docker compose config` prints it in full.
	for _, line := range splitLines(result.Stdout) {
		s.Log(LevelOutput, line)
	}
	for _, line := range splitLines(result.Stderr) {
		s.Log(LevelOutput, line)
	}

	return result, nil
}

// Test reports whether cmd exited zero. Used by Check, where a non-zero exit
// is the expected way of saying "not done yet".
func (s *Session) Test(ctx context.Context, cmd string) (bool, error) {
	result, err := s.exec.Run(ctx, s.elevate(cmd))
	if err != nil {
		return false, err
	}
	return result.OK(), nil
}

// Capture runs cmd and returns its trimmed stdout, failing if it exits non-zero.
func (s *Session) Capture(ctx context.Context, cmd string) (string, error) {
	result, err := s.exec.Run(ctx, s.elevate(cmd))
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
