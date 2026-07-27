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

	// A domain decides where a machine answers on the web and whose
	// certificates are issued for it, so pointing one at somebody else's
	// server is worth its own case rather than being assumed from the reads.
	t.Run("setting a domain refuses another owner's server", func(t *testing.T) {
		err := st.SetServerDomain(ctx, "usr_alice", bob, "example.com", "alice@example.com")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("alice set a domain on bob's server; got err %v, want ErrNotFound", err)
		}

		unchanged, err := st.GetServer(ctx, "usr_bob", bob)
		if err != nil {
			t.Fatalf("re-read bob's server: %v", err)
		}
		if unchanged.Domain != "" {
			t.Errorf("bob's server now answers at %q", unchanged.Domain)
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

// A saved query is written by hand and names real tables and columns, so
// handing one account another's is handing over their schema.
func TestSavedQueriesAreScoped(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	bobServer := seedServer(t, st, "usr_bob", "srv_bob", "bob-box")

	bobs := SavedQuery{
		ID: "qry_bob", UserID: "usr_bob", ServerID: bobServer, ToolID: "postgres",
		Name: "revenue", Query: "select sum(amount) from bobs_private_orders",
		SavedAt: time.Now().UTC(),
	}
	if err := st.SaveQuery(ctx, bobs); err != nil {
		t.Fatalf("save: %v", err)
	}

	found, err := st.ListSavedQueries(ctx, "usr_alice", bobServer, "postgres")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("alice sees %d of bob's saved queries, revealing his schema", len(found))
	}

	if err := st.DeleteSavedQuery(ctx, "usr_alice", bobServer, "qry_bob"); !errors.Is(err, ErrNotFound) {
		t.Errorf("alice deleted bob's saved query; got %v, want ErrNotFound", err)
	}

	mine, err := st.ListSavedQueries(ctx, "usr_bob", bobServer, "postgres")
	if err != nil || len(mine) != 1 {
		t.Fatalf("bob cannot read his own: %d rows, err %v", len(mine), err)
	}
}

// Saving under a name already used replaces it, which is what someone
// refining a query expects rather than a growing pile of near-duplicates.
func TestSavingTwiceUnderOneNameReplaces(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	server := seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	now := time.Now().UTC()

	first := SavedQuery{
		ID: "qry_1", UserID: "usr_alice", ServerID: server, ToolID: "postgres",
		Name: "signups", Query: "select 1", SavedAt: now,
	}
	if err := st.SaveQuery(ctx, first); err != nil {
		t.Fatalf("save: %v", err)
	}

	second := first
	second.ID = "qry_2"
	second.Query = "select count(*) from users"
	if err := st.SaveQuery(ctx, second); err != nil {
		t.Fatalf("resave: %v", err)
	}

	found, err := st.ListSavedQueries(ctx, "usr_alice", server, "postgres")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("got %d saved queries under one name, want 1", len(found))
	}
	if found[0].Query != second.Query {
		t.Errorf("query is %q, want the newer one", found[0].Query)
	}
}

// The failures page reaches across every server an account has, which makes it
// exactly the kind of query where a forgotten owner check leaks — it is the
// one page whose whole purpose is to show rows from servers the reader did not
// name.
func TestFailedJobsAreScoped(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	alice := seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	bob := seedServer(t, st, "usr_bob", "srv_bob", "bob-box")

	seedFailedJob(t, st, "job_alice", alice, "Alice's backup")
	seedFailedJob(t, st, "job_bob", bob, "Bob's backup")

	failures, err := st.ListFailedJobs(ctx, "usr_alice", 20)
	if err != nil {
		t.Fatalf("list failed jobs: %v", err)
	}
	if len(failures) != 1 {
		t.Fatalf("alice sees %d failures, want only her own", len(failures))
	}
	if failures[0].ID != "job_alice" {
		t.Errorf("alice sees %q", failures[0].ID)
	}
}

// Only failures, and every failure. A run that succeeded appearing here would
// be alarming; one that failed being missing is worse.
func TestFailedJobsAreOnlyTheFailures(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	server := seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	seedFailedJob(t, st, "job_failed", server, "This broke")

	ok := Job{
		ID: "job_ok", ServerID: server, Kind: "setup", Title: "This worked",
		Actor: "alice@example.com", Status: JobSucceeded, StartedAt: time.Now().UTC(),
	}
	if err := st.CreateJob(ctx, ok); err != nil {
		t.Fatalf("create job: %v", err)
	}

	failures, err := st.ListFailedJobs(ctx, "usr_alice", 20)
	if err != nil {
		t.Fatalf("list failed jobs: %v", err)
	}
	if len(failures) != 1 || failures[0].ID != "job_failed" {
		t.Fatalf("got %d jobs, want just the failed one", len(failures))
	}
	if failures[0].Error == "" {
		t.Error("a failure with no error text tells the reader nothing")
	}
}

