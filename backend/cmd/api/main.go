package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	application "takehome/internal/application/inventory"
	"takehome/internal/config"
	"takehome/internal/infrastructure/database"
	transport "takehome/internal/infrastructure/http"
	"takehome/internal/infrastructure/persistence"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	if err := run(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config) error {
	// Open the shared PostgreSQL connection first; the API, migrations, and
	// repositories all use the same GORM handle and underlying connection pool.
	dbConn, err := database.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	// Keep ownership of the underlying sql.DB in main so process shutdown closes
	// database resources even though the rest of the app receives only GORM.
	sqlDB, err := dbConn.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// The API can start against a fresh database; migrations are idempotent and
	// skipped when their version is already recorded.
	if err := database.ApplyMigrations(ctx, dbConn, cfg.Migrations); err != nil {
		return err
	}

	// Compose the layers from the outside in: infrastructure implements the
	// repository, application owns the query use case, and HTTP exposes it.
	repo := persistence.NewInventoryRepository(dbConn)
	service := application.NewService(repo)
	app := transport.NewInventoryHandler(service).Routes()

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stdout, "api listening on %s\n", cfg.APIAddr)
		errCh <- app.Listen(cfg.APIAddr)
	}()

	// Wait for either the process context to be cancelled or Fiber to fail.
	// SIGINT/SIGTERM flow through ctx and trigger a graceful shutdown.
	select {
	case <-ctx.Done():
		return app.Shutdown()
	case err := <-errCh:
		return err
	}

}
