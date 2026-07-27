package notify

import (
	"strings"
	"testing"

	"github.com/ebnsina/ferrite-ship/internal/config"
)

// A newline in a header ends it and starts another. Nothing user-written
// reaches these headers today, but a server name will the moment somebody
// wants it in a subject line, and by then this is a header injection rather
// than a formatting bug.
func TestHeadersCannotBeSplit(t *testing.T) {
	m := &mailer{settings: config.SMTP{From: "alerts@example.com"}}

	encoded := string(m.encode(Message{
		To:      "someone@example.com\r\nBcc: attacker@example.com",
		Subject: "Broken\r\nX-Injected: yes",
		Body:    "body",
	}))

	// The injected text is allowed to survive as gibberish inside the header it
	// was written into. What must not happen is a new header line: that is what
	// turns a wrong subject into a message with recipients nobody chose.
	head, body, found := strings.Cut(encoded, "\r\n\r\n")
	if !found {
		t.Fatal("no header separator")
	}

	for _, line := range strings.Split(head, "\r\n") {
		name, _, ok := strings.Cut(line, ":")
		if !ok {
			t.Errorf("a header line with no name appeared: %q", line)
			continue
		}
		switch name {
		case "From", "To", "Subject", "Date", "MIME-Version", "Content-Type":
		default:
			t.Errorf("a header was split open, producing %q", name)
		}
	}
	if strings.Count(head, "\r\n")+1 != 6 {
		t.Errorf("unexpected header count in:\n%s", head)
	}
	if !strings.HasPrefix(body, "body") {
		t.Errorf("body did not survive: %q", body)
	}
}

// The writer returned by Data does dot-stuffing and writes the terminating
// dot itself. Doing it here as well truncates any message with a line that
// happens to start with a period.
func TestTheTerminatingDotIsLeftToTheWriter(t *testing.T) {
	m := &mailer{settings: config.SMTP{From: "alerts@example.com"}}

	encoded := string(m.encode(Message{To: "a@b.com", Subject: "s", Body: "line one\n. line two"}))

	if strings.HasSuffix(encoded, "\r\n.\r\n") {
		t.Error("the message must not terminate itself")
	}
	if !strings.Contains(encoded, ". line two") {
		t.Error("the body was altered on the way out")
	}
}

// Without a mail server, sending has to fail loudly. A test button that
// reports success when nothing was sent is the worst outcome this package has.
func TestDisabledSendingReportsItself(t *testing.T) {
	sender := New(config.SMTP{})

	if sender.Enabled() {
		t.Fatal("no host means not enabled")
	}
	if err := sender.Send(t.Context(), Message{To: "a@b.com"}); err != ErrDisabled {
		t.Errorf("want ErrDisabled, got %v", err)
	}
}

// Every alert has to name what happened and what to do about it. A message
// that arrives at 4am with only a status code in it is one nobody can act on.
func TestEveryAlertSaysWhatToDo(t *testing.T) {
	for kind := range wording {
		alert := Alert{
			Kind:    kind,
			Server:  "web-1",
			Subject: "PostgreSQL",
			Detail:  "the detail",
			Link:    "/dashboard/servers/srv_1",
		}

		message := Render(alert, "https://ship.example.com")

		if message.Subject == "" {
			t.Errorf("%s has no subject line", kind)
		}
		if !strings.Contains(message.Body, "the detail") {
			t.Errorf("%s drops the specifics", kind)
		}
		if !strings.Contains(message.Body, "https://ship.example.com/dashboard/servers/srv_1") {
			t.Errorf("%s has no link to the page that can fix it", kind)
		}
	}
}

// A dashboard address that was never configured must not produce a link to
// nowhere: a broken link in the one message that matters is worse than none.
func TestNoOriginMeansNoLink(t *testing.T) {
	message := Render(Alert{
		Kind: KindServerDown, Server: "web-1", Detail: "timed out", Link: "/dashboard",
	}, "")

	if strings.Contains(message.Body, "/dashboard") {
		t.Errorf("a link was rendered without an origin:\n%s", message.Body)
	}
}
