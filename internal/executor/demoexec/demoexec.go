// Package demoexec simulates a fresh Ubuntu server.
//
// It exists so the whole pipeline — job, step engine, live logs, status — can
// be exercised without owning a VPS. It runs the real steps and the real
// runner; only the machine on the other end is imaginary.
//
// It is not a test double for the SSH client. It deliberately models state, so
// running a playbook twice produces "changed" then "unchanged", which is the
// property most worth being able to see.
package demoexec

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/executor"
)

// fact keys tracked by the imaginary machine.
const (
	factAptUpdated  = "apt-updated"
	factPackages    = "packages"
	factAdminUser   = "admin-user"
	factAdminKey    = "admin-key"
	factSSHHardened = "ssh-hardened"
	factFirewall    = "firewall"
	factFail2ban    = "fail2ban"
	factAutoUpdates = "auto-updates"
	factTimezone    = "timezone"
	factSwap        = "swap"
	factSysctl      = "sysctl"
	factNetwork     = "docker-network"
)

type Machine struct {
	mu    sync.Mutex
	facts map[string]bool

	// Latency makes streamed logs arrive at a human pace rather than instantly.
	Latency time.Duration
}

func New() *Machine {
	return &Machine{facts: map[string]bool{}, Latency: 140 * time.Millisecond}
}

func (m *Machine) Describe() string { return "demo server (simulated)" }
func (m *Machine) Close() error     { return nil }

func (m *Machine) get(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.facts[key]
}

func (m *Machine) set(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.facts[key] = true
}

// rule maps a command to an effect on the imaginary machine. Rules are matched
// in order and the first match wins, so more specific patterns come first.
type rule struct {
	// all substrings must be present for the rule to match.
	all []string
	// when set, the rule reports whether fact holds (exit 0 if it does).
	checks string
	// when set, the rule marks fact as true and exits 0.
	sets string
	// stdout returned on a successful match.
	stdout string
	// inverted flips the check: exit 0 when the fact does NOT hold.
	inverted bool
}

var rules = []rule{
	// --- checks -----------------------------------------------------------
	{all: []string{"/var/lib/apt/lists", "newermt"}, checks: factAptUpdated},
	{all: []string{"dpkg -s"}, checks: factPackages},
	{all: []string{"id -u"}, checks: factAdminUser},
	{all: []string{"grep -qxF", "authorized_keys"}, checks: factAdminKey},

	// The ssh-harden precondition: "no key anywhere" is true until one is added.
	{all: []string{"! grep -rqs", "authorized_keys"}, checks: factAdminKey, inverted: true},

	{all: []string{"sshd -T", "permitrootlogin"}, checks: factSSHHardened},
	{all: []string{"ufw status"}, checks: factFirewall},
	{all: []string{"systemctl is-active", "fail2ban"}, checks: factFail2ban},
	{all: []string{"20auto-upgrades", "grep"}, checks: factAutoUpdates},
	{all: []string{"timedatectl show"}, checks: factTimezone},
	{all: []string{"swapon --show"}, checks: factSwap},
	{all: []string{"sysctl -n"}, checks: factSysctl},
	{all: []string{"docker network inspect"}, checks: factNetwork},

	// --- applies ----------------------------------------------------------
	{all: []string{"apt-get update"}, sets: factAptUpdated,
		stdout: "Reading package lists..."},
	{all: []string{"apt-get install"}, sets: factPackages,
		stdout: "Setting up ufw ...\nSetting up fail2ban ...\nSetting up unattended-upgrades ..."},
	{all: []string{"useradd"}, sets: factAdminUser},
	{all: []string{"authorized_keys"}, sets: factAdminKey},
	{all: []string{"sshd_config.d/10-ferrite.conf"}, sets: factSSHHardened},
	{all: []string{"ufw --force enable"}, sets: factFirewall,
		stdout: "Firewall is active and enabled on system startup"},
	{all: []string{"systemctl enable --now fail2ban"}, sets: factFail2ban},
	{all: []string{"20auto-upgrades"}, sets: factAutoUpdates},
	{all: []string{"timedatectl set-timezone"}, sets: factTimezone},
	{all: []string{"docker network create"}, sets: factNetwork,
		stdout: "b1946ac92492d2347c6235b4d2611184"},
	{all: []string{"mkswap"}, sets: factSwap,
		stdout: "Setting up swapspace version 1, size = 2 GiB"},
	{all: []string{"60-ferrite.conf"}, sets: factSysctl},
}

// facts describes the imaginary machine's hardware and OS.
var factOutputs = map[string]string{
	"hostname":    "demo-fra-1",
	"os":          "Ubuntu 24.04.1 LTS",
	"kernel":      "6.8.0-45-generic",
	"cpus":        "4",
	"memory":      "8589934592 2415919104",
	"disk":        "85899345920 21474836480",
	"uptime":      "184320.42",
	"ip":          "203.0.113.10",
	"loadaverage": "0.42",
	"whoami":      "root",
}

func (m *Machine) Run(ctx context.Context, cmd string) (executor.Result, error) {
	if m.Latency > 0 {
		select {
		case <-ctx.Done():
			return executor.Result{}, ctx.Err()
		case <-time.After(m.Latency):
		}
	}

	if out, ok := m.factCommand(cmd); ok {
		return executor.Result{Stdout: out}, nil
	}

	for _, r := range rules {
		if !matches(cmd, r.all) {
			continue
		}

		switch {
		case r.checks != "":
			held := m.get(r.checks)
			if r.inverted {
				held = !held
			}
			if held {
				return executor.Result{Stdout: r.stdout}, nil
			}
			return executor.Result{ExitCode: 1}, nil

		case r.sets != "":
			m.set(r.sets)
			return executor.Result{Stdout: r.stdout}, nil
		}
	}

	// Anything unrecognised is a benign command (chmod, install -d, reload).
	return executor.Result{}, nil
}

// factCommand answers the fact-gathering probes, which are matched on their
// distinctive fragments rather than the whole command.
func (m *Machine) factCommand(cmd string) (string, bool) {
	switch {
	case strings.Contains(cmd, "whoami"):
		// The imaginary machine is entered as root, so no sudo wrapping.
		return factOutputs["whoami"], true
	case strings.Contains(cmd, "PRETTY_NAME"):
		return factOutputs["os"], true
	case strings.Contains(cmd, "uname -r"):
		return factOutputs["kernel"], true
	case strings.Contains(cmd, "nproc"):
		return factOutputs["cpus"], true
	case strings.Contains(cmd, "/^Mem:/"):
		return factOutputs["memory"], true
	case strings.Contains(cmd, "df -B1"):
		return factOutputs["disk"], true
	case strings.Contains(cmd, "/proc/uptime"):
		return factOutputs["uptime"], true
	case strings.Contains(cmd, "/proc/loadavg"):
		return factOutputs["loadaverage"], true
	case strings.Contains(cmd, "hostname -I"):
		return factOutputs["ip"], true
	case strings.Contains(cmd, "hostname"):
		return factOutputs["hostname"], true
	}
	return "", false
}

func matches(cmd string, all []string) bool {
	for _, needle := range all {
		if !strings.Contains(cmd, needle) {
			return false
		}
	}
	return true
}
