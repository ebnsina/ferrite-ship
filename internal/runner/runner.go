// Package runner executes a playbook against a server and records what happened.
package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/executor"
	"github.com/ebnsina/ferrite-ship/internal/executor/demoexec"
	"github.com/ebnsina/ferrite-ship/internal/executor/sshexec"
	"github.com/ebnsina/ferrite-ship/internal/facts"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/steps"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// Event types published on the bus and stored in job history.
const (
	EventJobStarted  = "job-started"
	EventStepStarted = "step-started"
	EventLog         = "log"
	EventStepEnded   = "step-ended"
	EventJobEnded    = "job-ended"
)

// runTimeout bounds a whole playbook. A hung apt can otherwise pin a job open
// for as long as the process lives.
const runTimeout = 20 * time.Minute

type Runner struct {
	store  *store.Store
	sealer *secret.Sealer
	bus    *Bus
	log    *slog.Logger

	// demoMachines keeps each demo server's simulated state across runs, so a
	// second baseline run reports "unchanged" exactly as a real one would.
	demoMu       sync.Mutex
	demoMachines map[string]*demoexec.Machine

	// running guards against two jobs touching one server at once.
	runningMu sync.Mutex
	running   map[string]bool

	// DemoLatency paces the simulated machine so streamed logs arrive at a
	// readable speed. Tests set it to zero.
	DemoLatency time.Duration
}

func New(st *store.Store, sealer *secret.Sealer, bus *Bus, log *slog.Logger) *Runner {
	return &Runner{
		store:        st,
		sealer:       sealer,
		bus:          bus,
		log:          log,
		demoMachines: make(map[string]*demoexec.Machine),
		running:      make(map[string]bool),
		DemoLatency:  140 * time.Millisecond,
	}
}

var ErrAlreadyRunning = errors.New("a job is already running on this server")

// StartBaseline queues the first-run playbook and returns as soon as the job
// row exists, so the caller can redirect to the log view immediately.
func (r *Runner) StartBaseline(
	ctx context.Context, serverID, actor string, dryRun bool,
) (store.Job, error) {
	server, err := r.store.GetServer(ctx, serverID)
	if err != nil {
		return store.Job{}, err
	}

	r.runningMu.Lock()
	if r.running[serverID] {
		r.runningMu.Unlock()
		return store.Job{}, ErrAlreadyRunning
	}
	r.running[serverID] = true
	r.runningMu.Unlock()

	kind, title := "baseline", "Setting up "+server.Name
	if dryRun {
		kind, title = "baseline-check", "Checking "+server.Name+" (no changes)"
	}

	job := store.Job{
		ID:        newID("job"),
		ServerID:  serverID,
		Kind:      kind,
		Title:     title,
		Actor:     actor,
		Status:    store.JobRunning,
		StartedAt: time.Now().UTC(),
	}

	if err := r.store.CreateJob(ctx, job); err != nil {
		r.markDone(serverID)
		return store.Job{}, err
	}

	// Detached context: the run must outlive the HTTP request that started it.
	go r.execute(context.WithoutCancel(ctx), server, job, dryRun)

	return job, nil
}

func (r *Runner) markDone(serverID string) {
	r.runningMu.Lock()
	delete(r.running, serverID)
	r.runningMu.Unlock()
}

func (r *Runner) execute(
	parent context.Context, server store.Server, job store.Job, dryRun bool,
) {
	defer r.markDone(server.ID)

	ctx, cancel := context.WithTimeout(parent, runTimeout)
	defer cancel()

	emitter := newEmitter(r.store, r.bus, job.ID)

	emitter.emit(ctx, store.Event{Type: EventJobStarted, Message: job.Title})

	exec, err := r.connect(ctx, server)
	if err != nil {
		emitter.emit(ctx, store.Event{
			Type: EventLog, Level: string(steps.LevelError),
			Message: "Could not connect: " + err.Error(),
		})
		r.finish(ctx, emitter, server, job, store.JobFailed, err, steps.Summary{})
		return
	}
	defer func() { _ = exec.Close() }()

	_ = r.store.SetServerStatus(ctx, server.ID, store.StatusProvisioning)

	session := steps.NewSession(exec, func(level steps.Level, message string) {
		emitter.emit(ctx, store.Event{
			Type: EventLog, Level: string(level), Message: message,
		})
	})

	// Work out up front whether this account can make changes at all. Failing
	// here reads far better than every step failing with "permission denied".
	privilege, err := steps.DetectPrivilege(ctx, exec)
	if err != nil {
		emitter.emit(ctx, store.Event{
			Type: EventLog, Level: string(steps.LevelError), Message: err.Error(),
		})
		r.finish(ctx, emitter, server, job, store.JobFailed, err, steps.Summary{DryRun: dryRun})
		return
	}
	session = session.WithPrivilege(privilege)
	if privilege == steps.PrivilegeSudo {
		session.Log(steps.LevelInfo, "Using sudo, since this account is not root.")
	}

	playbook := steps.Baseline(steps.BaselineOptions{PublicKey: server.PublicKey})
	summary := r.runPlaybook(ctx, session, playbook, emitter, dryRun)

	// Refresh facts whatever the outcome — even a failed run tells us something
	// about the machine.
	if gathered, err := facts.Gather(ctx, session); err == nil {
		_ = r.store.UpdateServerState(ctx, server.ID, statusFor(summary), gathered, time.Now().UTC())
	} else {
		_ = r.store.SetServerStatus(ctx, server.ID, statusFor(summary))
	}

	// Record a fleet snapshot so the dashboard's trends come from measurements
	// rather than from a made-up series.
	if err := r.store.SampleFleet(ctx); err != nil {
		r.log.Warn("could not record fleet sample", "error", err)
	}

	status := store.JobSucceeded
	var runErr error
	if summary.Failed > 0 {
		status = store.JobFailed
		runErr = summary.FirstError
	}
	r.finish(ctx, emitter, server, job, status, runErr, summary)
}

