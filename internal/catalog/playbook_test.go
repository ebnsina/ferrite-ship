package catalog

import (
	"strings"
	"testing"
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
