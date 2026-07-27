package store

import (
	"context"
	"testing"
)

// The whole value of storing alerts is that a condition is announced once. A
// disk that is checked every five minutes and mailed every time it is still
// full teaches somebody to filter these messages, and the next one that
// matters is never read.
func TestAnAlertIsOnlyNewOnce(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	seedServer(t, st, "usr_a", "srv_a", "box")

	alert := Alert{ID: "alr_1", UserID: "usr_a", ServerID: "srv_a", Kind: "disk-low", Detail: "91% full"}

	fresh, err := st.OpenAlert(ctx, alert)
	if err != nil {
		t.Fatalf("open alert: %v", err)
	}
	if !fresh {
		t.Fatal("the first time a condition is seen it has to be new, or nothing is ever sent")
	}

	alert.ID = "alr_2"
	alert.Detail = "93% full"
	fresh, err = st.OpenAlert(ctx, alert)
	if err != nil {
		t.Fatalf("open alert again: %v", err)
	}
	if fresh {
		t.Error("the same condition reported again must not be new")
	}

	open, err := st.OpenAlerts(ctx, "usr_a")
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("want one open alert, got %d", len(open))
	}
}

// Clearing is what makes the next occurrence reportable. Without it a disk
// that filled, was cleaned, and filled again would be silent the second time.
func TestClearingLetsAConditionBeReportedAgain(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	seedServer(t, st, "usr_a", "srv_a", "box")

	alert := Alert{ID: "alr_1", UserID: "usr_a", ServerID: "srv_a", Kind: "server-down"}
	if _, err := st.OpenAlert(ctx, alert); err != nil {
		t.Fatalf("open alert: %v", err)
	}

	was, err := st.ClearAlert(ctx, "usr_a", "srv_a", "server-down", "")
	if err != nil {
		t.Fatalf("clear alert: %v", err)
	}
	if !was {
		t.Fatal("clearing an open alert has to report that one was open, or no recovery is ever sent")
	}

	// Clearing again reports nothing, so a server that is fine every five
	// minutes does not send "resolved" every five minutes.
	was, err = st.ClearAlert(ctx, "usr_a", "srv_a", "server-down", "")
	if err != nil {
		t.Fatalf("clear alert again: %v", err)
	}
	if was {
		t.Error("clearing what is already clear must be silent")
	}

	alert.ID = "alr_2"
	fresh, err := st.OpenAlert(ctx, alert)
	if err != nil {
		t.Fatalf("reopen alert: %v", err)
	}
	if !fresh {
		t.Error("a condition that happens again after being cleared has to be new")
	}
}

// Two tools on one server fail independently, and hearing about one must not
// suppress the other.
func TestSubjectsAreSeparateConditions(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	seedServer(t, st, "usr_a", "srv_a", "box")

	for _, subject := range []string{"postgres", "redis"} {
		fresh, err := st.OpenAlert(ctx, Alert{
			ID: "alr_" + subject, UserID: "usr_a", ServerID: "srv_a",
			Kind: "backup-failed", Subject: subject,
		})
		if err != nil {
			t.Fatalf("open alert for %s: %v", subject, err)
		}
		if !fresh {
			t.Errorf("%s should be its own condition", subject)
		}
	}
}

// An account that has said nothing about notifications gets a usable answer
// rather than an error, because "nowhere to send" is a page state, not a fault.
func TestNotificationsDefaultToSomethingSensible(t *testing.T) {
	st := openTestStore(t)

	settings, err := st.GetNotifications(context.Background(), "usr_nobody")
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if settings.Email != "" {
		t.Error("an account that never set an address must not have one invented for it")
	}
	if settings.DiskPercent == 0 {
		t.Error("a zero threshold would report every disk as full the moment alerts were switched on")
	}
	if settings.Wants("disk-low") {
		t.Error("with no address, nothing can be wanted")
	}
}
