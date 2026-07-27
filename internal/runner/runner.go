// Package runner executes a playbook against a server and records what happened.
package runner

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/alerts"
	"github.com/ebnsina/ferrite-ship/internal/dialer"
	"github.com/ebnsina/ferrite-ship/internal/executor"
	"github.com/ebnsina/ferrite-ship/internal/executor/demoexec"
	"github.com/ebnsina/ferrite-ship/internal/facts"
	"github.com/ebnsina/ferrite-ship/internal/notify"
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
	dialer *dialer.Dialer
	bus    *Bus
	sealer *secret.Sealer
	log    *slog.Logger

	// demoMachines keeps each demo server's simulated state across runs, so a
	// second baseline run reports "unchanged" exactly as a real one would.
	demoMu       sync.Mutex
	demoMachines map[string]*demoexec.Machine

	// alerts reports failures of jobs nobody started by hand. Nil until it is
	// set, so tests and the demo path run without a mail server.
	alerts *alerts.Reporter

	// acmeDirectory is which Let's Encrypt endpoint Traefik installs point at.
	acmeDirectory string

	// running guards against two jobs touching one server at once.
	runningMu sync.Mutex
	running   map[string]bool

	// DemoLatency paces the simulated machine so streamed logs arrive at a
	// readable speed. Tests set it to zero.
	DemoLatency time.Duration
}

func New(st *store.Store, d *dialer.Dialer, bus *Bus, sealer *secret.Sealer, log *slog.Logger) *Runner {
	return &Runner{
		store:        st,
		dialer:       d,
		bus:          bus,
		sealer:       sealer,
		log:          log,
		demoMachines: make(map[string]*demoexec.Machine),
		running:      make(map[string]bool),
		DemoLatency:  140 * time.Millisecond,
	}
}

// Certificates says which Let's Encrypt endpoint installs should use.
//
// Set after construction like Reporting, and for the same reason: it is a
// property of how this control plane is configured rather than of any server,
// and threading it through the constructor would put a certificate decision in
// the signature of everything that makes a runner.
func (r *Runner) Certificates(directory string) { r.acmeDirectory = directory }

// Reporting gives the runner somewhere to send news of an unattended failure.
//
// Set after construction rather than passed to New: the reporter needs a
// store, the runner needs a store, and threading one through the other's
// constructor would order main's setup around a dependency neither really has.
func (r *Runner) Reporting(reporter *alerts.Reporter) { r.alerts = reporter }

var ErrAlreadyRunning = errors.New("a job is already running on this server")

// plan is what a job is going to do. Everything that differs between setting a
// server up, installing a tool and removing one lives here; the machinery that
// connects, streams events, records history and refreshes facts is shared.
type plan struct {
	kind  string
	title string
	// build produces the playbook. It is called after connecting, because some
	// playbooks depend on what the machine turned out to be.
	//
	// The context is passed in rather than captured from the caller, and that
	// is not a style preference: a closure that captures the HTTP request's
	// context is holding one that is cancelled the moment the handler returns.
	// A build that queries the database then silently gets nothing — which is
	// how a deploy produced a proxy config listing no applications at all.
	build func(ctx context.Context, server store.Server) []steps.Step
	// secrets are masked wherever they would otherwise appear in the log.
	secrets []string
	// onFinish records the outcome against whatever this job was about, such
	// as marking an installation ready or failed.
	onFinish func(ctx context.Context, server store.Server, status store.JobStatus)
	// notifyAs is what to say if this job fails when nobody was watching it.
	//
	// Written at the call site because that is where the tool's name and the
	// right link are known, and acted on in one place because whether a message
	// is sent at all depends on who started the job — not on what it did.
	notifyAs *notify.Alert
	// after runs on the still-open connection once the playbook has finished,
	// for the rare case where the outcome includes something only the server
	// knows — a backup's size, say. Steps report pass or fail and nothing else,
	// and adding a return channel to every step to serve one job would be a
	// poor trade.
	after func(ctx context.Context, session *steps.Session, status store.JobStatus)
}

// StartBaseline queues the first-run playbook and returns as soon as the job
// row exists, so the caller can redirect to the log view immediately.
func (r *Runner) StartBaseline(
	ctx context.Context, userID, serverID, actor string, dryRun bool,
) (store.Job, error) {
	server, err := r.store.GetServer(ctx, userID, serverID)
	if err != nil {
		return store.Job{}, err
	}

	kind, title := "baseline", "Setting up "+server.Name
	if dryRun {
		kind, title = "baseline-check", "Checking "+server.Name+" (no changes)"
	}

	return r.start(ctx, server, actor, dryRun, plan{
		kind:  kind,
		title: title,
		build: func(_ context.Context, server store.Server) []steps.Step {
			return steps.Baseline(steps.BaselineOptions{
				PublicKey: server.PublicKey,
				// sshexec prefers a key when one is stored, so a password is only
				// in play when no key is.
				LoginUsesPassword: server.Kind == store.ConnectionSSH &&
					server.SealedPrivateKey == "" && server.SealedPassword != "",
			})
		},
	})
}

