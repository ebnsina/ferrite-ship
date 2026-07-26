# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What this is

Ferrite Ship is a control plane for Ubuntu servers: connect one over SSH, run an
idempotent hardening playbook against it, and manage it from a browser. The
repository root is a Go module; `web/` is a separate SvelteKit project with its
own pnpm setup.

Read `README.md` for the product overview and `docs/product-scope.md` for the
longer-range plan. **`docs/` and `data/` are gitignored on purpose** — internal
planning and local state, never committed.

## Commands

```bash
# Go (from repo root)
go build ./...
go vet ./...
gofmt -l .                       # must print nothing
go run ./cmd/ferrite-ship        # needs .env loaded
go run ./cmd/ferrite-ship genkey # generate FERRITE_SECRET_KEY

# Run the control plane
set -a && . ./.env && set +a && go run ./cmd/ferrite-ship

# Web (from web/)
pnpm check                       # svelte-check + TypeScript — must be 0 errors
pnpm build
pnpm dev --port 5173 --strictPort
```

Always run `go vet ./...` and `pnpm check` before considering work done.

## Non-negotiable conventions

These were set by the repository owner. Follow them without being asked.

1. **Environment variables fail fast.** No defaults, no fallbacks, for anything
   that decides where data goes or how it is protected. Validate at load
   (`internal/config`, `web/src/lib/config/env.ts`) and throw. Turning a feature
   off is written down explicitly with the literal `none` — an unset variable is
   always an error, never a quiet "no".
2. **Never assume a library API.** Check current docs (context7, `npm view`,
   `go doc`) before writing framework code. This repo already caught
   `lucide-svelte` being renamed to `@lucide/svelte`.
3. **All formatting goes through `Intl`** — numbers, bytes, dates, relative
   time, durations. See `web/src/lib/utils/format/`. Never hand-roll a formatter.
4. **Small, composable files.** No ~200-line components. Split content, styles
   and logic out of Svelte files; `$lib/content/*` holds copy, `$lib/domain/*`
   holds mapping logic.
5. **Handle every error path.** 404, 500, network, empty and loading states are
   first-class. `AppError` normalises anything catchable; `createResource` gives
   async views loading/error/retry.
6. **UI copy is plain language, no technical jargon.** "Not responding", not
   "unreachable". Every dashboard section says what it shows and how to read it.
