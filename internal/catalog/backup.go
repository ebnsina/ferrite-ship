package catalog

import "github.com/ebnsina/ferrite-ship/internal/steps"

// Backup describes how a tool is copied off a server and put back.
//
// Both halves are streams rather than files: the dump writes to stdout and the
// restore reads from stdin, so the data goes straight to and from object
// storage without ever landing on the server's disk. A database large enough
// to be worth backing up is often large enough that a second copy of it does
// not fit beside the first.
type Backup struct {
	// Extension names the artefact, after the timestamp.
	Extension string
	// Dump writes the backup to stdout and nothing else to it. Anything
	// chatty has to be sent to stderr or the archive is corrupt.
	Dump string
	// RestoreBefore runs before the data arrives — stopping a service that
	// will not accept a replacement while it is running, say.
	//
	// Separate from Restore rather than chained onto it with `&&`, and that is
	// not a stylistic choice: `stream | a && b` binds the pipe to `a`, so the
	// backup would be fed to the command that stops the container and `b`
	// would write an empty file. Which is precisely what happened.
	RestoreBefore []string
	// Restore reads the backup from stdin, and is the only part that does. It
	// must be safe to run over a database that already has data in it.
	Restore string
	// RestoreAfter runs once the data is in place, for tools that need coaxing
	// into reading it. Separate commands rather than one blob so each shows as
	// its own line in the log.
	RestoreAfter []string
	// Warning is shown before restoring, in the words the person reads. It
	// describes what they are about to overwrite.
	Warning string
}

// Supported reports whether this tool can be backed up yet.
func (t Tool) Supported() bool { return t.backup != nil }

// BackupSpec returns the tool's backup commands.
func (t Tool) BackupSpec() (Backup, bool) {
	if t.backup == nil {
		return Backup{}, false
	}
	return *t.backup, true
}

// exec runs a command inside a tool's running container.
//
// Through compose rather than `docker exec <name>`: the container name is
// derived from the project and would change if compose ever altered its naming,
// whereas the service name is ours and is written down in the file.
func inContainer(id, service, command string) string {
	return compose(id, "exec", "-T", service, "sh", "-c", steps.Quote(command))
}

// postgresBackup uses pg_dump's custom format, which is compressed and can be
// restored selectively, rather than a plain SQL file.
var postgresBackup = &Backup{
	Extension: "dump",
	Dump:      inContainer("postgres", "postgres", `PGPASSWORD="$POSTGRES_PASSWORD" pg_dump -U ferrite -d app -Fc`),
	// --clean --if-exists drops each object before recreating it, so restoring
	// over an existing database replaces it rather than colliding with it.
	// Without --if-exists the first missing object aborts the whole restore.
	Restore: inContainer("postgres", "postgres", `PGPASSWORD="$POSTGRES_PASSWORD" pg_restore -U ferrite -d app --clean --if-exists --no-owner`),
	Warning: "Everything currently in this database is replaced by the copy you are restoring. Anything written since that backup was taken is lost.",
}

