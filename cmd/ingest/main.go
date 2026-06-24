package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"takehome/internal/config"
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
	_ = ctx
	_ = cfg

	return nil
}
