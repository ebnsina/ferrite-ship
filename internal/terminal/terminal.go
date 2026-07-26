// Package terminal opens interactive shells on connected servers.
//
// This is separate from the job runner on purpose: a job is a bounded piece of
// work with a recorded outcome, whereas a terminal is an open-ended session
// nobody is grading. They share the SSH client and nothing else.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ebnsina/ferrite-ship/internal/executor/sshexec"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// ErrNotSupported is returned for servers with no real machine behind them.
var ErrNotSupported = errors.New("this server has no shell to open")

// Size is the terminal window in character cells.
type Size struct {
	Cols int
	Rows int
}

func (s Size) withDefaults() Size {
	if s.Cols <= 0 {
		s.Cols = 80
	}
	if s.Rows <= 0 {
		s.Rows = 24
	}
	return s
}

// Session is a live shell. Read yields output, Write sends keystrokes.
type Session struct {
	client *sshexec.Client
	shell  *sshexec.Shell
}

func (s *Session) Read(p []byte) (int, error)  { return s.shell.Read(p) }
func (s *Session) Write(p []byte) (int, error) { return s.shell.Write(p) }

func (s *Session) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	return s.shell.Resize(cols, rows)
}

// Close tears down the shell and the connection that carried it.
func (s *Session) Close() error {
	shellErr := s.shell.Close()
	clientErr := s.client.Close()
	if shellErr != nil {
		return shellErr
	}
	return clientErr
}

var _ io.ReadWriter = (*Session)(nil)

type Service struct {
	store  *store.Store
	sealer *secret.Sealer
}

func NewService(st *store.Store, sealer *secret.Sealer) *Service {
	return &Service{store: st, sealer: sealer}
}

// Open dials the server and starts a login shell.
//
// Each terminal gets its own SSH connection rather than sharing one: a shell
// that hangs or is killed must not disturb a job running at the same time.
func (s *Service) Open(ctx context.Context, userID, serverID string, size Size) (*Session, error) {
	server, err := s.store.GetServer(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}

	if server.Kind != store.ConnectionSSH {
		return nil, ErrNotSupported
	}

	password, err := s.sealer.Open(server.SealedPassword)
	if err != nil {
		return nil, fmt.Errorf("could not read the stored password: %w", err)
	}
	privateKey, err := s.sealer.Open(server.SealedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("could not read the stored key: %w", err)
	}

	client, err := sshexec.Dial(ctx, sshexec.Config{
		Host:       server.Host,
		Port:       server.Port,
		User:       server.User,
		Password:   password,
		PrivateKey: privateKey,
	})
	if err != nil {
		return nil, err
	}

	size = size.withDefaults()

	shell, err := client.OpenShell(size.Cols, size.Rows)
	if err != nil {
		_ = client.Close()
		return nil, err
	}

	return &Session{client: client, shell: shell}, nil
}
