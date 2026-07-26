package steps

import "context"

// Outcome is what happened to one step in a run.
type Outcome string

const (
	// OutcomeChanged means Apply ran and altered the machine.
	OutcomeChanged Outcome = "changed"
	// OutcomeUnchanged means Check found the work already done. Re-running a
	// playbook should produce these and nothing else — that is what makes it
	// safe to run repeatedly.
	OutcomeUnchanged Outcome = "unchanged"
	// OutcomeSkipped means a precondition said this step does not apply here.
	OutcomeSkipped Outcome = "skipped"
	// OutcomeWouldChange is only produced by a dry run: Check said the work is
	// outstanding, and Apply was deliberately not called.
	OutcomeWouldChange Outcome = "would-change"
	OutcomeFailed      Outcome = "failed"
)

// Step is one unit of convergence. Check asks "is this already true?"; Apply
// makes it true. Every step must be safe to run twice.
type Step interface {
	ID() string
	// Title is shown in the UI, so it is plain language rather than a command.
	Title() string
	// SkipReason returns a non-empty string when the step does not apply to
	// this machine, and is consulted before Check.
	SkipReason(ctx context.Context, s *Session) (string, error)
	Check(ctx context.Context, s *Session) (done bool, err error)
	Apply(ctx context.Context, s *Session) error
}

// shellStep expresses a step as shell: one command that tests for the desired
// state, and a list that establishes it. Almost every baseline step fits this,
// which keeps the playbook declarative and easy to audit.
type shellStep struct {
	id    string
	title string

	// skipIf, when set and exiting zero, skips the step with skipMessage.
	skipIf      string
	skipMessage string

	// check exits zero when the desired state already holds.
	check string

	// apply runs in order; the first non-zero exit fails the step.
	apply []string
}

func (s shellStep) ID() string    { return s.id }
func (s shellStep) Title() string { return s.title }

func (s shellStep) SkipReason(ctx context.Context, sess *Session) (string, error) {
	if s.skipIf == "" {
		return "", nil
	}
	skip, err := sess.Test(ctx, s.skipIf)
	if err != nil {
		return "", err
	}
	if skip {
		return s.skipMessage, nil
	}
	return "", nil
}

func (s shellStep) Check(ctx context.Context, sess *Session) (bool, error) {
	if s.check == "" {
		return false, nil
	}
	return sess.Test(ctx, s.check)
}

func (s shellStep) Apply(ctx context.Context, sess *Session) error {
	for _, cmd := range s.apply {
		result, err := sess.Run(ctx, cmd)
		if err != nil {
			return err
		}
		if !result.OK() {
			return &CommandError{Command: cmd, ExitCode: result.ExitCode, Stderr: result.Stderr}
		}
	}
	return nil
}

type CommandError struct {
	Command  string
	ExitCode int
	Stderr   string
}

func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	return "command failed with exit code " + itoa(e.ExitCode)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
