package catalog

import (
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// Destination is where a backup is sent, resolved to the values the server
// needs. Credentials arrive already decrypted and are masked in the job log by
// the session rather than by anything here.
type Destination struct {
	Endpoint string
	Region   string
	Bucket   string
	// Prefix keeps several servers apart inside one bucket.
	Prefix    string
	AccessKey string
	SecretKey string
}

const (
	// transferDir holds the rclone config, away from any one tool.
	transferDir  = root + "/backup"
	transferConf = transferDir + "/rclone.conf"
	// sizeFile is where the dump records how many bytes it produced, so the
	// size can be read back without asking the bucket.
	sizeFile = transferDir + "/last-size"
)

// rclone renders an rclone invocation against the configured remote.
func rclone(args ...string) string {
	return "rclone --config " + steps.Quote(transferConf) + " " + strings.Join(args, " ")
}

// inBash runs a command under bash explicitly.
//
// Steps are executed through `sh`, which on Ubuntu is dash — and dash has
// neither `set -o pipefail` nor the `>(...)` process substitution the transfer
// commands rely on. Without this the pipeline silently reports success when
// the dump half of it failed, which is the single worst way for a backup to
// behave.
func inBash(command string) string {
	return "bash -c " + steps.Quote(command)
}

// TransferSteps installs and configures rclone.
//
// rclone rather than the AWS CLI: one static binary with no Python runtime
// behind it, and it speaks to every S3-compatible service — AWS, R2, B2,
// Spaces, MinIO — from the same config.
func TransferSteps(d Destination) []steps.Step {
	config := strings.Join([]string{
		"[ferrite]",
		"type = s3",
		"provider = Other",
		"env_auth = false",
		"access_key_id = " + d.AccessKey,
		"secret_access_key = " + d.SecretKey,
		"endpoint = " + d.Endpoint,
		"region = " + d.Region,
		// Most S3-compatible services other than AWS itself only serve
		// path-style addressing, and getting this wrong looks like a DNS
		// failure rather than a configuration one.
		"force_path_style = true",
		"",
	}, "\n")

	return []steps.Step{
		steps.Shell(steps.ShellSpec{
			ID:    "rclone",
			Title: "Install the tool that copies backups off this server",
			Check: "command -v rclone >/dev/null 2>&1",
			Apply: []string{
				`DEBIAN_FRONTEND=noninteractive apt-get install -y -qq rclone`,
			},
		}),
		steps.Shell(steps.ShellSpec{
			ID:    "rclone-config",
			Title: "Point it at your storage",
			Check: matches(transferConf, config),
			Apply: append(
				[]string{"install -d -m 700 " + steps.Quote(transferDir)},
				write(transferConf, config, "600")...,
			),
		}),
	}
}

// BackupSteps dumps a tool and streams the result straight to storage.
func BackupSteps(t Tool, d Destination, key string) []steps.Step {
	spec, ok := t.BackupSpec()
	if !ok {
		return nil
	}

	remote := "ferrite:" + d.Bucket + "/" + key

	playbook := TransferSteps(d)
	return append(playbook, steps.Shell(steps.ShellSpec{
		ID:    t.ID + "-backup",
		Title: "Copy " + t.Name + " to your storage",
		Apply: []string{
			// Piped rather than written to a file first: a database worth
			// backing up is often too large to fit beside itself on the disk.
			// `tee` into wc counts the bytes on the way past, which is cheaper
			// than asking the bucket afterwards and works even where listing is
			// not permitted.
			inBash("set -o pipefail; " + spec.Dump +
				" | tee >(wc -c > " + steps.Quote(sizeFile) + ") | " +
				rclone("rcat", steps.Quote(remote))),
		},
	}))
}

// RestoreSteps streams a backup out of storage and back into the tool.
func RestoreSteps(t Tool, d Destination, key string) []steps.Step {
	spec, ok := t.BackupSpec()
	if !ok {
		return nil
	}

	remote := "ferrite:" + d.Bucket + "/" + key

	playbook := TransferSteps(d)

	if len(spec.RestoreBefore) > 0 {
		playbook = append(playbook, steps.Shell(steps.ShellSpec{
			ID:    t.ID + "-restore-stop",
			Title: "Pause " + t.Name + " while it is replaced",
			Apply: spec.RestoreBefore,
		}))
	}

	playbook = append(playbook, steps.Shell(steps.ShellSpec{
		ID:    t.ID + "-restore",
		Title: "Put " + t.Name + "'s data back",
		Apply: []string{
			inBash("set -o pipefail; " + rclone("cat", steps.Quote(remote)) + " | " + spec.Restore),
		},
	}))

	if len(spec.RestoreAfter) > 0 {
		playbook = append(playbook, steps.Shell(steps.ShellSpec{
			ID:    t.ID + "-restore-load",
			Title: "Get " + t.Name + " reading it",
			Apply: spec.RestoreAfter,
		}))
	}

	return playbook
}

// SizeFile is where BackupSteps leaves the byte count.
func SizeFile() string { return sizeFile }
