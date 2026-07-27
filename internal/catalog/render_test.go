package catalog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Every compose file in the catalogue is written by hand, and a YAML mistake
// or a variable nobody sets would only show up as a failed install on someone
// else's server. `docker compose config` parses the file, resolves every
// interpolation and rejects an unknown key, so running it here turns that into
// a test failure instead.
//
// Skipped where Docker is absent, so CI without it still passes the rest.
func TestComposeFilesAreValid(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not installed")
	}
	if err := exec.Command("docker", "compose", "version").Run(); err != nil {
		t.Skip("the docker compose plugin is not available")
	}

	for _, tool := range All() {
		t.Run(tool.ID, func(t *testing.T) {
			in := Install{
				Tool:     tool,
				Password: "0123456789abcdef0123",
				Address:  "203.0.113.10",
				Domain:   "example.com",
				Email:    "ops@example.com",
			}

			dir := t.TempDir()
			composeFile := filepath.Join(dir, "compose.yaml")
			writeFile(t, composeFile, tool.compose)
			writeFile(t, filepath.Join(dir, ".env"), strings.Join(tool.env(in), "\n")+"\n")

			out, err := exec.Command(
				"docker", "compose", "--project-directory", dir, "-f", composeFile, "config",
			).CombinedOutput()
			if err != nil {
				t.Fatalf("compose rejected the file: %v\n%s", err, out)
			}

			// The image the catalogue advertises must be the one that runs, or
			// the version shown in the dashboard is fiction.
			if !strings.Contains(string(out), tool.Image) {
				t.Errorf("resolved config does not use %s:\n%s", tool.Image, out)
			}

			// A password reaching the compose file would be readable by anyone
			// who can list /opt/ferrite, and would show up in the job log.
			if strings.Contains(tool.compose, in.Password) {
				t.Error("the compose file contains the password; it belongs in .env only")
			}
		})
	}
}

// A tool that is reached with a connection string needs a password generated
// for it, and one that is not must not claim to have credentials.
func TestAccessAndVolumesAgree(t *testing.T) {
	for _, tool := range All() {
		if tool.NeedsPassword() != (tool.Access != nil) {
			t.Errorf("%s: NeedsPassword disagrees with Access", tool.ID)
		}
		for _, volume := range tool.Volumes {
			mount := "\n  " + volume + ":"
			if !strings.Contains(tool.compose, mount) {
				t.Errorf("%s: declares volume %q that its compose file never defines, "+
					"so deleting the data would silently delete nothing", tool.ID, volume)
			}
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
