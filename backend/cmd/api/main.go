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
	dbConn, err := database.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := database.ApplyMigrations(ctx, dbConn, cfg.Migrations); err != nil {
		return err
	}

	repo := persistence.NewInventoryRepository(dbConn)
	service := application.NewService(repo)
	app := transport.NewInventoryHandler(service).Routes()

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stdout, "api listening on %s\n", cfg.APIAddr)
		errCh <- app.Listen(cfg.APIAddr)
	}()

	select {
	case <-ctx.Done():
		return app.Shutdown()
	case err := <-errCh:
		return err
	}

}
