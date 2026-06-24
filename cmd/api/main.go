package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"takehome/internal/config"
	"takehome/internal/db"
	"takehome/internal/inventory"
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
	database, err := db.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	sqlDB, err := database.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := db.ApplyMigrations(ctx, database, cfg.Migrations); err != nil {
		return err
	}

	app := inventory.NewAPI(database).Routes()

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
