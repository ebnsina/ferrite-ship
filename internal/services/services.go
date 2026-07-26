// Package services inspects and controls systemd units on a server.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ebnsina/ferrite-ship/internal/executor/sshexec"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/steps"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

var (
	ErrNotSupported = errors.New("this server has no services to manage")
	ErrBadUnit      = errors.New("that is not a service name we recognise")
	ErrProtected    = errors.New("that service is protected")
	ErrBadAction    = errors.New("that is not something we can do to a service")
)

// Unit is one systemd service.
type Unit struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Active is systemd's high-level state: active, inactive, failed.
	Active string `json:"active"`
	// Sub is the finer state: running, dead, exited.
	Sub string `json:"sub"`
	// Enabled says whether it starts at boot: enabled, disabled, static, masked.
	Enabled string `json:"enabled"`
	// Protected marks units this tool refuses to stop or disable.
	Protected bool `json:"protected"`
}

type Action string

const (
	ActionStart   Action = "start"
	ActionStop    Action = "stop"
	ActionRestart Action = "restart"
	ActionEnable  Action = "enable"
	ActionDisable Action = "disable"
)

var validActions = map[Action]bool{
	ActionStart: true, ActionStop: true, ActionRestart: true,
	ActionEnable: true, ActionDisable: true,
}

// protectedUnits must not be stopped or disabled from here.
//
// Stopping sshd severs the only way back in, and stopping the network stack
// does the same less obviously. Restarting them is allowed: existing SSH
// connections survive a reload, and refusing that would block a legitimate
// repair.
var protectedUnits = map[string]bool{
	"ssh.service":              true,
	"sshd.service":             true,
	"systemd-networkd.service": true,
	"networking.service":       true,
	"systemd-resolved.service": true,
	"dbus.service":             true,
	"NetworkManager.service":   true,
	"systemd-journald.service": true,
}

// unitPattern is deliberately strict. Unit names reach a shell command, and
// while they are also quoted, a name that cannot contain a metacharacter in
// the first place is a better guarantee than quoting alone.
var unitPattern = regexp.MustCompile(`^[a-zA-Z0-9@._:\-\\]{1,128}\.service$`)

type Service struct {
	store  *store.Store
	sealer *secret.Sealer
}

func NewService(st *store.Store, sealer *secret.Sealer) *Service {
	return &Service{store: st, sealer: sealer}
}

// session bundles a connection with an elevated command runner.
type session struct {
	client *sshexec.Client
	shell  *steps.Session
}

func (s *session) Close() { _ = s.client.Close() }

func (s *Service) connect(ctx context.Context, serverID string) (*session, error) {
	server, err := s.store.GetServer(ctx, serverID)
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
		Host: server.Host, Port: server.Port, User: server.User,
		Password: password, PrivateKey: privateKey,
	})
	if err != nil {
		return nil, err
	}

	// Reuse the step engine's privilege handling rather than repeating it:
	// systemctl needs root just as the playbook does. No log sink, because
	// nobody is watching these commands.
	privilege, err := steps.DetectPrivilege(ctx, client)
	if err != nil {
		_ = client.Close()
		return nil, err
	}

	return &session{
		client: client,
		shell:  steps.NewSession(client, nil).WithPrivilege(privilege),
	}, nil
}

// systemd's JSON output, which is far steadier than parsing its columns.
type unitRow struct {
	Unit        string `json:"unit"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

type unitFileRow struct {
	UnitFile string `json:"unit_file"`
	State    string `json:"state"`
}

func (s *Service) List(ctx context.Context, serverID string) ([]Unit, error) {
	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	unitsJSON, err := sess.shell.Capture(ctx,
		`systemctl list-units --type=service --all --output=json --no-pager`)
	if err != nil {
		return nil, err
	}

	var rows []unitRow
	if err := json.Unmarshal([]byte(unitsJSON), &rows); err != nil {
		return nil, fmt.Errorf("could not read the service list: %w", err)
	}

	// Boot state lives in a different command, so fetch it and merge.
	enabled := map[string]string{}
	if filesJSON, err := sess.shell.Capture(ctx,
		`systemctl list-unit-files --type=service --output=json --no-pager`); err == nil {
		var files []unitFileRow
		if json.Unmarshal([]byte(filesJSON), &files) == nil {
			for _, file := range files {
				enabled[file.UnitFile] = file.State
			}
		}
	}

	units := make([]Unit, 0, len(rows))
	for _, row := range rows {
		units = append(units, Unit{
			Name:        row.Unit,
			Description: row.Description,
			Active:      row.Active,
			Sub:         row.Sub,
			Enabled:     enabled[row.Unit],
			Protected:   protectedUnits[row.Unit],
		})
	}
	return units, nil
}

// Perform runs an action, refusing the ones that would cut the server off.
func (s *Service) Perform(ctx context.Context, serverID, unit string, action Action) error {
	if !unitPattern.MatchString(unit) {
		return ErrBadUnit
	}
	if !validActions[action] {
		return fmt.Errorf("%w: %q", ErrBadAction, action)
	}
	if protectedUnits[unit] && (action == ActionStop || action == ActionDisable) {
		verb := "Stopping"
		if action == ActionDisable {
			verb = "Turning off"
		}
		return fmt.Errorf("%w: %s %s would cut off access to this server. Restarting it is allowed.",
			ErrProtected, verb, unit)
	}

	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return err
	}
	defer sess.Close()

	result, err := sess.shell.Run(ctx,
		fmt.Sprintf("systemctl %s %s", action, shellQuote(unit)))
	if err != nil {
		return err
	}
	if !result.OK() {
		detail := strings.TrimSpace(result.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(result.Stdout)
		}
		if detail == "" {
			detail = fmt.Sprintf("systemctl exited %d", result.ExitCode)
		}
		return errors.New(detail)
	}
	return nil
}

// Logs returns the most recent journal lines for a unit, oldest first.
func (s *Service) Logs(ctx context.Context, serverID, unit string, lines int) (string, error) {
	if !unitPattern.MatchString(unit) {
		return "", ErrBadUnit
	}
	if lines <= 0 || lines > 2000 {
		lines = 300
	}

	sess, err := s.connect(ctx, serverID)
	if err != nil {
		return "", err
	}
	defer sess.Close()

	result, err := sess.shell.Run(ctx, fmt.Sprintf(
		"journalctl -u %s -n %d --no-pager --output=short-iso 2>&1",
		shellQuote(unit), lines))
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// shellQuote mirrors the step engine's quoting: wrap in single quotes and
// escape any the value contains.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
