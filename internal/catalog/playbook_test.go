package catalog

import (
	"strings"
	"testing"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// ufw takes --force on enable, reset and delete, and rejects it anywhere else
// as "ERROR: Invalid syntax". Putting it on `allow` failed the whole MediaMTX
// install on a real server, so the exact wording is pinned here.
func TestFirewallRulesUseTheSyntaxUfwAccepts(t *testing.T) {
	ports := []Port{
		{Number: 8554, Protocol: "tcp", Public: true},
		{Number: 8189, Protocol: "udp", Public: true},
	}

	opens := portRules(ports, "allow ")
	want := []string{"ufw allow 8554/tcp", "ufw allow 8189/udp"}
	for i, rule := range opens {
		if rule != want[i] {
			t.Errorf("open rule %d is %q, want %q", i, rule, want[i])
		}
	}

	// Deleting is the one that must not prompt: there is no terminal on the
	// other end to answer "Proceed with operation (y|n)?".
	closes := portRules(ports, "--force delete allow ")
	for _, rule := range closes {
		if !strings.HasPrefix(rule, "ufw --force delete allow ") {
			t.Errorf("close rule %q would stop at a prompt", rule)
		}
	}
}

// The steps that open ports must cover every port the catalogue advertises as
// public, or a tool installs cleanly and is unreachable.
func TestEveryPublicPortIsOpened(t *testing.T) {
	for _, tool := range All() {
		public := tool.PublicPorts()
		if len(public) == 0 {
			continue
		}

		var opened string
		for _, step := range (Install{Tool: tool, Password: "x", Address: "1.2.3.4"}).Steps() {
			if step.ID() == tool.ID+"-firewall" {
				opened = strings.Join(portRules(public, "allow "), "\n")
			}
		}
		if opened == "" {
			t.Fatalf("%s has public ports but no step opens them", tool.ID)
		}
		for _, port := range public {
			if !strings.Contains(opened, portRules([]Port{port}, "allow ")[0]) {
				t.Errorf("%s: port %d/%s is advertised as public but never opened",
					tool.ID, port.Number, port.Protocol)
			}
		}
	}
}

// A private port reaching the firewall step would put a database on the
// internet, which is the failure this catalogue exists to avoid.
func TestPrivatePortsAreNeverOpened(t *testing.T) {
	for _, tool := range All() {
		for _, port := range tool.Ports {
			if port.Public {
				continue
			}
			bind := "127.0.0.1:" + itoa(port.Number)
			if !strings.Contains(tool.compose, bind) {
				t.Errorf("%s: port %d is private but its compose file does not bind it to %s",
					tool.ID, port.Number, bind)
			}
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

// "Also delete the data" must only be offered where there is data to delete,
// and must always be offered where there is. Getting this wrong either hides
// an irreversible action or invents a frightening one.
func TestKeepsDataMatchesTheVolumesThatExist(t *testing.T) {
	for _, tool := range All() {
		if tool.KeepsData != (len(tool.Volumes) > 0) {
			t.Errorf("%s: KeepsData is %v but it has %d volumes",
				tool.ID, tool.KeepsData, len(tool.Volumes))
		}
		mentionsKeeping := strings.Contains(tool.DataNote, "keeps")
		if tool.KeepsData != mentionsKeeping {
			t.Errorf("%s: KeepsData is %v but the note shown to the person says %q",
				tool.ID, tool.KeepsData, tool.DataNote)
		}
	}
}

// Every tool needs a brand colour, because the catalogue leans on them to be
// recognisable at a glance and one missing entry renders as an invisible tile.
func TestEveryToolHasABrandColour(t *testing.T) {
	for _, tool := range All() {
		if len(tool.Accent) != 7 || tool.Accent[0] != '#' {
			t.Errorf("%s: accent %q is not a #rrggbb colour", tool.ID, tool.Accent)
		}
	}
}

// The restore pipeline feeds a stream into exactly one command, and which one
// is decided by shell precedence rather than by intent: `stream | a && b`
// binds the pipe to `a`. Chaining a "stop the container" onto Restore with &&
// therefore fed the backup to the stop command and wrote an empty file — a
// restore that reported success and destroyed the data. Anything that has to
// happen first belongs in RestoreBefore.
func TestRestoreCommandsDoNotChainAroundThePipe(t *testing.T) {
	for _, tool := range All() {
		spec, ok := tool.BackupSpec()
		if !ok {
			continue
		}

		// Only top-level chaining matters. An && inside a quoted `sh -c '...'`
		// is that command's own business — the outer process still receives
		// the pipe, and passes it down.
		if strings.Contains(unquoted(spec.Restore), "&&") {
			t.Errorf("%s: Restore chains commands at the top level (%q). The pipe "+
				"binds to the first one, so only the first would receive the "+
				"backup — use RestoreBefore.", tool.ID, spec.Restore)
		}
		if strings.Contains(unquoted(spec.Dump), "&&") {
			t.Errorf("%s: Dump chains commands at the top level, so only the last "+
				"one's output is piped onward and the backup would be truncated", tool.ID)
		}
	}
}

// A tool that can be backed up must be restorable, or the backup is a file
// nobody can use.
func TestEveryBackupCanBeRestored(t *testing.T) {
	for _, tool := range All() {
		spec, ok := tool.BackupSpec()
		if !ok {
			continue
		}
		if spec.Dump == "" || spec.Restore == "" {
			t.Errorf("%s: has a backup spec with an empty half", tool.ID)
		}
		if spec.Extension == "" {
			t.Errorf("%s: backups need a file extension to be named by", tool.ID)
		}
		if spec.Warning == "" {
			t.Errorf("%s: restoring overwrites data, so it needs a warning to show first", tool.ID)
		}
	}
}

// unquoted drops single-quoted spans, leaving the shell operators that the
// outer shell will actually act on.
func unquoted(command string) string {
	var out strings.Builder
	inQuotes := false
	for _, r := range command {
		if r == '\'' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// ClickHouse refuses to write a backup anywhere it has not been told about, and
// the three places that have to agree are far apart: the disk named in the
// compose file's XML, the path it is mounted at, and the name used in the
// BACKUP statement. A rename in one of them fails at three in the morning with
// "Disk backups not found", which reads like a bug in the product.
func TestClickHouseBackupDiskIsWiredUpEndToEnd(t *testing.T) {
	tool, err := Find("clickhouse")
	if err != nil {
		t.Fatalf("find clickhouse: %v", err)
	}

	spec, ok := tool.BackupSpec()
	if !ok {
		t.Fatal("clickhouse has no backup spec")
	}

	for _, needed := range []string{
		"<allowed_disk>backups</allowed_disk>",
		"<allowed_path>/backups/</allowed_path>",
		"<path>/backups/</path>",
		"- backups:/backups",
		"target: /etc/clickhouse-server/config.d/backups.xml",
	} {
		if !strings.Contains(tool.compose, needed) {
			t.Errorf("the compose file is missing %q, so BACKUP has nowhere to write", needed)
		}
	}

	// Read back through the escaping. These commands are quoted for a remote
	// `sh -c`, so the statement's own quotes appear as '\'' and the literal
	// SQL is not there to compare against until that is undone.
	if !strings.Contains(unescaped(spec.Dump), "Disk('backups', 'ferrite.zip')") {
		t.Errorf("the dump does not write to the disk the compose file declares:\n%s", spec.Dump)
	}
	if !strings.Contains(spec.Dump, "/backups/ferrite.zip") {
		t.Error("the dump does not read back the file it just wrote")
	}
	if !strings.Contains(spec.Restore, "/backups/restore.zip") {
		t.Error("the restore does not put the incoming backup on the backups volume")
	}

	// Every temporary file has to be removed, or a server accumulates a copy
	// of its own database on every run until the disk is full.
	if strings.Count(spec.Dump, "rm -f /backups/ferrite.zip") != 2 {
		t.Error("the dump should clear a stale file before writing and delete its own afterwards")
	}
	if !strings.Contains(strings.Join(spec.RestoreAfter, " "), "rm -f /backups/restore.zip") {
		t.Error("the restore leaves its copy of the backup on the server")
	}
}

// Restoring has to replace, not merge. ClickHouse appends when told to write
// into a table that already has rows, so a restore meant to undo a mistake
// would leave every row in the database twice.
func TestClickHouseRestoreReplacesRatherThanAppends(t *testing.T) {
	tool, _ := Find("clickhouse")
	spec, _ := tool.BackupSpec()

	after := unescaped(strings.Join(spec.RestoreAfter, "\n"))

	if !strings.Contains(after, "DROP DATABASE IF EXISTS app SYNC") {
		t.Error("the existing database is not dropped first")
	}
	if strings.Contains(after, "allow_non_empty_tables") {
		t.Error("allow_non_empty_tables appends rows to what is already there; it must not be used to restore")
	}

	drop := strings.Index(after, "DROP DATABASE")
	restore := strings.Index(after, "RESTORE DATABASE")
	if drop < 0 || restore < 0 || drop > restore {
		t.Error("the drop has to happen before the restore, or the restore fails on a database that exists")
	}
}

// unescaped undoes one round of single-quote escaping, so a test can compare
// against the command as the server's shell will finally see it.
func unescaped(command string) string {
	return strings.ReplaceAll(command, `'\''`, "'")
}

// Every tool has to join the shared network, and has to expect to find it
// rather than declare one of its own.
//
// A tool that quietly stays on its own project network installs cleanly, runs
// correctly, and is simply invisible to the reverse proxy — so the failure
// does not appear until someone points a domain at it and gets a 404 from
// Traefik, which reads like a DNS problem rather than a missing line of YAML.
//
// `external: true` is half the assertion. Without it compose creates a second
// network with the same name prefixed by the project, joins that instead, and
// everything above happens anyway with nothing to see in the file.
func TestEveryToolJoinsTheSharedNetwork(t *testing.T) {
	for _, tool := range All() {
		// A prefix, not the whole list: a tool that brings its own dependency
		// puts the routed service on both this network and the project's own,
		// and only the routed one belongs here. The dependency itself must
		// stay off the shared network entirely — everything on it can reach
		// everything else, so a bundled database that joined would be visible
		// to every other tool on the server.
		if !strings.Contains(tool.compose, "networks: ["+steps.Network) {
			t.Errorf("%s has no service on the %s network, so nothing can route to it",
				tool.ID, steps.Network)
		}

		declared := "networks:\n  " + steps.Network + ":\n    external: true"
		if !strings.Contains(tool.compose, declared) {
			t.Errorf("%s does not declare %s as external, so compose would make its own",
				tool.ID, steps.Network)
		}
	}
}

// The network has to exist before any compose file that expects to find it is
// started, or the first tool on a fresh server fails with "network ferrite
// declared as external, but could not be found".
func TestTheSharedNetworkIsCreatedBeforeAnythingUsesIt(t *testing.T) {
	in := Install{Tool: postgres, Password: "x", Address: "1.2.3.4"}

	created, started := -1, -1
	for i, step := range in.Steps() {
		switch step.ID() {
		case "docker-network":
			created = i
		case postgres.ID + "-start":
			started = i
		}
	}

	if created < 0 {
		t.Fatal("nothing creates the shared network")
	}
	if started < 0 {
		t.Fatal("nothing starts the tool")
	}
	if created > started {
		t.Errorf("the network is created at step %d but needed at step %d", created, started)
	}
}

// The subdomain in a tool's routing rule and the address the dashboard hands
// out have to be the same name.
//
// They are written in two places — a label inside a compose file, and
// Subdomain() in Go — so nothing but a test stops them drifting. If they do,
// Traefik answers on one name while the dashboard sends people to another, and
// the only symptom is a certificate error on an address that looks right.
func TestWebToolsRouteToTheirOwnSubdomain(t *testing.T) {
	for _, tool := range All() {
		if !tool.Web {
			continue
		}

		rule := "Host(`" + tool.ID + ".${FERRITE_DOMAIN}`)"
		if !strings.Contains(tool.compose, rule) {
			t.Errorf("%s does not route %s, so the dashboard would send people "+
				"to an address nothing answers on", tool.ID, rule)
		}

		if got := tool.Subdomain("example.com"); got != tool.ID+".example.com" {
			t.Errorf("%s.Subdomain() = %q, want %s.example.com", tool.ID, got, tool.ID)
		}

		// Which port, exactly. Traefik refuses to guess once a container
		// exposes more than one, and guessing wrong is worse than not
		// guessing: routing RabbitMQ's 5672 would put a binary protocol
		// behind an HTTP proxy, which fails looking like a broken queue
		// rather than like a misrouted port.
		port := "loadbalancer.server.port=" + itoa(tool.RoutedPort())
		if !strings.Contains(tool.compose, port) {
			t.Errorf("%s should route to %s, and its compose file does not say so",
				tool.ID, port)
		}

		// The two-faced tools are the reason RoutedPort exists. If it ever
		// returns the port clients use, the management page is unreachable and
		// the queue is being proxied.
		if tool.HasSeparateWeb() && tool.RoutedPort() == tool.Access.Port {
			t.Errorf("%s routes the port clients use rather than its web interface", tool.ID)
		}
	}
}

// A web tool must not be published directly. Traefik reaches it over the
// shared network, so a public port would be a second way in that bypasses the
// certificate and the redirect — and, because published ports bypass ufw,
// nothing downstream would catch it.
func TestWebToolsAreNotPublishedThemselves(t *testing.T) {
	for _, tool := range All() {
		if !tool.Web {
			continue
		}
		if len(tool.PublicPorts()) > 0 {
			t.Errorf("%s opens %d public port(s); it should only be reachable through Traefik",
				tool.ID, len(tool.PublicPorts()))
		}
	}
}

// Exactly one service per tool sits on the shared network.
//
// Everything on that network can reach everything else on it, so a tool that
// brings its own database must keep it on the project's own network instead.
// Keycloak's PostgreSQL holds every account on the server; joining the shared
// network would make it reachable by every other container in the catalogue,
// and the compose file would still look entirely reasonable.
//
// One is also the right number at the other end: a second service on the
// shared network would be competing for a name in a namespace the whole
// catalogue shares, and compose resolves those by service name.
func TestOnlyTheRoutedServiceIsOnTheSharedNetwork(t *testing.T) {
	for _, tool := range All() {
		if got := strings.Count(tool.compose, "networks: ["+steps.Network); got != 1 {
			t.Errorf("%s puts %d services on the %s network; exactly one should be",
				tool.ID, got, steps.Network)
		}
	}
}

// A tool that cannot work without a domain has to say so, or it installs
// cleanly and fails the moment somebody tries to use it.
func TestToolsNeedingADomainBuildTheirAddressFromIt(t *testing.T) {
	for _, tool := range All() {
		if !tool.NeedsDomain {
			continue
		}

		in := Install{Tool: tool, Password: "x", Domain: "example.com", Email: "ops@example.com"}
		var rendered string
		for _, line := range tool.env(in) {
			rendered += line + "\n"
		}

		// Whatever address it is told about itself must be the one Traefik
		// routes. Keycloak compares its issuer exactly, so a mismatch here is
		// sign-ins that work and applications that then reject every token.
		if strings.Contains(rendered, "_URL=") &&
			!strings.Contains(rendered, "https://"+tool.Subdomain("example.com")) {
			t.Errorf("%s is told an address that is not %s:\n%s",
				tool.ID, tool.Subdomain("example.com"), rendered)
		}
	}
}
