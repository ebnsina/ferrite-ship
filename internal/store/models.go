package store

import (
	"time"

	"github.com/ebnsina/ferrite-ship/internal/facts"
)

// ConnectionKind is how the control plane reaches a server.
type ConnectionKind string

const (
	// ConnectionSSH is a real machine.
	ConnectionSSH ConnectionKind = "ssh"
	// ConnectionDemo is the simulated machine, for trying the product out.
	ConnectionDemo ConnectionKind = "demo"
)

type ServerStatus string

const (
	StatusUnknown      ServerStatus = "unknown"
	StatusOnline       ServerStatus = "online"
	StatusDegraded     ServerStatus = "degraded"
	StatusOffline      ServerStatus = "offline"
	StatusProvisioning ServerStatus = "provisioning"
)

type Server struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Kind     ConnectionKind `json:"connectionKind"`
	Host     string         `json:"host"`
	Port     int            `json:"port"`
	User     string         `json:"user"`
	Region   string         `json:"region"`
	Status   ServerStatus   `json:"status"`
	Facts    facts.Facts    `json:"facts"`
	Services []string       `json:"services"`

	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt *time.Time `json:"lastSeenAt"`

	// Sealed credentials. Never serialised to the API.
	SealedPassword   string `json:"-"`
	SealedPrivateKey string `json:"-"`
}

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
)

type Job struct {
	ID       string    `json:"id"`
	ServerID string    `json:"serverId"`
	Kind     string    `json:"kind"`
	Title    string    `json:"title"`
	Actor    string    `json:"actor"`
	Status   JobStatus `json:"status"`

	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`

	Changed   int    `json:"changed"`
	Unchanged int    `json:"unchanged"`
	Skipped   int    `json:"skipped"`
	Failed    int    `json:"failed"`
	Error     string `json:"error,omitempty"`
}

// DurationMs is nil while the job is still running.
func (j Job) DurationMs() *int64 {
	if j.FinishedAt == nil {
		return nil
	}
	ms := j.FinishedAt.Sub(j.StartedAt).Milliseconds()
	return &ms
}

// Event is one line of a job's history: a step boundary or a log line.
type Event struct {
	ID        int64     `json:"id"`
	JobID     string    `json:"jobId"`
	Seq       int       `json:"seq"`
	Type      string    `json:"type"`
	StepID    string    `json:"stepId,omitempty"`
	StepTitle string    `json:"stepTitle,omitempty"`
	Level     string    `json:"level,omitempty"`
	Message   string    `json:"message,omitempty"`
	Outcome   string    `json:"outcome,omitempty"`
	At        time.Time `json:"at"`
}
