package steps

import "fmt"

// Summary counts what a playbook run did. The number worth watching is
// Changed: on a second run of the same playbook it should be zero.
type Summary struct {
	Changed     int
	Unchanged   int
	Skipped     int
	WouldChange int
	Failed      int
	FirstError  error
	// DryRun records that nothing was actually applied.
	DryRun bool
}

func (s Summary) Total() int {
	return s.Changed + s.Unchanged + s.Skipped + s.WouldChange + s.Failed
}

// Describe renders the summary in plain language for the activity feed.
func (s Summary) Describe() string {
	if s.Failed > 0 {
		return fmt.Sprintf("%d of %d steps could not be completed", s.Failed, s.Total())
	}

	if s.DryRun {
		if s.WouldChange == 0 {
			return fmt.Sprintf("Nothing would change — all %d checks already pass", s.Total())
		}
		return fmt.Sprintf("%d of %d would change. Nothing was altered.",
			s.WouldChange, s.Total())
	}

	if s.Changed == 0 {
		return fmt.Sprintf("Nothing needed changing — all %d checks already passed", s.Total())
	}
	return fmt.Sprintf("%d changed, %d already fine, %d not needed",
		s.Changed, s.Unchanged, s.Skipped)
}
