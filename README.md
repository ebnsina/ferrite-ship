# Ferrite Ship

Connect a fresh Ubuntu server, get it set up safely in minutes, then manage it
from a browser — files, services, updates, storage — without memorising another
`apt` incantation.

> **Status: early, but deployable.** Single-tenant, and the roadmap below is
> longer than what exists. See [Known limits](#known-limits) first.

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
- **Install what you actually need** — twelve tools: PostgreSQL, Redis,
  ClickHouse, Meilisearch, Qdrant, NATS, RabbitMQ, MinIO, Grafana, Keycloak,
  MediaMTX and Traefik. Each gets a generated password and a connection string
  you can copy. Databases listen on loopback only and are reached over an SSH
  tunnel the dashboard writes out for you; a media server is public, because a
  stream nobody can watch is not a stream.
- **Query them without leaving the page.** Every installed database has a
  console: SQL for PostgreSQL and ClickHouse, commands for Redis, with
  ready-made queries to start from and somewhere to keep your own.
- **Deploy your own code.** Point it at a git repository — public, or private
  with a deploy key. A Dockerfile is used if there is one; otherwise the
  project is read and built, which covers Rust, Go, Python and Node. Give it a
  domain and it is served over HTTPS with a certificate that renews itself.
- **Back it up somewhere else.** PostgreSQL, Redis and ClickHouse stream to
  any S3-compatible storage, and restore from it. Deliberately not the same
  disk. Backups can run on a daily or weekly schedule with a retention count,
  and old copies are removed only after the new one has arrived.
- **Be told when something breaks.** A health check every five minutes, alerts
  that are raised once and cleared when the condition ends, and optional email.
  A "what needs attention" page collects anything currently wrong and anything
  that failed while nobody was watching, with the error text in full.
- **Take them away again.** Removing keeps your data unless you ask for it to
  go, and says which it is doing before you agree.
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

To create an account from the command line instead — useful for the first one
on a headless box — `adduser` generates a password and prints it once:

```bash
go run ./cmd/ferrite-ship adduser you@example.com
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

## Running it on a server

The control plane holds credentials for every server you connect, so it should
never face the internet directly. Both routes below put a proxy in front and
keep the app itself on a private network or loopback.

### With Docker

```bash
export FERRITE_DOMAIN=ferrite.example.com
export FERRITE_SECRET_KEY=$(docker compose run --rm ferrite-ship genkey)

docker compose up -d --build
```

Caddy gets a certificate on its own; point the domain at the machine first.
The dashboard and API are served from one origin, so no cross-origin
permission is needed at all.

Keep `FERRITE_SECRET_KEY` somewhere safe. It decrypts every stored credential,
and losing it means reconnecting every server.

### Without Docker

`deploy/ferrite-ship.service` runs the binary under systemd, bound to
loopback, with the sandboxing options a process like this should have. Put
Caddy, nginx or Traefik in front for TLS.

Whichever proxy you use, **disable response buffering**. Job logs stream over
Server-Sent Events and the terminal is a WebSocket; a buffering proxy makes
logs arrive in a lump at the end and the shell appear frozen.
`deploy/Caddyfile` shows the setting.

---

## Configuration

| Variable | Required | Meaning |
|---|---|---|
| `FERRITE_ADDR` | yes | Listen address. Keep it on loopback (`127.0.0.1:8080`) until there is authentication — this API holds root credentials for every connected server |
| `FERRITE_DATABASE_PATH` | yes | SQLite file; created on first run |
| `FERRITE_SECRET_KEY` | yes | Base64 of 32 bytes. Seals stored credentials — **lose it and every stored credential becomes unreadable** |
| `FERRITE_ALLOWED_ORIGIN` | yes | Origin allowed to call the API cross-site, or `none` |
| `FERRITE_WEB_DIR` | yes | Built dashboard to serve, or `none` for API-only |
| `FERRITE_PUBLIC_URL` | yes | Where the dashboard is reachable, for links in alert email, or `none` |
| `FERRITE_SMTP_URL` | yes | `smtp://user:pass@host:587`, or `none` to send no mail |
| `FERRITE_MAIL_FROM` | with SMTP | Address alerts are sent from |
| `FERRITE_ACME_ENDPOINT` | yes | `staging` or `production`. No default: production allows five duplicate certificates per week |
| `FERRITE_GITHUB_APP_ID` | yes | Numeric app id, or `none` to deploy only from a pasted git URL |
| `FERRITE_GITHUB_APP_SLUG` | with GitHub | The name in the app's URL |
| `FERRITE_GITHUB_PRIVATE_KEY` | with GitHub | `base64 < your-app.private-key.pem` |
| `FERRITE_GITHUB_WEBHOOK_SECRET` | with GitHub | Without it, anybody could ask for a deploy |

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

### The catalogue

Each installable tool is a Docker Compose project under `/opt/ferrite/<id>`, so
one mechanism installs, inspects, restarts and removes all of them, and you can
read the file on your own server to see exactly what is running. Adding a tool
is a data change in `internal/catalog` rather than new code anywhere else.

The compose file holds no secret: generated passwords go in an `.env` file
beside it, written `0600`, so the part describing what runs can be read without
exposing the part that gets you in. Those passwords are also masked in the job
log, which is persisted and shown in a browser — `Session` redacts at that
boundary rather than asking each step to remember.

Databases publish on `127.0.0.1` only. Note that Docker's published ports skip
ufw entirely, so the bind address is the control here and the firewall rules
are a statement of intent.

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
│   ├── alerts/         Saying a thing once, and saying when it stops
│   ├── api/            HTTP handlers, SSE, static hosting
│   ├── apierr/         Every user-facing error, in one catalogue
│   ├── auth/           Passwords and sessions
│   ├── catalog/        The installable tools, and their playbooks
│   ├── config/         Environment loading and validation
│   ├── console/        Running a query against an installed database
│   ├── deploy/         Building and running your own applications
│   ├── dialer/         The one way an SSH connection is opened
│   ├── executor/       Command transport (ssh, demo)
│   ├── facts/          Reading what a server is and how busy it is
│   ├── files/          Browsing and editing files over SFTP
│   ├── github/         Acting as a GitHub App: JWTs and installation tokens
│   ├── insight/        What is using the disk, and what can be reclaimed
│   ├── notify/         The words an alert email is made of, and sending it
│   ├── runner/         Job execution and the event bus
│   ├── scheduler/      Backups nobody pressed a button for
│   ├── secret/         Credential sealing
│   ├── services/       systemd units and their logs
│   ├── steps/          The step engine and the baseline playbook
│   ├── store/          SQLite persistence
│   ├── terminal/       Interactive shells over SSH
│   └── watch/          The only thing that looks at a server unasked
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
| `GET` | `/v1/servers/{id}/apps` | Applications on a server |
| `POST` | `/v1/servers/{id}/apps` | Add an application |
| `PUT` | `/v1/apps/{app}` | Change one |
| `POST` | `/v1/apps/{app}/deploy` | Build and run it |
| `DELETE` | `/v1/apps/{app}` | Stop it and remove its route |
| `GET` | `/v1/backups/destination` | Where backups go (never returns keys) |
| `PUT` | `/v1/backups/destination` | Set it |
| `GET` | `/v1/servers/{id}/tools/{tool}/backups` | Backups taken |
| `POST` | `/v1/servers/{id}/tools/{tool}/backups` | Take one |
| `POST` | `/v1/backups/{backup}/restore` | Put one back |
| `POST` | `/v1/servers/{id}/tools/{tool}/query` | Run a query |
| `GET` | `/v1/servers/{id}/tools/{tool}/queries` | Saved queries |
| `GET` | `/v1/catalog` | Everything installable |
| `GET` | `/v1/servers/{id}/tools` | The catalogue, with what is installed here |
| `POST` | `/v1/servers/{id}/tools` | Install a tool, or repair one |
| `DELETE` | `/v1/servers/{id}/tools/{tool}` | Remove one (`?purge=true` deletes its data) |
| `GET` | `/v1/servers/{id}/tools/{tool}/connection` | How to connect, credentials included |
| `GET` | `/v1/servers/{id}/services` | List services |
| `POST` | `/v1/servers/{id}/services/{unit}/actions` | Start, stop or restart a service |
| `GET` | `/v1/servers/{id}/services/{unit}/logs` | Read a service's journal |
| `GET` | `/v1/metrics` | Fleet metrics |
| `PUT` | `/v1/servers/{id}/domain` | Point a domain at a server |
| `GET` | `/v1/servers/{id}/tools/{tool}/schedule` | When a backup runs on its own |
| `PUT` | `/v1/servers/{id}/tools/{tool}/schedule` | Set that |
| `DELETE` | `/v1/servers/{id}/tools/{tool}/schedule` | Stop it |
| `GET` | `/v1/notifications` | What this account asked to be told |
| `PUT` | `/v1/notifications` | Change it |
| `POST` | `/v1/notifications/test` | Prove the whole mail path |
| `GET` | `/v1/alerts` | Conditions that are true right now |
| `GET` | `/v1/problems` | Those, plus runs that did not finish |
| `GET` | `/v1/github/status` | Whether GitHub is configured, and connected |
| `POST` | `/v1/github/connect` | Where to send the browser to install the app |
| `GET` | `/v1/github/callback` | Where GitHub sends it back |
| `DELETE` | `/v1/github/installations/{id}` | Forget a connection |

Everything except `/v1/health` and `/v1/auth/*` requires a session cookie.

Errors share one envelope: `{ "code", "message", "action", "request_id" }`.
Every message and action is defined in `internal/apierr`, and the dashboard
shows what it is sent — the wording exists in one place.

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

- **Single account, no roles.** One person, no teams, no permissions. Servers
  are scoped to their owner and enforced in SQL, but row-level security waits
  for PostgreSQL — SQLite has none.
- **Trust on first use.** A server's SSH identity is recorded the first time it
  is reached and checked on every connection after, but that first connection
  is taken on trust. Verifying a fingerprint out of band would close it.
- **Sign-in throttling is per process.** Restarting clears it, and behind a
  proxy it counts the proxy rather than the caller.
- **Single tenant.** No organisations, users or roles.
- **SSH, not an agent.** Servers behind NAT are unreachable, and the control
  plane holds credentials it would rather not have.
- **Two things want ports 80 and 443.** Deployed applications are fronted by
  Caddy (`internal/deploy`), and the Traefik tool in the catalogue binds the
  same ports. Installing both on one server means the second fails to start.
  Unifying them behind one ingress is the fix, and until then a server should
  have applications *or* Traefik-routed tools, not both.
- **Nothing has ever issued a certificate.** The whole routing layer — Traefik,
  the shared network, the ACME configuration, every `Host()` rule — is verified
  only by unit tests and by `docker compose config` rendering the files. It has
  never run against a real domain. `FERRITE_ACME_ENDPOINT=staging` exists so the
  first attempt is repeatable rather than rate limited.
- **No tool has been installed on a real server.** Compose files are validated,
  not executed. That covers a malformed file and not a wrong image argument.
- **`ssh-harden` has never applied.** It skips correctly on a
  password-authenticated server, so the guard is proven and the code it guards
  is not.

## Roadmap

1. Prove the routing layer against a real domain, then unify Caddy and Traefik
   behind one ingress
2. Finish the GitHub App: pick a repository from a list, then deploy on push
3. A long-lived agent, so credentials are not stored and NAT stops mattering
4. Organisations and roles, on PostgreSQL with row-level security
5. Build logs and rollback for deployments
