// Package dialer is the one way the control plane opens an SSH connection to a
// managed server.
//
// Before this existed, four packages each resolved the server, decrypted its
// credentials and dialled. Host key checking would have had to be added to all
// four and kept in step; now there is one place that can be got right.
package dialer

import (
	"context"
	"fmt"

	"github.com/ebnsina/ferrite-ship/internal/executor/sshexec"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// ErrNotSupported means the server has no real machine behind it.
type ErrNotSupported struct{}

func (ErrNotSupported) Error() string { return "this server is simulated" }

type Dialer struct {
	store  *store.Store
	sealer *secret.Sealer
}

func New(st *store.Store, sealer *secret.Sealer) *Dialer {
	return &Dialer{store: st, sealer: sealer}
}

// Dial resolves a server the user owns and connects to it.
//
// The server's identity is checked against the one recorded when it was first
// reached; a mismatch fails rather than connecting, because it means either a
// rebuilt machine or somebody in the middle, and only a person can tell which.
func (d *Dialer) Dial(ctx context.Context, userID, serverID string) (*sshexec.Client, store.Server, error) {
	server, err := d.store.GetServer(ctx, userID, serverID)
	if err != nil {
		return nil, store.Server{}, err
	}
	if server.Kind != store.ConnectionSSH {
		return nil, server, ErrNotSupported{}
	}

	password, err := d.sealer.Open(server.SealedPassword)
	if err != nil {
		return nil, server, fmt.Errorf("could not read the stored password: %w", err)
	}
	privateKey, err := d.sealer.Open(server.SealedPrivateKey)
	if err != nil {
		return nil, server, fmt.Errorf("could not read the stored key: %w", err)
	}

	client, err := sshexec.Dial(ctx, sshexec.Config{
		Host:         server.Host,
		Port:         server.Port,
		User:         server.User,
		Password:     password,
		PrivateKey:   privateKey,
		KnownHostKey: server.HostKey,
	})
	if err != nil {
		return nil, server, err
	}

	// First successful connection: remember who this server is, so the next
	// one can be checked. Failing to record it is not worth dropping a working
	// connection over — it just means we learn it next time.
	if server.HostKey == "" && client.HostKey() != "" {
		if err := d.store.RememberHostKey(ctx, server.ID, client.HostKey()); err == nil {
			server.HostKey = client.HostKey()
		}
	}

	return client, server, nil
}
