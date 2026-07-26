package runner

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/catalog"
	"github.com/ebnsina/ferrite-ship/internal/steps"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// ErrNoAddress is returned for a tool that has to be told how the outside
// world reaches the server, on a server that has no address — the simulated
// one. Installing anyway would produce a media server nobody could watch.
var ErrNoAddress = errors.New("this tool needs the server's public address")

// StartInstall puts a tool from the catalogue on a server.
//
// Running it again for a tool that is already there is a repair rather than a
// mistake: every step re-checks, and the generated password is reused so the
// applications already connecting keep working.
func (r *Runner) StartInstall(
	ctx context.Context, userID, serverID, toolID, actor string, dryRun bool,
) (store.Job, error) {
	server, err := r.store.GetServer(ctx, userID, serverID)
	if err != nil {
		return store.Job{}, err
	}

	tool, err := catalog.Find(toolID)
	if err != nil {
		return store.Job{}, err
	}

	install := catalog.Install{Tool: tool, Address: server.Host}
	if tool.NeedsAddress && install.Address == "" {
		return store.Job{}, ErrNoAddress
	}

	// Reuse the existing credential where there is one. Generating a fresh
	// password on a repair would silently break every application already
	// using the old one.
	existing, err := r.store.GetInstallation(ctx, userID, serverID, toolID)
	switch {
	case err == nil:
		install.Password, err = r.sealer.Open(existing.SealedPassword)
		if err != nil {
			return store.Job{}, err
		}
	case errors.Is(err, store.ErrNotFound):
		// New installation.
	default:
		return store.Job{}, err
	}

	sealed := ""
	if tool.NeedsPassword() && install.Password == "" {
		install.Password, err = generatePassword()
		if err != nil {
			return store.Job{}, err
		}
		sealed, err = r.sealer.Seal(install.Password)
		if err != nil {
			return store.Job{}, err
		}
	}

	verb := "Installing"
	if err == nil && existing.Status == store.InstallReady {
		verb = "Checking"
	}
	if dryRun {
		verb = "Checking"
	}

	job, err := r.start(ctx, server, actor, dryRun, plan{
		kind:  "tool-install",
		title: verb + " " + tool.Name + " on " + server.Name,
		build: func(context.Context, store.Server) []steps.Step { return install.Steps() },
		// The generated password is written to a file by a command that is
		// echoed into the job log. Without this it would be sitting in the
		// database and on screen in plain text.
		secrets: []string{install.Password},
		onFinish: func(ctx context.Context, server store.Server, status store.JobStatus) {
			state := store.InstallReady
			if status != store.JobSucceeded {
				state = store.InstallFailed
			}
			if err := r.store.SetInstallationStatus(ctx, server.ID, toolID, state); err != nil {
				r.log.Error("could not record install outcome",
					"server", server.ID, "tool", toolID, "error", err)
			}
		},
	})
	if err != nil {
		return store.Job{}, err
	}

	// Recorded after the job exists so the row can point at it, but the job
	// only runs in a goroutine — there is no window where it has finished and
	// found no installation to update.
	now := time.Now().UTC()
	record := store.Installation{
		ID:             newID("ins"),
		UserID:         userID,
		ServerID:       serverID,
		ToolID:         toolID,
		Version:        tool.Version,
		Status:         store.InstallPending,
		SealedPassword: sealed,
		LastJobID:      job.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := r.store.SaveInstallation(ctx, record); err != nil {
		return store.Job{}, err
	}

	return job, nil
}

// StartRemove takes a tool off a server.
//
// purge additionally deletes the stored data. It is the caller's job to have
// asked first: nothing here can put it back.
func (r *Runner) StartRemove(
	ctx context.Context, userID, serverID, toolID, actor string, purge bool,
) (store.Job, error) {
	server, err := r.store.GetServer(ctx, userID, serverID)
	if err != nil {
		return store.Job{}, err
	}

	tool, err := catalog.Find(toolID)
	if err != nil {
		return store.Job{}, err
	}

	// Refuse to remove something this account does not have. Without it, one
	// user could stop another's containers by guessing a server id — the store
	// call is the scoping check.
	if _, err := r.store.GetInstallation(ctx, userID, serverID, toolID); err != nil {
		return store.Job{}, err
	}

	title := "Removing " + tool.Name + " from " + server.Name
	if purge {
		title = "Removing " + tool.Name + " and its data from " + server.Name
	}

	job, err := r.start(ctx, server, actor, false, plan{
		kind:  "tool-remove",
		title: title,
		build: func(context.Context, store.Server) []steps.Step { return tool.RemoveSteps(purge) },
		onFinish: func(ctx context.Context, server store.Server, status store.JobStatus) {
			if status != store.JobSucceeded {
				// Leave the row behind on failure. Claiming the tool is gone
				// while its containers are still running is worse than showing
				// it as broken, because the owner would stop looking for it.
				if err := r.store.SetInstallationStatus(ctx, server.ID, toolID, store.InstallFailed); err != nil {
					r.log.Error("could not record removal failure",
						"server", server.ID, "tool", toolID, "error", err)
				}
				return
			}
			if err := r.store.DeleteInstallation(ctx, userID, server.ID, toolID); err != nil {
				r.log.Error("could not forget installation",
					"server", server.ID, "tool", toolID, "error", err)
			}
		},
	})
	if err != nil {
		return store.Job{}, err
	}

	if err := r.store.SetInstallationStatus(ctx, serverID, toolID, store.InstallRemoving); err != nil {
		return store.Job{}, err
	}
	return job, nil
}

// generatePassword returns a credential for a newly installed tool.
//
// crypto/rand.Text rather than an alphabet of our own: it is uniform without
// the modulo bias that hand-rolled versions usually have, and its base32
// output contains no punctuation. That matters more than it sounds, because
// these passwords are substituted into YAML, shell commands, .env files and
// connection URLs, each with different ideas about what needs escaping. Its 26
// characters carry about 130 bits.
func generatePassword() (string, error) {
	return rand.Text(), nil
}
