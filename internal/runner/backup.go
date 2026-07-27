package runner

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/catalog"
	"github.com/ebnsina/ferrite-ship/internal/notify"
	"github.com/ebnsina/ferrite-ship/internal/steps"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

var (
	// ErrNoDestination means backups have nowhere to go yet.
	ErrNoDestination = errors.New("runner: no backup destination is configured")
	// ErrNoBackupSupport is returned for a tool we cannot copy yet.
	ErrNoBackupSupport = errors.New("runner: this tool cannot be backed up yet")
)

// StartBackup copies a tool's data to the account's storage.
func (r *Runner) StartBackup(
	ctx context.Context, userID, serverID, toolID, actor string,
) (store.Job, error) {
	server, tool, destination, err := r.backupContext(ctx, userID, serverID, toolID)
	if err != nil {
		return store.Job{}, err
	}
	if !tool.Supported() {
		return store.Job{}, ErrNoBackupSupport
	}

	spec, _ := tool.BackupSpec()
	taken := time.Now().UTC()

	// The key carries the server, the tool and the moment, so a bucket shared
	// between several servers stays legible from the storage side alone.
	key := strings.TrimSuffix(destination.Prefix, "/")
	if key != "" {
		key += "/"
	}
	key += server.ID + "/" + tool.ID + "/" + taken.Format("2006-01-02T150405Z") + "." + spec.Extension

	record := store.Backup{
		ID:        newID("bak"),
		UserID:    userID,
		ServerID:  server.ID,
		ToolID:    tool.ID,
		ObjectKey: key,
		Status:    store.BackupRunning,
		CreatedAt: taken,
	}

	job, err := r.start(ctx, server, actor, false, plan{
		kind:  "tool-backup",
		title: "Backing up " + tool.Name + " from " + server.Name,
		build: func(context.Context, store.Server) []steps.Step {
			return catalog.BackupSteps(tool, destination, key)
		},
		secrets: []string{destination.AccessKey, destination.SecretKey},
		notifyAs: &notify.Alert{
			Kind: notify.KindBackupFailed,
			// Display text and de-duplication key are separate on purpose: the
			// name can be reworded in the catalogue without an open alert
			// becoming a second, unrelated one.
			Subject: tool.Name,
			Key:     tool.ID,
			Link:    "/dashboard/servers/" + server.ID + "/tools/" + tool.ID,
		},
		after: func(ctx context.Context, session *steps.Session, status store.JobStatus) {
			if status != store.JobSucceeded {
				return
			}
			// Read back what the dump actually produced. A backup row with a
			// size of zero and a status of "ready" is the kind of thing people
			// only discover when they try to restore it.
			size := readSize(ctx, session)
			if size == 0 {
				r.log.Warn("backup reported no bytes", "backup", record.ID, "tool", tool.ID)
			}
			if err := r.store.FinishBackup(ctx, record.ID, store.BackupReady, size); err != nil {
				r.log.Error("could not record backup size", "backup", record.ID, "error", err)
			}

			// Only now, with a good copy definitely in the bucket, is it safe
			// to remove the old ones.
			r.prune(ctx, session, userID, server.ID, tool, destination)
		},
		onFinish: func(ctx context.Context, _ store.Server, status store.JobStatus) {
			if status != store.JobSucceeded {
				if err := r.store.FinishBackup(ctx, record.ID, store.BackupFailed, 0); err != nil {
					r.log.Error("could not record backup failure", "backup", record.ID, "error", err)
				}
			}
		},
	})
	if err != nil {
		return store.Job{}, err
	}

	record.JobID = job.ID
	if err := r.store.CreateBackup(ctx, record); err != nil {
		return store.Job{}, err
	}

	return job, nil
}

// StartRestore puts a backup back.
//
// This overwrites whatever is there now. The caller is responsible for having
// said so plainly and been told to go ahead.
func (r *Runner) StartRestore(
	ctx context.Context, userID, backupID, actor string,
) (store.Job, error) {
	backup, err := r.store.GetBackup(ctx, userID, backupID)
	if err != nil {
		return store.Job{}, err
	}
	if backup.Status != store.BackupReady {
		return store.Job{}, ErrNoBackupSupport
	}

	server, tool, destination, err := r.backupContext(ctx, userID, backup.ServerID, backup.ToolID)
	if err != nil {
		return store.Job{}, err
	}

	return r.start(ctx, server, actor, false, plan{
		kind:  "tool-restore",
		title: "Restoring " + tool.Name + " on " + server.Name,
		build: func(context.Context, store.Server) []steps.Step {
			return catalog.RestoreSteps(tool, destination, backup.ObjectKey)
		},
		secrets: []string{destination.AccessKey, destination.SecretKey},
	})
}

