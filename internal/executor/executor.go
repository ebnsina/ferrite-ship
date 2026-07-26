// Package executor abstracts "run a shell command on a target machine".
//
// Steps are written against this interface, so the same playbook runs over SSH
// today and over the agent later without a single step changing.
package executor

import "context"

// Result is the outcome of one command. A non-zero ExitCode is not an error:
// steps use exit codes to decide whether work is already done.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// OK reports whether the command completed successfully.
func (r Result) OK() bool { return r.ExitCode == 0 }

type Executor interface {
	// Run executes cmd and returns its result. It returns a non-nil error only
	// for transport failures — a command that runs and exits non-zero is a
	// successful Run with a non-zero ExitCode.
	Run(ctx context.Context, cmd string) (Result, error)

	// Describe names the target, for logs.
	Describe() string

	Close() error
}
