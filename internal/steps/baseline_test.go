package steps

import (
	"strings"
	"testing"
)

func TestShellQuoteNeutralisesInjection(t *testing.T) {
	// A public key, timezone or user name reaches these commands from the API.
	// If quoting fails, the value escapes into the shell as code.
	cases := []struct {
		name  string
		input string
	}{
		{"plain", "ssh-ed25519 AAAAC3Nza key comment"},
		{"single quote", "it's fine"},
		{"command substitution", "$(rm -rf /)"},
		{"backticks", "`id`"},
		{"semicolon", "UTC; rm -rf /"},
		{"quote break-out", `'; rm -rf / ; echo '`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			quoted := shellQuote(tc.input)

			if !strings.HasPrefix(quoted, "'") || !strings.HasSuffix(quoted, "'") {
				t.Fatalf("shellQuote(%q) = %q, want it wrapped in single quotes", tc.input, quoted)
			}

			// Every interior quote must be escaped, so the string cannot be
			// closed early. After stripping the escape sequence there must be
			// no bare quote left inside.
			interior := quoted[1 : len(quoted)-1]
			if strings.Contains(strings.ReplaceAll(interior, `'\''`, ""), "'") {
				t.Errorf("shellQuote(%q) = %q leaves an unescaped quote", tc.input, quoted)
			}
		})
	}
}

func TestBaselineIsSafeToDescribe(t *testing.T) {
	playbook := Baseline(BaselineOptions{})

	if len(playbook) == 0 {
		t.Fatal("Baseline returned no steps")
	}

	seen := map[string]bool{}
	for _, step := range playbook {
		if step.ID() == "" {
			t.Errorf("step %q has no id", step.Title())
		}
		if seen[step.ID()] {
			t.Errorf("duplicate step id %q — the UI keys timeline rows by id", step.ID())
		}
		seen[step.ID()] = true

		// Titles are shown to people who do not know Linux, so they must not
		// leak command names.
		if step.Title() == "" {
			t.Errorf("step %q has no title", step.ID())
		}
	}
}

func TestBaselineInstallsKeyOnlyWhenGiven(t *testing.T) {
	without := Baseline(BaselineOptions{})
	if hasStep(without, "admin-key") {
		t.Error("admin-key step present with no public key configured")
	}

	with := Baseline(BaselineOptions{PublicKey: "ssh-ed25519 AAAA test@host"})
	if !hasStep(with, "admin-key") {
		t.Error("admin-key step missing even though a public key was given")
	}
}

// The firewall must open the port sshd is really on. Allowing only the OpenSSH
// profile would cut off a server running SSH anywhere but 22 — including the
// connection doing the work.
func TestFirewallAllowsDetectedSSHPort(t *testing.T) {
	firewall, ok := findStep(Baseline(BaselineOptions{}), "firewall")
	if !ok {
		t.Fatal("no firewall step in the baseline")
	}

	shell, ok := firewall.(shellStep)
	if !ok {
		t.Fatalf("firewall step is %T, expected shellStep", firewall)
	}

	joined := strings.Join(shell.apply, "\n")
	if !strings.Contains(joined, "sshd -T") {
		t.Error("firewall never asks sshd which port it is on")
	}

	enableAt := indexOfContains(shell.apply, "ufw --force enable")
	portAt := indexOfContains(shell.apply, "SSH_PORT")
	if enableAt < 0 || portAt < 0 {
		t.Fatalf("expected both a port rule and an enable command, got %v", shell.apply)
	}
	if portAt > enableAt {
		t.Error("the firewall is enabled before the SSH port is allowed, which would lock the operator out")
	}
}

// ssh-harden must refuse to disable password logins when no key exists. An
// unrecoverable server is far worse than a slightly weaker one.
func TestSSHHardenGuardsAgainstLockout(t *testing.T) {
	harden, ok := findStep(Baseline(BaselineOptions{}), "ssh-harden")
	if !ok {
		t.Fatal("no ssh-harden step in the baseline")
	}

	shell, ok := harden.(shellStep)
	if !ok {
		t.Fatalf("ssh-harden is %T, expected shellStep", harden)
	}

	if shell.skipIf == "" {
		t.Fatal("ssh-harden has no precondition, so it would disable password logins with no key present")
	}
	if !strings.Contains(shell.skipIf, "authorized_keys") {
		t.Errorf("ssh-harden precondition %q does not look for an installed key", shell.skipIf)
	}
	if shell.skipMessage == "" {
		t.Error("ssh-harden skips without telling anyone why")
	}

	// A bad sshd config plus a reload is a locked door with the key inside.
	if indexOfContains(shell.apply, "sshd -t") < 0 {
		t.Error("ssh-harden reloads sshd without validating the config first")
	}
	if indexOfContains(shell.apply, "sshd -t") > indexOfContains(shell.apply, "systemctl reload") {
		t.Error("ssh-harden validates the config after reloading, which is too late")
	}
}

func hasStep(playbook []Step, id string) bool {
	_, ok := findStep(playbook, id)
	return ok
}

func findStep(playbook []Step, id string) (Step, bool) {
	for _, step := range playbook {
		if step.ID() == id {
			return step, true
		}
	}
	return nil, false
}

func indexOfContains(commands []string, needle string) int {
	for i, cmd := range commands {
		if strings.Contains(cmd, needle) {
			return i
		}
	}
	return -1
}
