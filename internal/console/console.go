// Package console runs a query against an installed tool and reads the answer
// back as a table.
//
// It exists so that installing a database is worth doing: the point of having
// one is to put data in and get it out, and needing a terminal and the right
// client flags to do that makes the rest of the product decoration.
package console

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/catalog"
	"github.com/ebnsina/ferrite-ship/internal/dialer"
	"github.com/ebnsina/ferrite-ship/internal/steps"
)

// MaxRows is how many rows come back at most.
//
// A `select *` on a table with a million rows is a thing people type by
// accident, and the answer has to travel over SSH and into a browser. The cap
// is applied on the server by the command itself, so the rows are never
// carried in the first place.
const MaxRows = 500

// queryTimeout bounds the whole round trip. The database clients are also told
// to give up sooner, so this only catches a connection that has gone quiet.
const queryTimeout = 30 * time.Second

var (
	// ErrNoConsole is returned for a tool with nothing to query.
	ErrNoConsole = errors.New("console: this tool cannot be queried")
	// ErrEmptyQuery keeps a blank editor from running a command.
	ErrEmptyQuery = errors.New("console: no query")
)

// Result is one query's answer.
type Result struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	// Truncated is true when the cap cut the answer short.
	Truncated bool `json:"truncated"`
	// Failed is true when the tool reported a problem rather than a result;
	// Message then holds what it said.
	Failed  bool   `json:"failed"`
	Message string `json:"message,omitempty"`
	// TookMs is the round trip, so a slow query is visibly slow.
	TookMs int64 `json:"tookMs"`
}

type Service struct {
	dialer *dialer.Dialer
}

func New(d *dialer.Dialer) *Service { return &Service{dialer: d} }

// Run executes one query against a tool on a server.
func (s *Service) Run(
	ctx context.Context, userID, serverID, toolID, query string,
) (Result, error) {
	tool, err := catalog.Find(toolID)
	if err != nil {
		return Result{}, err
	}

	spec, ok := tool.ConsoleSpec()
	if !ok {
		return Result{}, ErrNoConsole
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return Result{}, ErrEmptyQuery
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	client, _, err := s.dialer.Dial(ctx, userID, serverID)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = client.Close() }()

	privilege, err := steps.DetectPrivilege(ctx, client)
	if err != nil {
		return Result{}, err
	}
	// Nothing is logged from here: a query is the person's own text and its
	// results are their own data, and neither belongs in a job history.
	session := steps.NewSession(client, nil).WithPrivilege(privilege)

	started := time.Now()
	out, err := session.Capture(ctx, spec.Command(query, MaxRows))
	took := time.Since(started).Milliseconds()

	if err != nil {
		// The command itself failed to run. Everything the client prints —
		// including a syntax error — arrives on stdout, so this is a transport
		// problem rather than a rejected query.
		return Result{}, err
	}

	result := parse(spec.Format, out)
	result.TookMs = took
	return result, nil
}

// parse turns a client's output into a table.
func parse(format catalog.ResultFormat, out string) Result {
	out = strings.TrimRight(out, "\n")
	if out == "" {
		// A statement that returns nothing — an insert, a create — is a
		// success with no table to show.
		return Result{Columns: []string{}, Rows: [][]string{}}
	}

	if format == catalog.FormatLines {
		lines := strings.Split(out, "\n")
		rows := make([][]string, 0, len(lines))
		for _, line := range lines {
			rows = append(rows, []string{line})
		}
		return Result{
			Columns:   []string{"Result"},
			Rows:      rows,
			Truncated: len(rows) >= MaxRows,
		}
	}

	reader := csv.NewReader(strings.NewReader(out))
	// Row width varies: psql prints a NOTICE or an error as a single bare
	// line, and refusing to read it would turn a useful message into a parse
	// failure.
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil || len(records) == 0 {
		// Not a table, so it is whatever the client wanted to tell us. That is
		// nearly always a syntax error, and showing it verbatim is far more
		// use than "the query failed".
		return Result{Failed: true, Message: firstLines(out, 6)}
	}

	// A single column named after an error is what psql produces for a failed
	// statement under --csv.
	if len(records) == 1 && len(records[0]) == 1 && looksLikeError(records[0][0]) {
		return Result{Failed: true, Message: firstLines(out, 6)}
	}

	rows := records[1:]
	truncated := false
	if len(rows) >= MaxRows {
		rows = rows[:MaxRows]
		truncated = true
	}

	return Result{Columns: records[0], Rows: rows, Truncated: truncated}
}

func looksLikeError(text string) bool {
	lower := strings.ToLower(text)
	return strings.HasPrefix(lower, "error") ||
		strings.HasPrefix(lower, "psql:") ||
		strings.Contains(lower, "code: ") // ClickHouse prefixes its errors this way
}

func firstLines(text string, n int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > n {
		lines = append(lines[:n], fmt.Sprintf("… and %d more lines", len(lines)-n))
	}
	return strings.Join(lines, "\n")
}
