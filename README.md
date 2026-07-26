# Ferrite Ship

Connect a fresh Ubuntu server, get it set up safely in minutes, then manage it
from a browser — files, services, updates, storage — without memorising another
`apt` incantation.

> **Status: early.** SSH host keys are trusted on first use and the product is
> single-tenant. See [Known limits](#known-limits) before pointing this at
> anything you care about.

---

## What works today

- **Connect a server** over SSH (password or private key), or spin up a
  simulated one to try things out.
- **Run the baseline playbook** — package updates, an admin account, SSH
  hardening, firewall, fail2ban, automatic security updates, timezone, swap and
  sysctl tuning.
- **Watch it happen live.** Every command and its output streams to the
  dashboard as it runs.
- **Run it again safely.** A second run of the same playbook changes nothing
  and says so.
- **Open a real shell in the browser** — a PTY over SSH, with resize, colour
  and full-screen programs.
- **Browse and edit files** over SFTP: navigate, open a config, save it back
  with its permissions intact, download, delete.
- **See and control services** — what is running, start, stop, restart, and
  read the journal. Units that would cut off access refuse to be stopped.
- **See your fleet** — status, CPU, memory, storage and uptime, gathered from
  the machines themselves.

## What it looks like

Running the baseline against a fresh machine:

```
> Refresh the list of available updates      -> CHANGED
> Install the essentials                     -> CHANGED
> Create your everyday login account         -> CHANGED
> Turn off root and password logins          -> SKIPPED
> Close every door except the ones you use   -> CHANGED
> Block repeated login attempts              -> CHANGED
> Install security updates automatically     -> CHANGED
> Set the clock to UTC                       -> CHANGED
> Add breathing room when memory runs short  -> CHANGED
> Apply sensible network settings            -> CHANGED

DONE: 9 changed, 0 already fine, 1 not needed
```

Immediately running it a second time:

```
DONE: Nothing needed changing — all 10 checks already passed
```

That second result is the point of the whole design. Steps converge on a
desired state rather than executing blindly, so re-running is always safe.

---

## Quick start

### Requirements

- Go 1.25+
- Node 22+ and pnpm 10+ (for the dashboard)

### 1. Configure

```bash
cp .env.example .env
go run ./cmd/ferrite-ship genkey     # paste into FERRITE_SECRET_KEY
```

Every variable in `.env` is required. The process refuses to start if one is
missing — a misconfigured deploy should fail loudly rather than quietly run
against the wrong database.

### 2. Run the control plane

```bash
mkdir -p data
set -a && . ./.env && set +a
go run ./cmd/ferrite-ship
```

It listens on `127.0.0.1:8080` and creates the SQLite database on first run.

### 3. Run the dashboard

```bash
cd web
cp .env.example .env     # defaults point at http://localhost:8080
pnpm install
pnpm dev
```

Open <http://localhost:5173>. The first visit asks you to create an account;
after that it is a sign-in page, and only one account can be created this way.

Forgotten the password? There is no reset over the network — that would be a
second way in. Instead, prove you control the machine:

```bash
set -a && . ./.env && set +a
go run ./cmd/ferrite-ship reset-account   # then create a new one in the browser
```

---

## Try it without a server

You do not need a VPS to see the whole thing work. Create a simulated machine:

```bash
curl -s -X POST localhost:8080/v1/servers \
  -H 'content-type: application/json' \
  -d '{"name":"demo-1","connectionKind":"demo"}'
```

Then press **Connect a server → Run setup** in the dashboard, or:

```bash
curl -s -X POST localhost:8080/v1/servers/<id>/jobs \
  -H 'content-type: application/json' -d '{"kind":"baseline"}'
```

The simulated machine runs the *real* steps through the *real* runner and models
its own state — only the machine on the other end is imaginary. Run the job
twice and the second run reports everything unchanged, exactly as a real server
would.

## Connect a real server

Ubuntu 22.04 or 24.04, reachable over SSH, with an account that can `sudo`
without a password prompt (root works):

```bash
curl -s -X POST localhost:8080/v1/servers \
  -H 'content-type: application/json' \
  -d '{
        "name": "edge-1",
        "connectionKind": "ssh",
        "host": "203.0.113.10",
        "port": 22,
        "user": "root",
        "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\n..."
      }'
```

Credentials are encrypted with AES-256-GCM under `FERRITE_SECRET_KEY` before
they are stored, so a copy of the database is not a copy of every server.

**Use a throwaway server the first time.** The baseline changes SSH
configuration and enables a firewall.

---

## Configuration

| Variable | Required | Meaning |
|---|---|---|
| `FERRITE_ADDR` | yes | Listen address. Keep it on loopback (`127.0.0.1:8080`) until there is authentication — this API holds root credentials for every connected server |
| `FERRITE_DATABASE_PATH` | yes | SQLite file; created on first run |
| `FERRITE_SECRET_KEY` | yes | Base64 of 32 bytes. Seals stored credentials — **lose it and every stored credential becomes unreadable** |
| `FERRITE_ALLOWED_ORIGIN` | yes | Origin allowed to call the API cross-site, or `none` |
| `FERRITE_WEB_DIR` | yes | Built dashboard to serve, or `none` for API-only |

Switching a feature off is written down explicitly with `none`. An unset
variable is always an error, never a quiet default.

Dashboard variables live in `web/.env` and follow the same rule.

---

## How it works

### The step engine

Every unit of work implements two halves:

```go
type Step interface {
	Check(ctx, *Session) (done bool, err error)  // is this already true?
	Apply(ctx, *Session) error                   // make it true
}
```

`Check` runs first. If it says the work is done, `Apply` is skipped and the step
reports *unchanged*. That is what makes a playbook safe to run repeatedly, and
it is the single most important idea in the codebase.

Steps also declare a precondition. `ssh-harden` refuses to disable password
logins when no key is installed anywhere on the machine — an unrecoverable
server is far worse than a slightly weaker one, so it skips with an explanation.

### Transport

Steps are written against an `Executor` interface, not against SSH:

```go
type Executor interface {
	Run(ctx context.Context, cmd string) (Result, error)
}
```

Two implementations exist today — real SSH, and the simulated machine. A
long-lived agent is planned, and the step engine will not change when it lands.

### Jobs and logs

A job runs in the background and emits events. Each event is written to the
database *before* it is published to subscribers, so a log viewer that drops its
connection resumes by sequence number (`Last-Event-ID`) with nothing lost.

### Trends

Fleet metrics are drawn from recorded snapshots. Where there is not enough
history the API returns an empty series and the dashboard omits the sparkline,
rather than drawing a trend that was never measured.

---

## Layout

```
.                       Go module — the control plane
├── cmd/ferrite-ship/   Entry point
├── internal/
│   ├── api/            HTTP handlers, SSE, static hosting
│   ├── auth/           Passwords and sessions
│   ├── config/         Environment loading and validation
│   ├── executor/       Command transport (ssh, demo)
│   ├── facts/          Reading what a server is and how busy it is
│   ├── files/          Browsing and editing files over SFTP
│   ├── runner/         Job execution and the event bus
│   ├── secret/         Credential sealing
│   ├── services/       systemd units and their logs
│   ├── steps/          The step engine and the baseline playbook
│   ├── store/          SQLite persistence
│   └── terminal/       Interactive shells over SSH
└── web/                SvelteKit dashboard (own pnpm project)
```

SQLite keeps the MVP to a single binary with nothing to run alongside it. The
queries are plain SQL behind narrow methods, so moving to PostgreSQL when
multi-tenancy arrives is a change to `internal/store` alone.

## API

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/health` | Liveness |
| `GET` | `/v1/auth/status` | Whether setup is needed and who is signed in |
| `POST` | `/v1/auth/setup` | Create the first and only account |
| `POST` | `/v1/auth/login` | Sign in |
| `POST` | `/v1/auth/logout` | Sign out |
| `GET` | `/v1/servers` | List servers |
| `POST` | `/v1/servers` | Connect a server |
| `DELETE` | `/v1/servers/{id}` | Remove a server |
| `POST` | `/v1/servers/{id}/jobs` | Start a job |
| `GET` | `/v1/jobs/{id}` | Job status |
| `GET` | `/v1/jobs/{id}/events` | Live job log (SSE) |
| `GET` | `/v1/activity` | Recent jobs |
| `GET` | `/v1/servers/{id}/terminal` | Interactive shell (WebSocket) |
| `GET` | `/v1/servers/{id}/files` | List a directory |
| `DELETE` | `/v1/servers/{id}/files` | Delete a file or empty folder |
| `GET` | `/v1/servers/{id}/files/content` | Read a text file |
| `PUT` | `/v1/servers/{id}/files/content` | Save a text file |
| `GET` | `/v1/servers/{id}/files/download` | Download a file |
| `GET` | `/v1/servers/{id}/services` | List services |
| `POST` | `/v1/servers/{id}/services/{unit}/actions` | Start, stop or restart a service |
| `GET` | `/v1/servers/{id}/services/{unit}/logs` | Read a service's journal |
| `GET` | `/v1/metrics` | Fleet metrics |

Everything except `/v1/health` and `/v1/auth/*` requires a session cookie.

Errors share one envelope: `{ "code", "message", "request_id" }`.

---

## Development

```bash
go build ./...          # compile
go vet ./...            # vet
gofmt -l .              # formatting (should print nothing)

cd web
pnpm check              # svelte-check + TypeScript
pnpm build              # production build
```

To serve the dashboard from the Go binary instead of Vite:

```bash
cd web && pnpm build && cd ..
FERRITE_WEB_DIR=./web/build FERRITE_ALLOWED_ORIGIN=none go run ./cmd/ferrite-ship
```

---

## Known limits

These are the reasons this is not production software yet:

- **Single account, no roles.** One person, no teams, no permissions.
- **No rate limiting on sign-in.** Argon2id makes guessing expensive, but
  nothing stops an attacker trying repeatedly.
- **SSH host keys are trusted on first use** and never verified again, which
  leaves the first connection open to interception. Pinning is required before
  this is used across an untrusted network.
- **Single tenant.** No organisations, users or roles.
- **SSH, not an agent.** Servers behind NAT are unreachable, and the control
  plane holds credentials it would rather not have.
- **One playbook.** Baseline setup only — no application deploys, no services,
  no backups yet.

## Roadmap

1. Connect-a-server flow and live log view in the dashboard
2. A long-lived agent, so credentials are not stored and NAT stops mattering
3. Authentication, organisations and roles
4. Day-2 operations: terminal, files, services, packages, disks
5. One-click services, ingress with automatic TLS, backups