7. **Use icons generously** (`@lucide/svelte`, per-icon imports). Anything
   AI-related uses **`astroid`** — the four-pointed star, which lucide itself
   tags for artificial intelligence. Not `Sparkles` (already used for the
   console's query presets, where it does not mean AI), not `Bot`, not `Brain`.
   Note the spelling: "astroid", not "asteroid".
8. **Never invent data.** No fabricated metrics, adoption numbers, testimonials
   or trends. Where history is missing, return an empty series and let the UI
   omit the chart. Placeholder content must be marked as such in its module.

## Architecture invariants

Breaking these costs far more later than the shortcut saves now.

**Step engine (`internal/steps`).** Every step implements `Check` and `Apply`.
`Check` asks whether the desired state already holds; `Apply` establishes it.
Running the baseline twice must report zero changes. Most steps are declarative
`shellStep` values — prefer that over bespoke Go.

**Executor boundary (`internal/executor`).** Steps talk to an `Executor`
interface, never to SSH directly. Two implementations exist: real SSH, and
`demoexec`, a simulated Ubuntu machine that models state so the pipeline can be
exercised without a VPS. An agent transport is planned; adding it must not
require changing a single step.

**One way to reach a server (`internal/dialer`).** Everything that opens an SSH
connection goes through it. Four packages used to resolve the server, decrypt
its credentials and dial independently; host key checking would have had to be
added to all four and kept in step. Do not reintroduce a second path.

**Safety preconditions.** Steps declare `skipIf`. `ssh-harden` refuses to
disable password logins when no key is installed anywhere — an unrecoverable
server is worse than a slightly weaker one. Preserve that reasoning in any new
step that could lock someone out.

**Job events (`internal/runner`).** Events are persisted *before* being
published to the bus, so a reconnecting SSE client resumes by sequence number
with nothing lost. Do not reorder that.

**Errors have one home (`internal/apierr`).** Every failure a person can see —
its code, HTTP status, what happened, and what to do next — is defined in that
catalogue. Handlers pick an entry; they do not write sentences or choose status
codes. Infrastructure wording (SSH, SFTP, systemd) is interpreted once in
`apierr.From`. The web client renders the `message` and `action` it is sent
rather than keeping a second copy, so the words live in one repository. The
only copy the browser owns is for failures the API could not describe because
it never answered: network, timeout, parse, config.

**Credentials (`internal/secret`).** SSH passwords and keys are sealed with
AES-256-GCM before storage. Never log them, never put them in a response type —
`api.serverView` is deliberately separate from `store.Server` for this reason.

**Storage (`internal/store`).** Plain SQL behind narrow methods. SQLite today;
the move to PostgreSQL when multi-tenancy lands should touch this package alone.

**Ownership is a query parameter, never a filter applied afterwards.** Every
store method that reaches a server, job or fleet sample takes a `userID` and
puts it in the `WHERE` clause. That is deliberate: the compiler then refuses
any call site that has not established who is asking, so a scoping hole cannot
be introduced by forgetting a check. `internal/store/scoping_test.go` exists to
fail loudly if a query ever stops carrying it.

**Row-level security is the plan, not the present.** SQLite has none — RLS is a
PostgreSQL feature. When multi-tenancy arrives, add RLS policies as a second
layer *underneath* the application scoping rather than instead of it: the
scoping tests then keep passing and the database enforces the same rule
independently. Until then those tests are the only guarantee, so do not weaken
them.

## Web conventions

**Design tokens are three-tier** (`web/src/lib/styles/tokens.css`): primitives →
semantic → component. Components reference *only* the semantic tier
(`bg-surface`, `text-content-muted`). Brand colour is deep blue.

**The brand is defined once, in `tokens.css`, and generated everywhere else.**
`pnpm brand` derives the favicon and `src/lib/theme/brand.generated.ts` from
the token ramp; `pnpm check` runs it in `--check` mode and fails if they have
drifted. Anything that cannot read a CSS variable — the favicon, `<meta>`
colours, the terminal's canvas palette — reads the generated module. Never
hand-write a brand hex: the mark stayed lime through a cyan brand and a pink
one precisely because it was copied by hand.

**The accent is two tokens, not one.** `accent` is the brand as *ink* — text
and icons, so it must be legible on the surface behind it, which means bright
on dark and deep on light. `accent-solid` is the brand as a *fill* behind text
(`bg-accent-solid text-accent-content`), so it must be dark enough to carry
white. A deep blue carries white text at 6:1, which is why it was chosen: lime,
cyan and pink are all vivid only when *light*, so each forced dark labels or a
muddy fill. Never use `bg-accent`.

**Radius is a role-based scale**, never one value everywhere:
`rounded-pill` (badges, meters), `rounded-control` (buttons), `rounded-field`
(inputs), `rounded-tile` (small tiles, nav, icon chips), `rounded-card-sm` /
`rounded-card` / `rounded-panel` (surfaces, by size).

**Radius scales with the element**, and that applies within a role as well as
between them. `Card` takes `size="sm|md|lg"` and `Button` sets its own corner
per size, because the same number does not read the same way at two heights: a
2rem corner that looks generous on a full-width panel turns a 56px banner into
a lozenge, since the curve then eats most of the edge it is meant to soften.
Pick the size, do not override the radius.

**Theming is scoped.** Tokens are declared against `[data-theme]`, which matches
any element. Marketing is dark; the dashboard is wrapped in `ThemeScope` and is
light. They are independent by design (`web/src/lib/theme/theme.svelte.ts`).

**Design system is custom** — no component library. Unstyled behaviour
primitives are fine; reimplementing accessibility is not.

**Status vocabulary is shared.** One set of tones (`ok`/`warn`/`error`/`info`/
`pending`) in `$components/ui/tone.ts`, mapped to domain states in
`$lib/domain/status.ts`. Never encode status by colour alone.

**Adapter is `adapter-static`.** Marketing routes prerender; `/dashboard` sets
`ssr = false` and is served from the `200.html` fallback. The Go API is the
backend — do not add `+page.server.ts` or form actions.

**Data access goes through `DashboardRepository`** (`web/src/lib/data/`). No
component calls `fetch`. `PUBLIC_DATA_SOURCE` selects the mock or API
implementation.

## Git

- Author every commit as `ebnsina <ebnsina.me@gmail.com>` (already set in the
  repo's git config).
- **Do not add a `Co-Authored-By: Claude` trailer**, or any other identity.
- Remote uses the `github-es` SSH host alias.
- `docs/` and `data/` stay out of the repository.
- Commit messages: explain *why*, not just what. Wrap at ~76 columns.

## Gotchas

- **Check where an image keeps its data before pinning a volume.** PostgreSQL
  18 moved `PGDATA` to `/var/lib/postgresql/18/docker` and the image's `VOLUME`
  with it. Mounting the familiar `/var/lib/postgresql/data` leaves the database
  writing inside the container, and every row disappears the next time it is
  recreated — with no error at any point.
- **`ufw --force` belongs only on `enable`, `reset` and `delete`.** On `allow`
  it is rejected as `ERROR: Invalid syntax`. On `delete` it is required, or ufw
  waits at a `Proceed with operation (y|n)?` prompt nobody can answer.
- **Docker's published ports bypass ufw entirely**, so the firewall is not what
  keeps a database private — binding it to `127.0.0.1` in the compose file is.
  Treat the ufw rules as intent, and the bind address as the control.
- **Do not delete-and-recreate a source file to rewrite it.** Vite keeps
  serving the old transform, so the browser shows stale code while the file on
  disk is correct — and you end up debugging code that is already right. Edit
  in place instead. If a change genuinely appears not to take effect, compare
  what the dev server returns against disk before touching the code:
  `curl -s http://localhost:5173/src/path/File.svelte | grep something`.
  This has cost time three times.
- **Always start Vite with `--strictPort`.** Without it, a stale process keeps
  the port and the new server silently moves to 5174, so you end up testing the
  old build.
- **zsh does not word-split unquoted variables.** `for f in $files` iterates
  once with the whole string. Use arrays or explicit lists in Bash calls.
- **Never use `path` as a shell variable in zsh.** It is tied to `PATH`, so
  `for path in ...` wipes the shell's PATH and every later command fails with
  "command not found". Cost time once already.
- **`cd` persists between Bash tool calls.** Prefer absolute paths.
- **Vite reads `process.env` in preference to `.env`.** A Dockerfile `ARG` with
  an empty default is present in the build environment as an empty string and
  silently shadows the file, so writing `.env` and then building fails with
  "PUBLIC_API_BASE_URL is required". Pass build-time config as environment on
  the `pnpm build` command instead of writing a file.
- **`.dockerignore` is load-bearing.** Without it `COPY web/ ./` drops the
  host's macOS `node_modules` over the image's Linux ones, and pnpm aborts
  trying to purge a directory that no longer matches the lockfile.
- SQLite runs with `MaxOpenConns(1)`; it tolerates one writer, and more
  connections buy contention rather than speed.
