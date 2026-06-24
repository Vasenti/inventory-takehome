package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"takehome/internal/application/ingest"
	"takehome/internal/config"
	"takehome/internal/infrastructure/database"
	"takehome/internal/infrastructure/files"
	"takehome/internal/infrastructure/persistence"
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
	// Open PostgreSQL once and pass the GORM handle to infrastructure adapters.
	// The pool limit is enforced inside infrastructure/database.
	dbConn, err := database.OpenPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}

	// main owns process-level resources, so it also closes the underlying
	// database handle before exiting.
	sqlDB, err := dbConn.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Ingest can be run on a fresh database; already-applied migrations are
	// tracked in schema_migrations and skipped.
	if err := database.ApplyMigrations(ctx, dbConn, cfg.Migrations); err != nil {
		return err
	}

	// Compose the ingestion use case with concrete adapters. The application
	// layer coordinates readers and persistence without knowing about files or SQL.
	repo := persistence.NewInventoryRepository(dbConn)
	service := ingest.NewService(files.ProductCSVReader{}, files.NDJSONEventReader{}, repo)

	summary, err := service.Run(ctx, ingest.Options{
		ProductsCSV: cfg.ProductsCSV,
		EventsDir:   cfg.EventsDir,
	})
	if err != nil {
		return err
	}

	// Print a compact operational summary that is useful for manual validation
	// and for checking duplicate/invalid-line behavior on generated datasets.
	fmt.Fprintf(os.Stdout, "products loaded: %d\n", summary.ProductsLoaded)
	fmt.Fprintf(os.Stdout, "files processed: %d\n", summary.FilesProcessed)
	fmt.Fprintf(os.Stdout, "events inserted: %d\n", summary.EventsInserted)
	fmt.Fprintf(os.Stdout, "duplicates skipped: %d\n", summary.Duplicates)
	fmt.Fprintf(os.Stdout, "invalid lines: %d\n", summary.InvalidLines)

	return nil
}
