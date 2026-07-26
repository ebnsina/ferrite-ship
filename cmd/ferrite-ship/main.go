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

	"github.com/ebnsina/ferrite-ship/internal/api"
	"github.com/ebnsina/ferrite-ship/internal/config"
	"github.com/ebnsina/ferrite-ship/internal/files"
	"github.com/ebnsina/ferrite-ship/internal/runner"
	"github.com/ebnsina/ferrite-ship/internal/secret"
	"github.com/ebnsina/ferrite-ship/internal/services"
	"github.com/ebnsina/ferrite-ship/internal/store"
	"github.com/ebnsina/ferrite-ship/internal/terminal"
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

	if err := run(log); err != nil {
		log.Error("startup failed", "error", err)
		os.Exit(1)
	}
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

	bus := runner.NewBus()
	jobs := runner.New(st, sealer, bus, log)
	terminals := terminal.NewService(st, sealer)
	fileBrowser := files.NewService(st, sealer)
	units := services.NewService(st, sealer)

	restAPI := api.New(api.Options{
		Store:         st,
		Runner:        jobs,
		Bus:           bus,
		Sealer:        sealer,
		Terminals:     terminals,
		Files:         fileBrowser,
		Services:      units,
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
