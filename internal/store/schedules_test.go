package store

import (
	"testing"
	"time"
)

func at(day int, hour int) time.Time {
	// July 2026: the 1st is a Wednesday, so the 5th is a Sunday.
	return time.Date(2026, 7, day, hour, 0, 0, 0, time.UTC)
}

// The next run must always be strictly in the future. A boundary that returns
// "now" would have the scheduler fire the same backup on every tick.
func TestNextRunIsAlwaysAhead(t *testing.T) {
	cases := []struct {
		name     string
		schedule BackupSchedule
		from     time.Time
		want     time.Time
	}{
		{
			name:     "daily, later today",
			schedule: BackupSchedule{Cadence: Daily, Hour: 3},
			from:     at(10, 1),
			want:     at(10, 3),
		},
		{
			name:     "daily, already passed today",
			schedule: BackupSchedule{Cadence: Daily, Hour: 3},
			from:     at(10, 5),
			want:     at(11, 3),
		},
		{
			// Exactly on the hour counts as passed, or the run that just
			// happened would be scheduled again immediately.
			name:     "daily, exactly now",
			schedule: BackupSchedule{Cadence: Daily, Hour: 3},
			from:     at(10, 3),
			want:     at(11, 3),
		},
		{
			// The 5th is a Sunday; from Friday the 3rd it is two days away.
			name:     "weekly, later this week",
			schedule: BackupSchedule{Cadence: Weekly, Hour: 4, Weekday: int(time.Sunday)},
			from:     at(3, 9),
			want:     at(5, 4),
		},
		{
			name:     "weekly, that day has passed",
			schedule: BackupSchedule{Cadence: Weekly, Hour: 4, Weekday: int(time.Sunday)},
			from:     at(5, 9),
			want:     at(12, 4),
		},
		{
			name:     "weekly, today but earlier",
			schedule: BackupSchedule{Cadence: Weekly, Hour: 4, Weekday: int(time.Sunday)},
			from:     at(5, 1),
			want:     at(5, 4),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.schedule.NextRun(tc.from)
			if !got.Equal(tc.want) {
				t.Errorf("next run is %s, want %s", got.Format(time.RFC3339), tc.want.Format(time.RFC3339))
			}
			if !got.After(tc.from) {
				t.Errorf("next run %s is not after %s — the scheduler would loop",
					got.Format(time.RFC3339), tc.from.Format(time.RFC3339))
			}
		})
	}
}

// A process that was stopped over several runs must not fire all of them when
// it comes back: it moves to the next one ahead and carries on.
func TestAMissedRunDoesNotPileUp(t *testing.T) {
	schedule := BackupSchedule{Cadence: Daily, Hour: 3}

	// Due on the 10th, but nothing ran until the 14th.
	next := schedule.NextRun(at(14, 6))
	if !next.Equal(at(15, 3)) {
		t.Errorf("after a four-day outage the next run is %s, want the 15th at 03:00",
			next.Format(time.RFC3339))
	}
}

// Retention keeps the newest and returns the rest, which is what gets deleted.
func TestExpiredBackupsLeavesTheNewest(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()

	server := seedServer(t, st, "usr_alice", "srv_alice", "alice-box")

	for i := range 5 {
		err := st.CreateBackup(ctx, Backup{
			ID:        "bak_" + string(rune('a'+i)),
			UserID:    "usr_alice",
			ServerID:  server,
			ToolID:    "postgres",
			ObjectKey: "k/" + string(rune('a'+i)),
			Status:    BackupReady,
			CreatedAt: at(10+i, 3),
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	expired, err := st.ExpiredBackups(ctx, server, "postgres", 3)
	if err != nil {
		t.Fatalf("expired: %v", err)
	}
	if len(expired) != 2 {
		t.Fatalf("got %d expired, want 2 of 5 when keeping 3", len(expired))
	}
	// The two oldest, which are the ones created first.
	for _, b := range expired {
		if b.ID == "bak_e" || b.ID == "bak_d" || b.ID == "bak_c" {
			t.Errorf("%s is among the newest three and must be kept", b.ID)
		}
	}
}
