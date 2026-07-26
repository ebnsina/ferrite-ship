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
	// Presets are the questions people ask a database in its first five
	// minutes: what is in here, how big is it, what is it doing. Knowing the
	// answer means knowing each database's own system tables, which is exactly
	// the knowledge someone new to it does not have.
	Presets []Preset
	// build renders the command for one query.
	build func(query string, limit int) string
}

// Preset is a ready-made query, offered as a starting point.
type Preset struct {
	Label string `json:"label"`
	// Description says what the answer will tell you, in plain language.
	Description string `json:"description"`
	Query       string `json:"query"`
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
	Presets:     postgresPresets,
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
	Presets:     clickhousePresets,
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
	Presets:     redisPresets,
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

// The presets.
//
// Each is written against the database's own catalogue, and each answers a
// question someone actually has rather than demonstrating a feature. Sizes are
// formatted by the database, because it knows its own units.

var postgresPresets = []Preset{
	{
		Label:       "What tables are in here",
		Description: "Every table, largest first, with how much room it takes up.",
		Query: `select table_name,
       pg_size_pretty(pg_total_relation_size(quote_ident(table_name))) as size
from information_schema.tables
where table_schema = 'public'
order by pg_total_relation_size(quote_ident(table_name)) desc`,
	},
	{
		Label:       "How many rows",
		Description: "An estimate per table, kept by PostgreSQL itself, so it is instant even on a large table.",
		Query: `select relname as table_name, n_live_tup as approximate_rows
from pg_stat_user_tables
order by n_live_tup desc`,
	},
	{
		Label:       "How big is the database",
		Description: "The total on disk, including indexes.",
		Query: `select current_database() as database,
       pg_size_pretty(pg_database_size(current_database())) as size`,
	},
	{
		Label:       "Who is connected",
		Description: "Every open connection and what it is doing right now.",
		Query: `select usename as username, application_name, state, client_addr
from pg_stat_activity
where datname = current_database()
order by state`,
	},
	{
		Label:       "What is running now",
		Description: "Anything currently executing, longest first — where you look when things feel slow.",
		Query: `select pid, state, now() - query_start as running_for, left(query, 80) as query
from pg_stat_activity
where datname = current_database() and state <> 'idle'
order by query_start`,
	},
}

var clickhousePresets = []Preset{
	{
		Label:       "What tables are in here",
		Description: "Every table with its engine and row count.",
		Query: `select name, engine, total_rows
from system.tables
where database = 'app'
order by name`,
	},
	{
		Label:       "How big is each table",
		Description: "Size on disk and row count, largest first.",
		Query: `select table,
       formatReadableSize(sum(bytes_on_disk)) as size,
       sum(rows) as rows
from system.parts
where active and database = 'app'
group by table
order by sum(bytes_on_disk) desc`,
	},
	{
		Label:       "How big is the database",
		Description: "Everything this database holds, on disk.",
		Query: `select formatReadableSize(sum(bytes_on_disk)) as size, sum(rows) as rows
from system.parts
where active and database = 'app'`,
	},
	{
		Label:       "Recent queries",
		Description: "The last twenty that finished, with how long each took.",
		Query: `select event_time, query_duration_ms, left(query, 60) as query
from system.query_log
where type = 'QueryFinish'
order by event_time desc
limit 20`,
	},
}

var redisPresets = []Preset{
	{
		Label:       "How many keys",
		Description: "The number of keys stored right now.",
		Query:       "DBSIZE",
	},
	{
		Label:       "List the keys",
		Description: "Every key. Careful on a large cache — this walks all of them.",
		Query:       "KEYS *",
	},
	{
		Label:       "How much memory",
		Description: "What Redis is using, and the most it has ever used.",
		Query:       "INFO memory",
	},
	{
		Label:       "Is anything slow",
		Description: "The commands Redis itself recorded as taking too long.",
		Query:       "SLOWLOG GET 10",
	},
}
