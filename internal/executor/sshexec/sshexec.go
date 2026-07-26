// Package sshexec runs commands on a real machine over SSH.
package sshexec

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

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