func seedFailedJob(t *testing.T, st *Store, id, serverID, title string) {
	t.Helper()

	ctx := context.Background()
	job := Job{
		ID: id, ServerID: serverID, Kind: "backup", Title: title,
		Actor: ActorScheduled, Status: JobRunning, StartedAt: time.Now().UTC(),
	}
	if err := st.CreateJob(ctx, job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	finished := time.Now().UTC()
	job.Status = JobFailed
	job.FinishedAt = &finished
	job.Error = "could not reach the database"
	if err := st.FinishJob(ctx, job); err != nil {
		t.Fatalf("finish job: %v", err)
	}
}

// A GitHub installation is the right to mint a token that reads somebody's
// private repositories. Handing one account another's is handing over their
// source code, so it is scoped like a credential rather than like a label.
func TestGitHubInstallationsAreScoped(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	seedServer(t, st, "usr_bob", "srv_bob", "bob-box")

	bobs := GitHubInstallation{ID: 4242, UserID: "usr_bob", Account: "bob-corp", Selection: "selected"}
	if err := st.SaveGitHubInstallation(ctx, bobs); err != nil {
		t.Fatalf("save: %v", err)
	}

	t.Run("list is scoped", func(t *testing.T) {
		found, err := st.ListGitHubInstallations(ctx, "usr_alice")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(found) != 0 {
			t.Errorf("alice sees %d of bob's installations", len(found))
		}
	})

	t.Run("get refuses another owner's installation", func(t *testing.T) {
		if _, err := st.GetGitHubInstallation(ctx, "usr_alice", 4242); !errors.Is(err, ErrNotFound) {
			t.Errorf("alice reached bob's repositories; got err %v, want ErrNotFound", err)
		}
	})

	t.Run("delete refuses another owner's installation", func(t *testing.T) {
		if err := st.DeleteGitHubInstallation(ctx, "usr_alice", 4242); !errors.Is(err, ErrNotFound) {
			t.Errorf("alice disconnected bob's github; got err %v, want ErrNotFound", err)
		}
		if _, err := st.GetGitHubInstallation(ctx, "usr_bob", 4242); err != nil {
			t.Errorf("bob's installation went missing: %v", err)
		}
	})

	// Re-confirming is normal: it happens every time somebody changes which
	// repositories are shared, and must update rather than fail or duplicate.
	t.Run("reconnecting updates rather than duplicates", func(t *testing.T) {
		again := bobs
		again.Selection = "all"
		if err := st.SaveGitHubInstallation(ctx, again); err != nil {
			t.Fatalf("re-save: %v", err)
		}

		found, err := st.ListGitHubInstallations(ctx, "usr_bob")
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(found) != 1 {
			t.Fatalf("bob has %d rows for one installation", len(found))
		}
		if found[0].Selection != "all" {
			t.Errorf("selection is %q, want the updated value", found[0].Selection)
		}
	})
}

// Alerts are written by the watcher, not by a request, so a scoping hole here
// would not be caught by anything above it. Clearing is the dangerous half: it
// is how a condition stops being announced, and one account silencing
// another's "server is down" is worse than reading it.
func TestAlertWritesAreScoped(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	seedServer(t, st, "usr_alice", "srv_alice", "alice-box")
	seedServer(t, st, "usr_bob", "srv_bob", "bob-box")

	bobs := Alert{ID: "alr_bob", UserID: "usr_bob", ServerID: "srv_bob", Kind: "server-down"}
	if fresh, err := st.OpenAlert(ctx, bobs); err != nil || !fresh {
		t.Fatalf("open bob's alert: fresh=%v err=%v", fresh, err)
	}

	t.Run("another owner cannot clear it", func(t *testing.T) {
		was, err := st.ClearAlert(ctx, "usr_alice", "srv_bob", "server-down", "")
		if err != nil {
			t.Fatalf("clear: %v", err)
		}
		if was {
			t.Error("alice silenced bob's alert")
		}

		if open, err := st.OpenAlerts(ctx, "usr_bob"); err != nil || len(open) != 1 {
			t.Errorf("bob's alert is gone: %d open, err %v", len(open), err)
		}
	})

	t.Run("the owner can", func(t *testing.T) {
		was, err := st.ClearAlert(ctx, "usr_bob", "srv_bob", "server-down", "")
		if err != nil {
			t.Fatalf("clear: %v", err)
		}
		if !was {
			t.Error("bob could not clear his own alert")
		}
	})

	// The de-duplication itself still holds with the owner in the query — the
	// same condition twice must open once. (Two accounts cannot share a
	// condition: a server has exactly one owner, which is what makes the
	// unique index on server_id sufficient.)
	t.Run("the same condition twice opens once", func(t *testing.T) {
		shared := Alert{ID: "alr_a", UserID: "usr_alice", ServerID: "srv_alice", Kind: "disk-low"}
		if fresh, err := st.OpenAlert(ctx, shared); err != nil || !fresh {
			t.Fatalf("alice's alert: fresh=%v err=%v", fresh, err)
		}

		again := shared
		again.ID = "alr_a2"
		if fresh, err := st.OpenAlert(ctx, again); err != nil || fresh {
			t.Errorf("the same condition opened twice for one account: fresh=%v err=%v", fresh, err)
		}
	})
}
