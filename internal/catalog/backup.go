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
	// Restore reads the backup from stdin. It must be safe to run over a
	// database that already has data in it.
	Restore string
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
	// stops it, replaces the file and starts it again. The appendonly log has
	// to go too: Redis prefers it over the snapshot on boot, so leaving it in
	// place would silently restore the old data and report success.
	Restore: compose("redis", "stop", "redis") + " && " +
		`docker run --rm -i -v ferrite-redis_data:/data alpine:3.21 sh -c 'rm -rf /data/appendonlydir /data/dump.rdb && cat > /data/dump.rdb'` +
		" && " + compose("redis", "start", "redis"),
	Warning: "Redis is stopped for a moment while it is replaced, so anything using it will be briefly unable to connect. Everything currently stored is replaced.",
}
