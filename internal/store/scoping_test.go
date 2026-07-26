package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Ownership is enforced in SQL, not by filtering afterwards. SQLite has no
// row-level security, so until this moves to PostgreSQL these tests are the
// guarantee — they exist to fail loudly if a query ever stops carrying the
// owner id.
func TestOneAccountCannotReachAnothersServers(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	alice := seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	bob := seedServer(t, st, "usr_bob", "srv_bob", "bob-box")

	t.Run("list is scoped", func(t *testing.T) {
		servers, err := st.ListServers(ctx, "usr_alice")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(servers) != 1 || servers[0].ID != alice {
			t.Fatalf("alice sees %d servers, want only her own", len(servers))
		}
	})

	t.Run("get refuses another owner's server", func(t *testing.T) {
		if _, err := st.GetServer(ctx, "usr_alice", bob); !errors.Is(err, ErrNotFound) {
			t.Errorf("alice fetched bob's server; got err %v, want ErrNotFound", err)
		}
	})

	t.Run("delete refuses another owner's server", func(t *testing.T) {
		if err := st.DeleteServer(ctx, "usr_alice", bob); !errors.Is(err, ErrNotFound) {
			t.Errorf("alice deleted bob's server; got err %v, want ErrNotFound", err)
		}
		// And it is still there.
		if _, err := st.GetServer(ctx, "usr_bob", bob); err != nil {
			t.Errorf("bob's server went missing: %v", err)
		}
	})
}

