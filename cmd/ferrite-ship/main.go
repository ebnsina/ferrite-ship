// Command ferrite-ship is the control plane: an HTTP API, a job runner, and
// (optionally) the built dashboard.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebnsina/ferrite-ship/internal/alerts"
	"github.com/ebnsina/ferrite-ship/internal/api"
	"github.com/ebnsina/ferrite-ship/internal/auth"
	"github.com/ebnsina/ferrite-ship/internal/catalog"
	"github.com/ebnsina/ferrite-ship/internal/config"
	"github.com/ebnsina/ferrite-ship/internal/console"
	"github.com/ebnsina/ferrite-ship/internal/dialer"
	"github.com/ebnsina/ferrite-ship/internal/files"
	"github.com/ebnsina/ferrite-ship/internal/github"
	"github.com/ebnsina/ferrite-ship/internal/notify"
	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/scheduler"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/services"
	"github.com/ebnsina/ferrite-ship/internal/store"
	"github.com/ebnsina/ferrite-ship/internal/terminal"
	"github.com/ebnsina/ferrite-ship/internal/watch"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if len(os.Args) > 1 && os.Args[1] == "genkey" {
		key, err := secret.GenerateKey()
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not generate a key:", err)
			os.Exit(1)
		}
		fmt.Println(key)
		return
	}

	if len(os.Args) > 2 && os.Args[1] == "adduser" {
		if err := addUser(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "reset-account" {
		if err := resetAccount(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Account removed. Open the dashboard to create a new one.")
		return
	}

	if err := run(log); err != nil {
		log.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

// addUser creates an account with a generated password, printed once.
//
// The password is generated rather than accepted as an argument: an argument
// would sit in shell history and in the process list for anyone on the box to
// read.
func addUser(email string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	password, err := auth.GeneratePassword()
	if err != nil {
		return err
	}

	ctx := context.Background()
	accounts := auth.NewService(st)

	user, err := accounts.Create(ctx, email, password)
	if errors.Is(err, store.ErrEmailTaken) {
		return fmt.Errorf("%s already has an account. Use `reset-account` to start over", email)
	}
	if err != nil {
		return err
	}

	// Servers created before ownership existed belong to nobody. Hand them to
	// the first account, so an existing install does not appear to lose them.
	claimed, err := st.ClaimUnownedServers(ctx, user.ID)
	if err != nil {
		return err
	}

	fmt.Printf("Account created.\n\n  email:    %s\n  password: %s\n\n", user.Email, password)
	if claimed > 0 {
		fmt.Printf("Also gave this account the %d server(s) that had no owner.\n", claimed)
	}
	fmt.Println("Write the password down — it is not stored anywhere readable and is not shown again.")
	return nil
}

// resetAccount is the way back in after a forgotten password. It needs the
// database file, so it proves control of the machine rather than of an inbox.
func resetAccount() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	return st.DeleteAllUsers(context.Background())
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	sealer, err := secret.NewSealer(cfg.SecretKey)
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	// One dialer for everything that reaches a managed server, so host key
	// checking happens in exactly one place.
	connections := dialer.New(st, sealer)
	consoles := console.New(connections)

	bus := runner.NewBus()
	jobs := runner.New(st, connections, bus, sealer, log)
	terminals := terminal.NewService(connections)
	fileBrowser := files.NewService(connections)
	units := services.NewService(connections)
	accounts := auth.NewService(st)

	// Where news of a failure goes. With no mail server configured this still
	// records alerts — the dashboard shows them — and sends nothing, which the
	// settings page says out loud rather than implying.
	reporter := alerts.New(st, notify.New(cfg.SMTP), cfg.PublicURL, log)
	jobs.Reporting(reporter)
	jobs.Certificates(cfg.ACMEDirectory)
	if cfg.SMTP.Enabled() {
		log.Info("email alerts enabled", "host", cfg.SMTP.Host, "from", cfg.SMTP.From)
	} else {
		log.Info("email alerts disabled (FERRITE_SMTP_URL is none)")
	}

	// Worth a line at startup: staging certificates make browsers warn, and
	// somebody seeing that warning should be able to find out why from the log
	// rather than by reading the compose file on the server.
	if cfg.ACMEDirectory == catalog.ACMEStaging {
		log.Info("certificates come from Let's Encrypt STAGING; browsers will warn about them")
	} else {
		log.Info("certificates come from Let's Encrypt production")
	}

	// Built at startup, not at the first clone. A key that cannot be read is
	// then a refusal to start with the variable named, rather than a deploy
	// that fails hours later against a repository that looks fine.
	var repositories *github.App
	if cfg.GitHub.Enabled() {
		repositories, err = github.New(cfg.GitHub.AppID, cfg.GitHub.Slug, cfg.GitHub.PrivateKey)
		if err != nil {
			return err
		}
		log.Info("github app enabled", "app", cfg.GitHub.Slug, "id", cfg.GitHub.AppID)
	} else {
		log.Info("github app disabled (FERRITE_GITHUB_APP_ID is none); " +
			"private repositories need a deploy key")
	}

	restAPI := api.New(api.Options{
		Store:         st,
		Runner:        jobs,
		Bus:           bus,
		Sealer:        sealer,
		Terminals:     terminals,
		Files:         fileBrowser,
		Services:      units,
		Console:       consoles,
		Dialer:        connections,
		Auth:          accounts,
		Alerts:        reporter,
		GitHub:        repositories,
		PublicURL:     cfg.PublicURL,
		Logger:        log,
		AllowedOrigin: cfg.AllowedOrigin,
	})

	mux := http.NewServeMux()
	mux.Handle("/v1/", restAPI.Routes())

	// Serving the dashboard is opt-in: during development Vite serves it on
	// its own port, and the API only needs to allow that origin.
	if cfg.WebDir != "" {
		spa, err := api.SPAHandler(cfg.WebDir)
		if err != nil {
			return fmt.Errorf("could not serve the dashboard from %q: %w", cfg.WebDir, err)
		}
		mux.Handle("/", spa)
		log.Info("serving dashboard", "dir", cfg.WebDir)
	} else {
		log.Info("dashboard not served (FERRITE_WEB_DIR is empty); API only")
	}

	server := &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
		// Long, because SSE connections are meant to stay open. The runner has
		// its own timeout for the work itself.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Scheduled backups run in this process. Given the same context as the
	// server so a shutdown stops both, and started before the listener so a
	// backup that was due while the process was down goes as soon as it is up.
	schedules := scheduler.New(st, jobs, log)
	go schedules.Run(ctx)

	// Nothing else looks at a server unless somebody asks it to, so without
	// this a machine that stopped answering stays "online" until the next time
	// a job happens to run against it.
	go watch.New(st, connections, reporter, log).Run(ctx)

	errs := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
