package notify

import (
	"fmt"
	"strings"
)

// Kind is what went wrong. Persisted, so these strings are stable.
type Kind string

const (
	// KindBackupFailed is a scheduled backup that did not complete.
	KindBackupFailed Kind = "backup-failed"
	// KindServerDown is a server that stopped answering.
	KindServerDown Kind = "server-down"
	// KindDiskLow is a disk close enough to full to stop things working.
	KindDiskLow Kind = "disk-low"
)

// Alert is what happened, in the terms the message is written in.
type Alert struct {
	Kind Kind
	// Server is the name a person gave the machine, not its id.
	Server string
	// Subject is what within the server it concerns — a tool, a mount point —
	// and is empty when the whole machine is the subject. It is display text:
	// "PostgreSQL", not "postgres".
	Subject string
	// Key identifies the condition for de-duplication, and is stable in a way
	// display text is not. Empty means Subject is used, which is right when
	// there is no subject at all.
	Key string
	// Detail is the specific thing that went wrong, already redacted.
	Detail string
	// Link is where in the dashboard to go, relative to its origin.
	Link string
}

// This is the one place alert wording lives, for the same reason apierr holds
// every error a person can see: copy that is written at the call site drifts
// into five voices, and these arrive when somebody is least able to interpret
// jargon.
//
// Every message follows the same shape — what happened, what it means, what to
// do — because a person reading it on a phone at 4am is scanning, not reading.
var wording = map[Kind]struct {
	subject func(Alert) string
	body    func(Alert) string
}{
	KindBackupFailed: {
		subject: func(a Alert) string {
			return fmt.Sprintf("Backup of %s did not finish", a.Subject)
		},
		body: func(a Alert) string {
			return join(
				fmt.Sprintf("The scheduled backup of %s on %s did not finish.", a.Subject, a.Server),
				"",
				"What this means: the last backup you have is the one before this "+
					"attempt. Nothing was lost and nothing was overwritten — the copy "+
					"simply was not made.",
				"",
				"What went wrong:",
				indent(a.Detail),
				"",
				"You can take one by hand at any time, and the next scheduled run "+
					"will try again on its own.",
			)
		},
	},
	KindServerDown: {
		subject: func(a Alert) string {
			return fmt.Sprintf("%s is not responding", a.Server)
		},
		body: func(a Alert) string {
			return join(
				fmt.Sprintf("%s stopped answering.", a.Server),
				"",
				"What this means: anything running on it is probably not reachable "+
					"either. It may be restarting, the network may be interrupted, or "+
					"the machine may be gone.",
				"",
				"What we saw:",
				indent(a.Detail),
				"",
				"We keep checking, and you will get one more message when it comes "+
					"back. If it does not, your hosting provider's console is the place "+
					"to look — the machine has to be running before anything here can "+
					"reach it.",
			)
		},
	},
	KindDiskLow: {
		subject: func(a Alert) string {
			return fmt.Sprintf("%s is running out of disk", a.Server)
		},
		body: func(a Alert) string {
			return join(
				fmt.Sprintf("The disk on %s is nearly full.", a.Server),
				"",
				indent(a.Detail),
				"",
				"What this means: when it fills completely, databases stop accepting "+
					"writes and deployments fail. This is worth acting on before then, "+
					"not after.",
				"",
				"The storage page shows what is using the space and offers to delete "+
					"the things that can go — old images, build leftovers, package files "+
					"— none of which are anything you put there.",
			)
		},
	},
}

// Cleared is the follow-up when a condition ends.
//
// Sent because a message that never gets a second half trains people to ignore
// the first. Knowing something recovered without being told is not possible.
func Cleared(alert Alert, origin string) Message {
	name := alert.Server
	if alert.Subject != "" {
		name = alert.Subject + " on " + alert.Server
	}

	return Message{
		Subject: fmt.Sprintf("Resolved: %s", strings.ToLower(subjectOf(alert))),
		Body: join(
			fmt.Sprintf("%s is back to normal. Nothing else is needed.", name),
			"",
			link(alert, origin),
		),
	}
}

// Render turns an alert into the message that gets sent.
func Render(alert Alert, origin string) Message {
	entry, known := wording[alert.Kind]
	if !known {
		// Unreachable through the UI, but a message with no words at all would
		// be worse than a plain one.
		return Message{
			Subject: fmt.Sprintf("Something needs attention on %s", alert.Server),
			Body:    join(alert.Detail, "", link(alert, origin)),
		}
	}

	return Message{
		Subject: entry.subject(alert),
		Body:    join(entry.body(alert), "", link(alert, origin)),
	}
}

func subjectOf(alert Alert) string {
	if entry, known := wording[alert.Kind]; known {
		return entry.subject(alert)
	}
	return "alert"
}

// link points at the page that can do something about it.
//
// Omitted rather than guessed when the dashboard's address is not known: a
// broken link in the one message that matters is worse than no link.
func link(alert Alert, origin string) string {
	if origin == "" || alert.Link == "" {
		return ""
	}
	return strings.TrimSuffix(origin, "/") + alert.Link
}

func join(lines ...string) string {
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func indent(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}