// start creates the job row and hands the work to a goroutine.
func (r *Runner) start(
	ctx context.Context, server store.Server, actor string, dryRun bool, p plan,
) (store.Job, error) {
	// One job per server at a time. Two playbooks running apt or docker
	// compose against the same machine would interleave into a mess that is
	// very hard to explain from the log afterwards.
	r.runningMu.Lock()
	if r.running[server.ID] {
		r.runningMu.Unlock()
		return store.Job{}, ErrAlreadyRunning
	}
	r.running[server.ID] = true
	r.runningMu.Unlock()

	job := store.Job{
		ID:        newID("job"),
		ServerID:  server.ID,
		Kind:      p.kind,
		Title:     p.title,
		Actor:     actor,
		Status:    store.JobRunning,
		StartedAt: time.Now().UTC(),
	}

	if err := r.store.CreateJob(ctx, job); err != nil {
		r.markDone(server.ID)
		return store.Job{}, err
	}

	// Detached context: the run must outlive the HTTP request that started it.
	go r.execute(context.WithoutCancel(ctx), server, job, p, dryRun)

	return job, nil
}

func (r *Runner) markDone(serverID string) {
	r.runningMu.Lock()
	delete(r.running, serverID)
	r.runningMu.Unlock()
}

func (r *Runner) execute(
	parent context.Context, server store.Server, job store.Job, p plan, dryRun bool,
) {
	defer r.markDone(server.ID)

	ctx, cancel := context.WithTimeout(parent, runTimeout)
	defer cancel()

	emitter := newEmitter(r.store, r.bus, job.ID)

	emitter.emit(ctx, store.Event{Type: EventJobStarted, Message: job.Title})

	// The outcome is recorded against whatever the job was about however this
	// function leaves, including the early returns below — an install that
	// could not connect must not sit at "installing" for ever.
	status := store.JobFailed
	if p.onFinish != nil {
		defer func() { p.onFinish(ctx, server, status) }()
	}

	exec, err := r.connect(ctx, server)
	if err != nil {
		emitter.emit(ctx, store.Event{
			Type: EventLog, Level: string(steps.LevelError),
			Message: "Could not connect: " + err.Error(),
		})
		r.finish(ctx, emitter, server, job, store.JobFailed, err, steps.Summary{}, p)
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
		r.finish(ctx, emitter, server, job, store.JobFailed, err, steps.Summary{DryRun: dryRun}, p)
		return
	}
	session = session.WithPrivilege(privilege).WithSecrets(p.secrets...)
	if privilege == steps.PrivilegeSudo {
		session.Log(steps.LevelInfo, "Using sudo, since this account is not root.")
	}

	summary := r.runPlaybook(ctx, session, p.build(ctx, server), emitter, dryRun)

	if p.after != nil {
		outcome := store.JobSucceeded
		if summary.Failed > 0 {
			outcome = store.JobFailed
		}
		p.after(ctx, session, outcome)
	}

	// Refresh facts whatever the outcome — even a failed run tells us something
	// about the machine.
	if gathered, err := facts.Gather(ctx, session); err == nil {
		_ = r.store.UpdateServerState(ctx, server.ID, statusFor(summary), gathered, time.Now().UTC())
	} else {
		_ = r.store.SetServerStatus(ctx, server.ID, statusFor(summary))
	}

	// Record a fleet snapshot so the dashboard's trends come from measurements
	// rather than from a made-up series.
	if err := r.store.SampleFleet(ctx, server.UserID); err != nil {
		r.log.Warn("could not record fleet sample", "error", err)
	}

	status = store.JobSucceeded
	var runErr error
	if summary.Failed > 0 {
		status = store.JobFailed
		if summary.FirstError != nil {
			// Redacted on the way out: this is written to the job row rather
			// than through the log, so it misses the masking in Session.
			runErr = errors.New(session.Redact(summary.FirstError.Error()))
		}
	}
	r.finish(ctx, emitter, server, job, status, runErr, summary, p)
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
	status store.JobStatus, runErr error, summary steps.Summary, p plan,
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

	r.report(ctx, server, job, status, p)
}

// report tells somebody, if nobody was watching.
//
// Only jobs the scheduler started. A person who pressed a button is already
// looking at the log, and mailing them about what is on their screen is how a
// notification becomes something to filter away.
func (r *Runner) report(
	ctx context.Context, server store.Server, job store.Job, status store.JobStatus, p plan,
) {
	if r.alerts == nil || p.notifyAs == nil || job.Actor != store.ActorScheduled {
		return
	}

	alert := *p.notifyAs
	alert.Server = server.Name
	settings := r.alerts.Settings(ctx, server.UserID)

	if status == store.JobSucceeded {
		r.alerts.Resolve(ctx, server.UserID, server.ID, settings, alert)
		return
	}

	// The job's own error, which has already been through the log's redaction
	// on its way into the row — a bucket key must not reach an inbox.
	alert.Detail = job.Error
	if alert.Detail == "" {
		alert.Detail = "The run finished with failures. The job log has the detail."
	}

	r.alerts.Raise(ctx, server.UserID, server.ID, settings, alert)
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

	client, _, err := r.dialer.Dial(ctx, server.UserID, server.ID)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func statusFor(summary steps.Summary) store.ServerStatus {
	if summary.Failed > 0 {
		return store.StatusDegraded
	}
	return store.StatusOnline
}
