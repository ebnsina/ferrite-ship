package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"

	"github.com/ebnsina/ferrite-ship/internal/store"
	"github.com/ebnsina/ferrite-ship/internal/terminal"
)

// terminalMessage is what the browser sends up. Output travels the other way as
// raw binary frames, so xterm can hand partial UTF-8 sequences to its own
// decoder instead of us splitting characters.
type terminalMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

const (
	// readLimit caps a single inbound frame. Keystrokes and paste are small;
	// anything larger is a mistake or an attack.
	terminalReadLimit = 1 << 20
	// pingInterval keeps idle terminals alive through proxies that cull quiet
	// connections.
	terminalPingInterval = 30 * time.Second
)

func (a *API) handleTerminal(w http.ResponseWriter, r *http.Request) {
	size := terminal.Size{
		Cols: atoiOr(r.URL.Query().Get("cols"), 80),
		Rows: atoiOr(r.URL.Query().Get("rows"), 24),
	}

	// Open the shell before upgrading, so a failure is a normal HTTP error the
	// client can actually read rather than an immediate socket close.
	session, err := a.terminals.Open(r.Context(), currentUser(r).ID, r.PathValue("id"), size)
	switch {
	case errors.Is(err, store.ErrNotFound):
		a.writeError(w, http.StatusNotFound, "not_found", "We could not find that server.")
		return
	case errors.Is(err, terminal.ErrNotSupported):
		a.writeError(w, http.StatusBadRequest, "parse",
			"This is a simulated server, so there is no shell to open. Connect a real server to use the terminal.")
		return
	case err != nil:
		a.writeError(w, http.StatusBadGateway, "network", friendlyFileError(err))
		return
	}
	defer func() { _ = session.Close() }()

	accept := &websocket.AcceptOptions{}
	if a.allowedOriginHost != "" {
		accept.OriginPatterns = []string{a.allowedOriginHost}
	}

	conn, err := websocket.Accept(w, r, accept)
	if err != nil {
		a.log.Warn("terminal upgrade failed", "error", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	conn.SetReadLimit(terminalReadLimit)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go a.pumpShellOutput(ctx, cancel, conn, session)
	go keepAlive(ctx, conn)

	a.pumpTerminalInput(ctx, conn, session)
}

// pumpShellOutput forwards what the shell prints to the browser.
func (a *API) pumpShellOutput(
	ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, session *terminal.Session,
) {
	defer cancel()

	buf := make([]byte, 8192)
	for {
		n, readErr := session.Read(buf)
		if n > 0 {
			if err := conn.Write(ctx, websocket.MessageBinary, buf[:n]); err != nil {
				return
			}
		}
		if readErr != nil {
			// The shell exited or the connection dropped; either way we are done.
			_ = conn.Close(websocket.StatusNormalClosure, "shell closed")
			return
		}
	}
}

// pumpTerminalInput applies keystrokes and resizes from the browser.
func (a *API) pumpTerminalInput(
	ctx context.Context, conn *websocket.Conn, session *terminal.Session,
) {
	for {
		kind, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if kind != websocket.MessageText {
			continue
		}

		var msg terminalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue // a malformed frame is not worth dropping the session for
		}

		switch msg.Type {
		case "input":
			if _, err := session.Write([]byte(msg.Data)); err != nil {
				return
			}
		case "resize":
			if err := session.Resize(msg.Cols, msg.Rows); err != nil {
				a.log.Debug("terminal resize failed", "error", err)
			}
		}
	}
}

func keepAlive(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(terminalPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := conn.Ping(ctx); err != nil {
				return
			}
		}
	}
}

func atoiOr(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