// backupContext resolves the three things every backup job needs, with the
// ownership check that comes from asking the store with a user id.
func (r *Runner) backupContext(
	ctx context.Context, userID, serverID, toolID string,
) (store.Server, catalog.Tool, catalog.Destination, error) {
	server, err := r.store.GetServer(ctx, userID, serverID)
	if err != nil {
		return store.Server{}, catalog.Tool{}, catalog.Destination{}, err
	}

	tool, err := catalog.Find(toolID)
	if err != nil {
		return store.Server{}, catalog.Tool{}, catalog.Destination{}, err
	}

	// The tool has to be installed here, which is also the ownership check.
	if _, err := r.store.GetInstallation(ctx, userID, server.ID, toolID); err != nil {
		return store.Server{}, catalog.Tool{}, catalog.Destination{}, err
	}

	stored, err := r.store.GetBackupDestination(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return store.Server{}, catalog.Tool{}, catalog.Destination{}, ErrNoDestination
	}
	if err != nil {
		return store.Server{}, catalog.Tool{}, catalog.Destination{}, err
	}

	accessKey, err := r.sealer.Open(stored.SealedAccessKey)
	if err != nil {
		return store.Server{}, catalog.Tool{}, catalog.Destination{}, err
	}
	secretKey, err := r.sealer.Open(stored.SealedSecretKey)
	if err != nil {
		return store.Server{}, catalog.Tool{}, catalog.Destination{}, err
	}

	return server, tool, catalog.Destination{
		Endpoint:  stored.Endpoint,
		Region:    stored.Region,
		Bucket:    stored.Bucket,
		Prefix:    stored.Prefix,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}, nil
}

func readSize(ctx context.Context, session *steps.Session) int64 {
	out, err := session.Capture(ctx, "cat "+steps.Quote(catalog.SizeFile()))
	if err != nil {
		return 0
	}
	size, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return 0
	}
	return size
}

// prune deletes backups beyond what the schedule says to keep.
//
// Deliberately after the new backup has been recorded as ready, and never
// before: pruning first would mean a failure between the two left somebody
// with fewer copies than they asked for.
func (r *Runner) prune(
	ctx context.Context, session *steps.Session, userID, serverID string,
	tool catalog.Tool, destination catalog.Destination,
) {
	schedule, err := r.store.GetBackupSchedule(ctx, userID, serverID, tool.ID)
	if errors.Is(err, store.ErrNotFound) || schedule.Keep <= 0 {
		// No schedule, or no limit: backups taken by hand are kept until
		// somebody says otherwise.
		return
	}
	if err != nil {
		r.log.Error("could not read the schedule for pruning", "tool", tool.ID, "error", err)
		return
	}

	expired, err := r.store.ExpiredBackups(ctx, serverID, tool.ID, schedule.Keep)
	if err != nil {
		r.log.Error("could not find old backups", "tool", tool.ID, "error", err)
		return
	}
	if len(expired) == 0 {
		return
	}

	session.Logf(steps.LevelInfo, "Keeping the newest %d; removing %d older.",
		schedule.Keep, len(expired))

	for _, backup := range expired {
		out, err := session.Capture(ctx, catalog.DeleteCommand(destination, backup.ObjectKey))
		if err != nil {
			// Leave the row alone. A record pointing at an object that is
			// still there is recoverable; forgetting an object that still
			// costs money is not.
			r.log.Error("could not delete an old backup",
				"backup", backup.ID, "key", backup.ObjectKey, "error", err, "output", out)
			continue
		}
		if err := r.store.DeleteBackup(ctx, backup.ID); err != nil {
			r.log.Error("deleted a backup but could not forget it",
				"backup", backup.ID, "error", err)
		}
	}
}
