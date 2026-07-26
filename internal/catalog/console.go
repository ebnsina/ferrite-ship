package catalog

import (
	"encoding/base64"
	"strconv"

	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// Console describes how to run a query against a tool and read the answer.
//
// Installing a database you cannot then look inside is a demonstration rather
// than a feature: the whole reason to have one is to put data in and get it
// out again. This is what makes an installed tool usable without opening a
// terminal and remembering the flags.
type Console struct {
	// Language names what you type, in the words shown above the editor.
	Language string
	// Placeholder is an example query for an empty editor.
	Placeholder string
	// Format describes how results come back, so the caller knows how to read
	// stdout.
	Format ResultFormat
	// build renders the command for one query.
	build func(query string, limit int) string
}

// ResultFormat is how a tool's client is asked to print its answer.
type ResultFormat string

const (
	// FormatCSV is a header row followed by rows, RFC 4180 quoted.
	FormatCSV ResultFormat = "csv"
	// FormatLines is one value per line, for tools with no notion of a table.
	FormatLines ResultFormat = "lines"
)

// ConsoleSpec returns the tool's console, if it has one.
func (t Tool) ConsoleSpec() (Console, bool) {
	if t.console == nil {
		return Console{}, false
	}
	return *t.console, true
}

// Command renders the shell command that runs one query.
func (c Console) Command(query string, limit int) string { return c.build(query, limit) }

// pipeIn feeds text to a command inside a container without ever quoting it.
//
// The query is arbitrary text from a person: it will contain quotes, dollars,
// backslashes and newlines, and it passes through two shells and a container
// before it arrives. Base64 has none of those characters, so it survives every
// layer untouched — no amount of escaping is as reliable as having nothing to
// escape.
func pipeIn(id, service, text, command string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	return inContainer(id, service, "printf %s "+steps.Quote(encoded)+" | base64 -d | "+command)
}

// postgresConsole runs the query through psql.
//
// statement_timeout comes from PGOPTIONS rather than a SET statement, because
// a SET would have to share the connection with the query and psql would then
// print two result sets into what is meant to be one CSV document.
var postgresConsole = &Console{
	Language:    "SQL",
	Placeholder: "select * from information_schema.tables where table_schema = 'public';",
	Format:      FormatCSV,
	build: func(query string, limit int) string {
		return pipeIn("postgres", "postgres", query,
			`PGPASSWORD="$POSTGRES_PASSWORD" PGOPTIONS='-c statement_timeout=15000' `+
				`psql -U ferrite -d app --csv -v ON_ERROR_STOP=1 -f - 2>&1 | head -n `+strconv.Itoa(limit+1))
	},
}

// clickhouseConsole asks for CSV with a header so both databases come back in
// the same shape and one parser reads either.
var clickhouseConsole = &Console{
	Language:    "SQL",
	Placeholder: "select name, engine from system.tables where database = 'app';",
	Format:      FormatCSV,
	build: func(query string, limit int) string {
		return pipeIn("clickhouse", "clickhouse", query,
			`clickhouse-client --user ferrite --password "$CLICKHOUSE_PASSWORD" --database app `+
				`--max_execution_time 15 --format CSVWithNames 2>&1 | head -n `+strconv.Itoa(limit+1))
	},
}

// redisConsole takes commands rather than queries: Redis has no tables, so
// results come back as lines and the UI shows them as such.
var redisConsole = &Console{
	Language:    "Redis command",
	Placeholder: "KEYS *",
	Format:      FormatLines,
	build: func(query string, limit int) string {
		// xargs, and deliberately not `xargs -0`: redis-cli does not read
		// commands from stdin, so the line has to become arguments. Plain
		// xargs splits on whitespace and honours quotes, which is what makes
		// `SET motto "hello there"` arrive as two arguments rather than three.
		// With -0 the whole line arrives as a single argument and Redis
		// answers "unknown command".
		return pipeIn("redis", "redis", query,
			`xargs redis-cli -a "$REDIS_PASSWORD" --no-auth-warning 2>&1 | head -n `+strconv.Itoa(limit))
	},
}
