package notify

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/config"
)

// The SMTP conversation is the part that cannot be checked by reading it. Every
// mistake in it — a missing STARTTLS, a wrong AUTH encoding, a message the
// server never sees the end of — looks exactly like working code until the one
// night something has actually gone wrong.
//
// So this is a real mail server: a real TLS handshake, a real login, and the
// message read back off the wire.
func TestASendGoesAllTheWayThrough(t *testing.T) {
	server := startFakeSMTP(t)

	sender := &mailer{
		settings: config.SMTP{
			Host:     "localhost",
			Port:     server.port,
			User:     "postmaster",
			Password: "hunter2",
			From:     "alerts@example.com",
		},
		// The fake server's certificate is signed by nobody.
		tlsConfig: func(string) *tls.Config { return &tls.Config{InsecureSkipVerify: true} },
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	body := "The disk is 91% full.\n. a line that starts with a dot\nlast line"
	if err := sender.Send(ctx, Message{
		To:      "someone@example.com",
		Subject: "web-1 is running out of disk",
		Body:    body,
	}); err != nil {
		t.Fatalf("send: %v", err)
	}

	got := <-server.delivered

	if !got.startedTLS {
		t.Error("the password was sent without STARTTLS")
	}
	if got.user != "postmaster" || got.password != "hunter2" {
		t.Errorf("the login did not arrive intact: %q / %q", got.user, got.password)
	}
	if got.from != "alerts@example.com" {
		t.Errorf("wrong sender: %q", got.from)
	}
	if got.to != "someone@example.com" {
		t.Errorf("wrong recipient: %q", got.to)
	}

	if !strings.Contains(got.data, "Subject: web-1 is running out of disk") {
		t.Errorf("subject missing from:\n%s", got.data)
	}
	// The dot-stuffed line has to arrive as one dot, and the line after it has
	// to arrive at all — a message that terminates itself early loses the rest.
	if !strings.Contains(got.data, "\n. a line that starts with a dot") {
		t.Errorf("a leading dot was mangled:\n%s", got.data)
	}
	if !strings.HasSuffix(strings.TrimSpace(got.data), "last line") {
		t.Errorf("the message was cut short:\n%s", got.data)
	}
}

// A mail server that cannot secure the connection is refused rather than
// downgraded to, because the alternative is sending a password in the clear.
func TestAServerWithoutSTARTTLSIsRefused(t *testing.T) {
	server := startFakeSMTP(t)
	server.offerTLS = false

	sender := &mailer{settings: config.SMTP{
		Host: "localhost", Port: server.port, User: "u", Password: "p", From: "a@b.com",
	}}

	err := sender.Send(t.Context(), Message{To: "someone@example.com", Subject: "s", Body: "b"})
	if err == nil {
		t.Fatal("want a refusal, got a send")
	}
	if !strings.Contains(err.Error(), "STARTTLS") {
		t.Errorf("the reason should name the missing thing, got: %v", err)
	}
}

type delivery struct {
	startedTLS     bool
	user, password string
	from, to, data string
}

type fakeSMTP struct {
	port      int
	offerTLS  bool
	delivered chan delivery
}

// startFakeSMTP runs a mail server that speaks just enough SMTP.
func startFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	port, err := strconv.Atoi(strings.TrimPrefix(listener.Addr().String(), "127.0.0.1:"))
	if err != nil {
		t.Fatalf("port: %v", err)
	}

	server := &fakeSMTP{port: port, offerTLS: true, delivered: make(chan delivery, 1)}
	certificate := selfSigned(t)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handle(conn, certificate)
		}
	}()

	return server
}

func (s *fakeSMTP) handle(conn net.Conn, certificate tls.Certificate) {
	defer func() { _ = conn.Close() }()

	var got delivery
	reader := bufio.NewReader(conn)
	write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

	write("220 fake ESMTP")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		verb, rest, _ := strings.Cut(line, " ")

		switch strings.ToUpper(verb) {
		case "EHLO", "HELO":
			if s.offerTLS && !got.startedTLS {
				write("250-fake")
				write("250-STARTTLS")
				write("250 AUTH PLAIN")
			} else {
				write("250-fake")
				write("250 AUTH PLAIN")
			}

		case "STARTTLS":
			write("220 go ahead")
			secured := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{certificate}})
			if err := secured.Handshake(); err != nil {
				return
			}
			conn = secured
			reader = bufio.NewReader(conn)
			write = func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }
			got.startedTLS = true

		case "AUTH":
			// PLAIN is \x00user\x00password, base64.
			_, encoded, _ := strings.Cut(rest, " ")
			decoded, _ := base64.StdEncoding.DecodeString(encoded)
			parts := strings.Split(string(decoded), "\x00")
			if len(parts) == 3 {
				got.user, got.password = parts[1], parts[2]
			}
			write("235 fine")

		case "MAIL":
			got.from = strings.Trim(strings.TrimPrefix(rest, "FROM:"), "<>")
			write("250 ok")

		case "RCPT":
			got.to = strings.Trim(strings.TrimPrefix(rest, "TO:"), "<>")
			write("250 ok")

		case "DATA":
			write("354 go on")
			var body strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				// Undo dot-stuffing the way a real server does: a doubled dot
				// becomes one, rather than both being dropped.
				if strings.HasPrefix(dataLine, "..") {
					dataLine = dataLine[1:]
				}
				body.WriteString(dataLine)
			}
			got.data = body.String()
			write("250 queued")

		case "QUIT":
			write("221 bye")
			s.delivered <- got
			return

		default:
			write("250 ok")
		}
	}
}

func selfSigned(t *testing.T) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("certificate: %v", err)
	}

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}
