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
		fmt.Fprintln(os.Stderr, "ingest:", err)
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

	summary, err := inventory.RunIngest(ctx, database, inventory.IngestOptions{
		ProductsCSV: cfg.ProductsCSV,
		EventsDir:   cfg.EventsDir,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "products loaded: %d\n", summary.ProductsLoaded)
	fmt.Fprintf(os.Stdout, "files processed: %d\n", summary.FilesProcessed)
	fmt.Fprintf(os.Stdout, "events inserted: %d\n", summary.EventsInserted)
	fmt.Fprintf(os.Stdout, "duplicates skipped: %d\n", summary.Duplicates)
	fmt.Fprintf(os.Stdout, "invalid lines: %d\n", summary.InvalidLines)

	return nil
}
