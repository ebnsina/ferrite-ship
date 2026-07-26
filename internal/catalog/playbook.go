package catalog

import (
	"strconv"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// root is where every tool's files live. One predictable place means someone
// can look at a server and see what Ferrite Ship put there.
const root = "/opt/ferrite"

func dir(id string) string         { return root + "/" + id }
func composePath(id string) string { return dir(id) + "/compose.yaml" }
func envPath(id string) string     { return dir(id) + "/.env" }

// stampPath records the configuration that was last started. Comparing it with
// the files on disk is what makes "already running" mean "running *this*"
// rather than "running something" — without it, editing the compose file would
// report success while the old containers carried on.
func stampPath(id string) string { return dir(id) + "/.applied" }

func compose(id string, args ...string) string {
	return "docker compose --project-directory " + steps.Quote(dir(id)) +
		" -f " + steps.Quote(composePath(id)) + " " + strings.Join(args, " ")
}

// fingerprint hashes the two files that decide what runs.
func fingerprint(id string) string {
	return "cat " + steps.Quote(composePath(id)) + " " + steps.Quote(envPath(id)) +
		" | sha256sum | cut -d' ' -f1"
}

// matches tests whether a file already holds exactly this content.
func matches(path, content string) string {
	return "printf '%s' " + steps.Quote(content) + " | cmp -s - " + steps.Quote(path)
}

// write replaces a file's contents, creating it with mode first.
//
// The mode is set on creation rather than afterwards: `install -m 600` then
// writing leaves no window in which the env file exists world-readable with a
// password already in it.
func write(path, content, mode string) []string {
	return []string{
		"test -f " + steps.Quote(path) + " || install -m " + mode + " /dev/null " + steps.Quote(path),
		"chmod " + mode + " " + steps.Quote(path),
		"printf '%s' " + steps.Quote(content) + " > " + steps.Quote(path),
	}
}

// Steps is the playbook that puts this tool on a server, including the
// container runtime it needs. Running it again reports no changes.
func (in Install) Steps() []steps.Step {
	tool := in.Tool
	id := tool.ID
	envBody := strings.Join(tool.env(in), "\n") + "\n"

	playbook := steps.Docker()

	playbook = append(playbook, steps.Shell(steps.ShellSpec{
		ID:    id + "-config",
		Title: "Write " + tool.Name + "'s settings",
		Check: matches(composePath(id), tool.compose) + " && " + matches(envPath(id), envBody),
		Apply: append(
			append(
				[]string{"install -d -m 750 " + steps.Quote(dir(id))},
				write(composePath(id), tool.compose, "640")...,
			),
			write(envPath(id), envBody, "600")...,
		),
	}))

	if ports := tool.PublicPorts(); len(ports) > 0 {
		playbook = append(playbook, steps.Shell(steps.ShellSpec{
			ID:          id + "-firewall",
			Title:       "Let people reach " + tool.Name,
			SkipIf:      "! command -v ufw >/dev/null 2>&1",
			SkipMessage: "No firewall is running here, so these doors are already open.",
			Check:       strings.Join(portChecks(ports, true), " && "),
			Apply:       portRules(ports, "allow "),
		}))
	}

	playbook = append(playbook, steps.Shell(steps.ShellSpec{
		ID:    id + "-start",
		Title: "Start " + tool.Name + " and keep it started",
		// Three conditions, because any one alone lies. The stamp says the
		// running containers were started from these exact files; the service
		// count says none of them has since fallen over.
		Check: "test -f " + steps.Quote(stampPath(id)) +
			" && test \"$(cat " + steps.Quote(stampPath(id)) + ")\" = \"$(" + fingerprint(id) + ")\"" +
			" && test \"$(" + compose(id, "ps", "--services", "--status", "running") + " | wc -l)\"" +
			" = \"$(" + compose(id, "config", "--services") + " | wc -l)\"",
		Apply: []string{
			// Pulling first keeps the "starting" and "downloading a gigabyte"
			// parts of the log distinguishable.
			compose(id, "pull", "--quiet"),
			compose(id, "up", "-d", "--remove-orphans", "--wait"),
			fingerprint(id) + " > " + steps.Quote(stampPath(id)),
		},
	}))

	return playbook
}

// RemoveSteps stops a tool and takes its files away.
//
// purge additionally deletes the stored data, which cannot be undone — so it
// is a separate step with its own line in the log, rather than a flag buried
// inside the one that stops the containers.
func (t Tool) RemoveSteps(purge bool) []steps.Step {
	id := t.ID

	playbook := []steps.Step{
		steps.Shell(steps.ShellSpec{
			ID:          id + "-stop",
			Title:       "Stop " + t.Name + " and remove its settings",
			SkipIf:      "! test -d " + steps.Quote(dir(id)),
			SkipMessage: t.Name + " is not set up on this server, so there was nothing to remove.",
			// No check: reaching here means the directory exists, and the skip
			// above is what makes running this twice report "nothing to do".
			Apply: []string{
				compose(id, "down", "--remove-orphans"),
				"rm -rf " + steps.Quote(dir(id)),
			},
		}),
	}

	if ports := t.PublicPorts(); len(ports) > 0 {
		playbook = append(playbook, steps.Shell(steps.ShellSpec{
			ID:          id + "-firewall",
			Title:       "Close the doors " + t.Name + " was using",
			SkipIf:      "! command -v ufw >/dev/null 2>&1",
			SkipMessage: "No firewall is running here, so there was nothing to close.",
			Check:       strings.Join(portChecks(ports, false), " && "),
			// --force belongs only on delete: it is what stops ufw asking
			// "Proceed with operation (y|n)?" at a prompt nobody can answer.
			// Adding it to `allow` is rejected outright as invalid syntax.
			Apply: portRules(ports, "--force delete allow "),
		}))
	}

	if purge && len(t.Volumes) > 0 {
		names := volumeNames(t)
		playbook = append(playbook, steps.Shell(steps.ShellSpec{
			ID:    id + "-data",
			Title: "Delete the data " + t.Name + " stored",
			// Named explicitly rather than `compose down -v`, which needs the
			// compose file the step above has just deleted — and which would
			// therefore quietly leave the data behind.
			Check: "! docker volume ls --format '{{.Name}}' 2>/dev/null | grep -qxE " +
				steps.Quote(strings.Join(names, "|")),
			Apply: []string{"docker volume rm " + strings.Join(quoteAll(names), " ")},
		}))
	}

	return playbook
}

func volumeNames(t Tool) []string {
	names := make([]string, 0, len(t.Volumes))
	for _, volume := range t.Volumes {
		// Compose prefixes volumes with the project name, which is the `name:`
		// field at the top of every file in this catalogue.
		names = append(names, "ferrite-"+t.ID+"_"+volume)
	}
	return names
}

func quoteAll(values []string) []string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, steps.Quote(value))
	}
	return quoted
}

// portChecks asks whether the firewall already has (or no longer has) a rule
// for each port.
func portChecks(ports []Port, want bool) []string {
	checks := make([]string, 0, len(ports))
	for _, port := range ports {
		rule := strconv.Itoa(port.Number) + "/" + port.Protocol
		test := "ufw status | grep -q " + steps.Quote("^"+rule+" ")
		if !want {
			test = "! " + test
		}
		checks = append(checks, test)
	}
	return checks
}

func portRules(ports []Port, verb string) []string {
	rules := make([]string, 0, len(ports))
	for _, port := range ports {
		rules = append(rules, "ufw "+verb+strconv.Itoa(port.Number)+"/"+port.Protocol)
	}
	return rules
}
