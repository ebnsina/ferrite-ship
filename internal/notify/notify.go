// Package notify tells somebody that something happened while they were away.
//
// Everything else in this product runs because a person pressed a button and
// watched the log. Scheduled backups do not, and neither does a disk filling up
// at four in the morning. Without this, the failure is discovered by opening a
// page that nobody has a reason to open.
package notify

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/config"
)

// ErrDisabled is returned when no mail server is configured.
//
// A distinct error rather than a silent success: the settings page needs to be
// able to say "this would go nowhere", and a test send that reports "sent" when
// nothing was sent is the single worst thing this package could do.
var ErrDisabled = errors.New("no mail server is configured")

// Message is one email, in plain text.
//
// No HTML. These are read on a phone at an awkward hour by somebody who wants
// to know what broke, and plain text renders the same everywhere.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender hands a message to a mail server.
type Sender interface {
	Send(ctx context.Context, message Message) error
	// Enabled reports whether sending can work at all, so callers can avoid
	// queueing work that has nowhere to go.
	Enabled() bool
}

// New returns a sender for the configured mail server, or one that reports
// ErrDisabled if none was configured.
func New(settings config.SMTP) Sender {
	if !settings.Enabled() {
		return disabled{}
	}
	return &mailer{settings: settings}
}

type disabled struct{}

func (disabled) Enabled() bool                       { return false }
func (disabled) Send(context.Context, Message) error { return ErrDisabled }

type mailer struct {
	settings config.SMTP

	// tlsConfig is how the connection is secured.
	//
	// A field only so the tests can talk to a server with a self-signed
	// certificate. Nothing outside this package can set it, so a real send
	// always verifies the certificate against the host it asked for.
	tlsConfig func(host string) *tls.Config
}

func (m *mailer) secure() *tls.Config {
	if m.tlsConfig != nil {
		return m.tlsConfig(m.settings.Host)
	}
	return &tls.Config{ServerName: m.settings.Host}
}

func (m *mailer) Enabled() bool { return true }

// Send delivers one message.
//
// Synchronous and given a deadline by the caller. Mail servers are slow often
// enough that a send without a timeout eventually becomes a goroutine that
// never returns.
func (m *mailer) Send(ctx context.Context, message Message) error {
	if strings.TrimSpace(message.To) == "" {
		return errors.New("notify: no recipient")
	}

	address := net.JoinHostPort(m.settings.Host, strconv.Itoa(m.settings.Port))

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("notify: could not reach %s: %w", address, err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if m.settings.Implicit {
		conn = tls.Client(conn, m.secure())
	}

	client, err := smtp.NewClient(conn, m.settings.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("notify: %s did not answer as a mail server: %w", address, err)
	}
	defer func() { _ = client.Close() }()

	if !m.settings.Implicit {
		// STARTTLS is required, not attempted. The password goes over this
		// connection, and a server that cannot offer TLS is one this should
		// refuse rather than quietly downgrade to.
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("notify: %s does not offer STARTTLS, so the password cannot be sent safely", address)
		}
		if err := client.StartTLS(m.secure()); err != nil {
			return fmt.Errorf("notify: could not secure the connection to %s: %w", address, err)
		}
	}

	if m.settings.User != "" {
		auth := smtp.PlainAuth("", m.settings.User, m.settings.Password, m.settings.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("notify: %s rejected the login: %w", address, err)
		}
	}

	if err := client.Mail(m.settings.From); err != nil {
		return fmt.Errorf("notify: %s rejected the sender %q: %w", address, m.settings.From, err)
	}
	if err := client.Rcpt(message.To); err != nil {
		return fmt.Errorf("notify: %s rejected the recipient: %w", address, err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("notify: %s refused the message: %w", address, err)
	}
	if _, err := writer.Write(m.encode(message)); err != nil {
		return fmt.Errorf("notify: could not write the message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("notify: %s did not accept the message: %w", address, err)
	}

	return client.Quit()
}

// encode builds the RFC 5322 message.
//
// Nothing here escapes a leading period or writes the terminating dot: the
// writer returned by Data is a textproto dot-writer, which does both on Close.
// Doing it here as well would send every message with a stray dot and truncate
// any body that happened to contain one.
func (m *mailer) encode(message Message) []byte {
	var builder strings.Builder

	builder.WriteString("From: Ferrite Ship <" + m.settings.From + ">\r\n")
	builder.WriteString("To: " + header(message.To) + "\r\n")
	builder.WriteString("Subject: " + header(message.Subject) + "\r\n")
	builder.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	builder.WriteString("\r\n")

	for _, line := range strings.Split(strings.ReplaceAll(message.Body, "\r\n", "\n"), "\n") {
		builder.WriteString(line + "\r\n")
	}

	return []byte(builder.String())
}

// header strips anything that would end the header and start another one.
func header(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(value))
}