func TestJobsAreVisibleOnlyThroughAServerYouOwn(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	bobServer := seedServer(t, st, "usr_bob", "srv_bob", "bob-box")

	job := Job{
		ID:        "job_bob",
		ServerID:  bobServer,
		Kind:      "baseline",
		Title:     "Setting up bob-box",
		Status:    JobSucceeded,
		StartedAt: time.Now().UTC(),
	}
	if err := st.CreateJob(ctx, job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	if _, err := st.GetJob(ctx, "usr_alice", job.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("alice read bob's job; got err %v, want ErrNotFound", err)
	}
	if _, err := st.GetJob(ctx, "usr_bob", job.ID); err != nil {
		t.Errorf("bob could not read his own job: %v", err)
	}

	recent, err := st.ListRecentJobs(ctx, "usr_alice", 10)
	if err != nil {
		t.Fatalf("list recent: %v", err)
	}
	if len(recent) != 0 {
		t.Errorf("alice's activity feed shows %d of bob's jobs", len(recent))
	}
}

func TestFleetSamplesAreScoped(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	seedServer(t, st, "usr_bob", "srv_bob1", "bob-one")
	seedServer(t, st, "usr_bob", "srv_bob2", "bob-two")

	for _, user := range []string{"usr_alice", "usr_bob"} {
		for range 2 {
			if err := st.SampleFleet(ctx, user); err != nil {
				t.Fatalf("sample %s: %v", user, err)
			}
		}
	}

	alice, err := st.RecentSamples(ctx, "usr_alice", 10)
	if err != nil {
		t.Fatalf("recent samples: %v", err)
	}
	for _, sample := range alice {
		if sample.ServerCount != 1 {
			t.Errorf("alice's snapshot counts %d servers; it is counting bob's too", sample.ServerCount)
		}
	}
}

// Servers created before ownership existed are adopted by the first account,
// so an existing single-user install does not appear to lose them.
func TestUnownedServersAreClaimed(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedServer(t, st, "", "srv_orphan", "predates-ownership")

	claimed, err := st.ClaimUnownedServers(ctx, "usr_first")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed != 1 {
		t.Fatalf("claimed %d servers, want 1", claimed)
	}

	servers, err := st.ListServers(ctx, "usr_first")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(servers) != 1 {
		t.Errorf("the claimed server is not visible to its new owner")
	}

	// Claiming again must not take servers that now belong to somebody.
	again, err := st.ClaimUnownedServers(ctx, "usr_second")
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if again != 0 {
		t.Errorf("a second account claimed %d already-owned servers", again)
	}
}

// Installations hold a database password, so a scoping hole here hands one
// account the credentials to another account's Postgres.
func TestInstallationsAreScoped(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	bobServer := seedServer(t, st, "usr_bob", "srv_bob", "bob-box")

	now := time.Now().UTC()
	bobsPostgres := Installation{
		ID: "ins_bob", UserID: "usr_bob", ServerID: bobServer, ToolID: "postgres",
		Version: "18", Status: InstallReady, SealedPassword: "sealed-secret",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveInstallation(ctx, bobsPostgres); err != nil {
		t.Fatalf("save installation: %v", err)
	}

	t.Run("list is scoped", func(t *testing.T) {
		found, err := st.ListInstallations(ctx, "usr_alice", bobServer)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(found) != 0 {
			t.Errorf("alice sees %d of bob's installations", len(found))
		}
	})

	t.Run("get refuses another owner's installation", func(t *testing.T) {
		if _, err := st.GetInstallation(ctx, "usr_alice", bobServer, "postgres"); !errors.Is(err, ErrNotFound) {
			t.Errorf("alice read bob's database password; got err %v, want ErrNotFound", err)
		}
	})

	t.Run("delete refuses another owner's installation", func(t *testing.T) {
		if err := st.DeleteInstallation(ctx, "usr_alice", bobServer, "postgres"); !errors.Is(err, ErrNotFound) {
			t.Errorf("alice deleted bob's installation; got err %v, want ErrNotFound", err)
		}
		if _, err := st.GetInstallation(ctx, "usr_bob", bobServer, "postgres"); err != nil {
			t.Errorf("bob's installation went missing: %v", err)
		}
	})
}

// Re-running an install to repair it must not change the password the owner's
// application is already connecting with.
func TestRepairingAnInstallKeepsThePassword(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	server := seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	now := time.Now().UTC()

	first := Installation{
		ID: "ins_1", UserID: "usr_alice", ServerID: server, ToolID: "postgres",
		Status: InstallReady, SealedPassword: "the-original", CreatedAt: now, UpdatedAt: now,
	}
	if err := st.SaveInstallation(ctx, first); err != nil {
		t.Fatalf("save: %v", err)
	}

	repair := first
	repair.Status = InstallPending
	repair.SealedPassword = "" // no new credential generated
	if err := st.SaveInstallation(ctx, repair); err != nil {
		t.Fatalf("repair: %v", err)
	}

	got, err := st.GetInstallation(ctx, "usr_alice", server, "postgres")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SealedPassword != "the-original" {
		t.Errorf("password became %q; repairing an install must keep the one in use", got.SealedPassword)
	}
	if got.Status != InstallPending {
		t.Errorf("status is %q, want the repair's own status", got.Status)
	}
}

// Removing a server must not leave its installations, and their passwords,
// behind in the database.
func TestInstallationsGoWithTheirServer(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	server := seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	now := time.Now().UTC()

	err := st.SaveInstallation(ctx, Installation{
		ID: "ins_1", UserID: "usr_alice", ServerID: server, ToolID: "redis",
		Status: InstallReady, SealedPassword: "sealed", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := st.DeleteServer(ctx, "usr_alice", server); err != nil {
		t.Fatalf("delete server: %v", err)
	}

	found, err := st.ListInstallations(ctx, "usr_alice", server)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("%d installations outlived their server, still holding credentials", len(found))
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func seedServer(t *testing.T, st *Store, userID, id, name string) string {
	t.Helper()

	server := Server{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Kind:      ConnectionDemo,
		Status:    StatusOnline,
		Services:  []string{},
		CreatedAt: time.Now().UTC(),
	}
	if err := st.CreateServer(context.Background(), server); err != nil {
		t.Fatalf("create server: %v", err)
	}
	return id
}
