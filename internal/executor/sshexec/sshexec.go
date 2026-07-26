// Package sshexec runs commands on a real machine over SSH.
package sshexec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/ebnsina/ferrite-ship/internal/executor"
)

type Config struct {
	Host string
	Port int
	User string
	// Exactly one of Password or PrivateKey must be set.
	Password   string
	PrivateKey string
	// Passphrase decrypts PrivateKey when it is encrypted.
	Passphrase string
	Timeout    time.Duration
}

type Client struct {
	client *ssh.Client
	target string
}

func Dial(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, errors.New("ssh: host is required")
	}
	if cfg.User == "" {
		return nil, errors.New("ssh: user is required")
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}

	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port))

	clientCfg := &ssh.ClientConfig{
		User: cfg.User,
		Auth: auth,
		// MVP: trust on first use. Before this is exposed to anyone else, pin
		// the host key on enrolment and verify it on every later connection —
		// without that this is open to a man-in-the-middle.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // see comment
		Timeout:         cfg.Timeout,
	}

	dialer := &net.Dialer{Timeout: cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("ssh: dial %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, clientCfg)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh: handshake with %s: %w", addr, err)
	}

	return &Client{
		client: ssh.NewClient(sshConn, chans, reqs),
		target: fmt.Sprintf("%s@%s", cfg.User, addr),
	}, nil
}

func authMethods(cfg Config) ([]ssh.AuthMethod, error) {
	switch {
	case cfg.PrivateKey != "":
		var signer ssh.Signer
		var err error
		if cfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(
				[]byte(cfg.PrivateKey), []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("ssh: parse private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil

	case cfg.Password != "":
		return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil

	default:
		return nil, errors.New("ssh: either a private key or a password is required")
	}
}

func (c *Client) Describe() string { return c.target }

func (c *Client) Run(ctx context.Context, cmd string) (executor.Result, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return executor.Result{}, fmt.Errorf("ssh: new session: %w", err)
	}
	defer func() { _ = session.Close() }()

	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()

	select {
	case <-ctx.Done():
		// Best effort: ask the remote side to stop, then give up on the session.
		_ = session.Signal(ssh.SIGKILL)
		return executor.Result{}, ctx.Err()

	case err := <-done:
		result := executor.Result{
			Stdout: strings.TrimRight(stdout.String(), "\n"),
			Stderr: strings.TrimRight(stderr.String(), "\n"),
		}

		var exitErr *ssh.ExitError
		switch {
		case err == nil:
			return result, nil
		case errors.As(err, &exitErr):
			// The command ran and failed — that is data, not a transport error.
			result.ExitCode = exitErr.ExitStatus()
			return result, nil
		default:
			return result, fmt.Errorf("ssh: run %q: %w", cmd, err)
		}
	}
}

func (c *Client) Close() error { return c.client.Close() }

// --- interactive shells -----------------------------------------------------

// Shell is a PTY-backed interactive session: what a person sees when they open
// a terminal, as opposed to the one-shot commands steps run.
type Shell struct {
	session *ssh.Session
	stdin   io.WriteCloser
	output  *io.PipeReader
}

// OpenShell starts a login shell with a pseudo-terminal attached.
//
// stdout and stderr are merged into one stream: a terminal shows them
// interleaved, and splitting them would reorder what the user sees.
func (c *Client) OpenShell(cols, rows int) (*Shell, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh: new session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("ssh: request pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("ssh: stdin pipe: %w", err)
	}

	reader, writer := io.Pipe()
	session.Stdout = writer
	session.Stderr = writer

	if err := session.Shell(); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("ssh: start shell: %w", err)
	}

	// Closing the write half when the shell exits gives readers a clean EOF
	// instead of a connection that simply stops producing bytes.
	go func() {
		_ = session.Wait()
		_ = writer.Close()
	}()

	return &Shell{session: session, stdin: stdin, output: reader}, nil
}

func (s *Shell) Read(p []byte) (int, error)  { return s.output.Read(p) }
func (s *Shell) Write(p []byte) (int, error) { return s.stdin.Write(p) }

// Resize tells the remote shell the window changed, so full-screen programs
// such as top or an editor redraw at the right size.
func (s *Shell) Resize(cols, rows int) error {
	return s.session.WindowChange(rows, cols)
}

func (s *Shell) Close() error {
	_ = s.stdin.Close()
	_ = s.output.Close()
	return s.session.Close()
}

// OpenSFTP starts the SFTP subsystem on this connection, for browsing and
// editing files. It rides the same connection as everything else, so it needs
// no extra port and no second credential.
func (c *Client) OpenSFTP() (*sftp.Client, error) {
	client, err := sftp.NewClient(c.client)
	if err != nil {
		return nil, fmt.Errorf("ssh: open sftp: %w", err)
	}
	return client, nil
}