func (r *Runner) runPlaybook(
	ctx context.Context, session *steps.Session, playbook []steps.Step, emitter *emitter,
	dryRun bool,
) steps.Summary {
	summary := steps.Summary{DryRun: dryRun}

	for _, step := range playbook {
		if ctx.Err() != nil {
			break
		}

		emitter.emit(ctx, store.Event{
			Type: EventStepStarted, StepID: step.ID(), StepTitle: step.Title(),
		})

		outcome, err := runStep(ctx, session, step, dryRun)
		switch outcome {
		case steps.OutcomeChanged:
			summary.Changed++
		case steps.OutcomeUnchanged:
			summary.Unchanged++
		case steps.OutcomeSkipped:
			summary.Skipped++
		case steps.OutcomeWouldChange:
			summary.WouldChange++
		case steps.OutcomeFailed:
			summary.Failed++
			if summary.FirstError == nil {
				summary.FirstError = err
			}
		}

		event := store.Event{
			Type: EventStepEnded, StepID: step.ID(), StepTitle: step.Title(),
			Outcome: string(outcome),
		}
		if err != nil {
			event.Message = err.Error()
		}
		emitter.emit(ctx, event)

		// A failed step usually invalidates everything after it, so stop.
		if outcome == steps.OutcomeFailed {
			break
		}
	}

	return summary
}

func runStep(
	ctx context.Context, session *steps.Session, step steps.Step, dryRun bool,
) (steps.Outcome, error) {
	reason, err := step.SkipReason(ctx, session)
	if err != nil {
		return steps.OutcomeFailed, err
	}
	if reason != "" {
		session.Log(steps.LevelSkipped, reason)
		return steps.OutcomeSkipped, nil
	}

	done, err := step.Check(ctx, session)
	if err != nil {
		return steps.OutcomeFailed, err
	}
	if done {
		session.Log(steps.LevelInfo, "Already done — nothing to change.")
		return steps.OutcomeUnchanged, nil
	}

	if dryRun {
		session.Log(steps.LevelInfo, "This would be changed. Nothing was altered.")
		return steps.OutcomeWouldChange, nil
	}

	if err := step.Apply(ctx, session); err != nil {
		session.Log(steps.LevelError, err.Error())
		return steps.OutcomeFailed, err
	}

	session.Log(steps.LevelChanged, "Done.")
	return steps.OutcomeChanged, nil
}

func (r *Runner) finish(
	ctx context.Context, emitter *emitter, server store.Server, job store.Job,
	status store.JobStatus, runErr error, summary steps.Summary,
) {
	finishedAt := time.Now().UTC()

	job.Status = status
	job.FinishedAt = &finishedAt
	job.Changed = summary.Changed
	job.Unchanged = summary.Unchanged
	job.Skipped = summary.Skipped + summary.WouldChange
	job.Failed = summary.Failed
	if runErr != nil {
		job.Error = runErr.Error()
	}

	if err := r.store.FinishJob(ctx, job); err != nil {
		r.log.Error("could not record job completion", "job", job.ID, "error", err)
	}

	emitter.emit(ctx, store.Event{
		Type:    EventJobEnded,
		Outcome: string(status),
		Message: summary.Describe(),
	})

	r.log.Info("job finished",
		"job", job.ID, "server", server.Name, "status", status,
		"changed", summary.Changed, "unchanged", summary.Unchanged,
		"skipped", summary.Skipped, "failed", summary.Failed)
}

// connect builds the executor for a server: the simulator, or a real SSH session.
func (r *Runner) connect(ctx context.Context, server store.Server) (executor.Executor, error) {
	if server.Kind == store.ConnectionDemo {
		r.demoMu.Lock()
		defer r.demoMu.Unlock()

		machine, ok := r.demoMachines[server.ID]
		if !ok {
			machine = demoexec.New()
			machine.Latency = r.DemoLatency
			r.demoMachines[server.ID] = machine
		}
		return machine, nil
	}

	password, err := r.sealer.Open(server.SealedPassword)
	if err != nil {
		return nil, fmt.Errorf("could not read the stored password: %w", err)
	}
	privateKey, err := r.sealer.Open(server.SealedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("could not read the stored key: %w", err)
	}

	return sshexec.Dial(ctx, sshexec.Config{
		Host:       server.Host,
		Port:       server.Port,
		User:       server.User,
		Password:   password,
		PrivateKey: privateKey,
	})
}

func statusFor(summary steps.Summary) store.ServerStatus {
	if summary.Failed > 0 {
		return store.StatusDegraded
	}
	return store.StatusOnline
}