// redisBackup asks Redis for a point-in-time snapshot rather than copying the
// file underneath it, which would catch a half-written page.
var redisBackup = &Backup{
	Extension: "rdb",
	Dump: inContainer("redis", "redis",
		`redis-cli -a "$REDIS_PASSWORD" --no-auth-warning --rdb /tmp/ferrite.rdb >/dev/null 2>&1 && cat /tmp/ferrite.rdb && rm -f /tmp/ferrite.rdb`),
	// Redis cannot be told to load a snapshot while it is running, so this
	// stops it and replaces the file.
	//
	// The append-only log has to go with it, and removing it is not enough on
	// its own: with appendonly on, Redis does not read dump.rdb at boot at all.
	// Finding no log it writes a fresh empty one, comes up with nothing, and
	// reports success — which is exactly what happened the first time this was
	// tested. RestoreAfter below is what actually loads the snapshot.
	RestoreBefore: []string{compose("redis", "stop", "redis")},
	Restore: `docker run --rm -i -v ferrite-redis_data:/data alpine:3.21 ` +
		`sh -c 'rm -rf /data/appendonlydir /data/dump.rdb && cat > /data/dump.rdb'`,
	RestoreAfter: []string{
		// A throwaway Redis with the log switched off, which therefore does
		// read the snapshot. No password: this one is not published anywhere.
		`docker rm -f ferrite-redis-restore >/dev/null 2>&1 || true`,
		`docker run -d --name ferrite-redis-restore -v ferrite-redis_data:/data redis:8-trixie redis-server --appendonly no --dir /data`,
		`for i in $(seq 30); do docker exec ferrite-redis-restore redis-cli ping >/dev/null 2>&1 && break; sleep 1; done`,
		// Turning the log back on rebuilds it from what is now in memory,
		// which is the restored data.
		`docker exec ferrite-redis-restore redis-cli CONFIG SET appendonly yes`,
		`for i in $(seq 60); do docker exec ferrite-redis-restore redis-cli INFO persistence 2>/dev/null | tr -d '\r' | grep -q 'aof_rewrite_in_progress:0' && break; sleep 1; done`,
		`docker exec ferrite-redis-restore redis-cli SHUTDOWN NOSAVE >/dev/null 2>&1 || true`,
		`docker rm -f ferrite-redis-restore >/dev/null 2>&1 || true`,
		compose("redis", "start", "redis"),
	},
	Warning: "Redis is stopped for a moment while it is replaced, so anything using it will be briefly unable to connect. Everything currently stored is replaced.",
}

// clickhouseBackup uses ClickHouse's own BACKUP statement rather than copying
// files out of /var/lib/clickhouse.
//
// The data directory is a live thing — parts are being merged and replaced
// while it is read — so a copy taken from underneath it is a copy of a moment
// that never existed. BACKUP is consistent, and produces one zip.
//
// It writes to a disk rather than to stdout, because there is no form of the
// statement that streams. So the file is assembled on the backups volume,
// handed to stdout, and deleted — the only tool here that needs a moment of
// disk on the server, and the reason it has a volume of its own.
var clickhouseBackup = &Backup{
	Extension: "zip",
	// BACKUP refuses to overwrite, so anything left by a run that was
	// interrupted has to go first or every later backup fails.
	Dump: inContainer("clickhouse", "clickhouse",
		`rm -f /backups/ferrite.zip && `+
			// Output goes nowhere: the statement prints a status row, and one
			// line of "BACKUP_CREATED" at the front of a zip file is a zip
			// file nothing can open.
			clickhouseQuery(`BACKUP DATABASE app TO Disk('backups', 'ferrite.zip')`)+` >/dev/null && `+
			`cat /backups/ferrite.zip && rm -f /backups/ferrite.zip`),
	Restore: inContainer("clickhouse", "clickhouse",
		`rm -f /backups/restore.zip && cat > /backups/restore.zip`),
	RestoreAfter: []string{
		// Dropped rather than restored over. RESTORE refuses to write into a
		// table that already has rows unless told to allow it, and allowing it
		// appends — so a restore meant to undo a mistake would leave the
		// database holding every row twice. SYNC waits for the drop to finish
		// rather than returning while it is still happening.
		inContainer("clickhouse", "clickhouse",
			clickhouseQuery(`DROP DATABASE IF EXISTS app SYNC`)),
		inContainer("clickhouse", "clickhouse",
			clickhouseQuery(`RESTORE DATABASE app FROM Disk('backups', 'restore.zip')`)+` >/dev/null`),
		inContainer("clickhouse", "clickhouse", `rm -f /backups/restore.zip`),
	},
	Warning: "Everything currently in this database is replaced by the copy you are restoring. " +
		"Anything written since that backup was taken is lost.",
}

// clickhouseQuery runs one statement as the account the tool was set up with.
//
// The password comes from the container's own environment rather than being
// written into the command, so it never reaches a job log or a process list on
// the server.
//
// The statement is wrapped in double quotes rather than passed through
// steps.Quote. Everything here is already inside a single-quoted `sh -c`, and
// SQL string literals are single-quoted too — quoting it a second time turns
// Disk('backups', 'x.zip') into a line with eleven consecutive quote marks
// that nobody can check by reading. Double quotes leave the single ones alone,
// and these statements are constants with no expansion in them.
func clickhouseQuery(sql string) string {
	return `clickhouse-client --user ferrite --password "$CLICKHOUSE_PASSWORD" --query "` + sql + `"`
}
