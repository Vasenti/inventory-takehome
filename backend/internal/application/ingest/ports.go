package ingest

import (
	"context"

	"takehome/internal/domain/inventory"
)

type Options struct {
	ProductsCSV string
	EventsDir   string
}

type Summary struct {
	ProductsLoaded int
	FilesProcessed int
	EventsInserted int
	Duplicates     int
	InvalidLines   int
}

type EventLine struct {
	SourceFile string
	LineNumber int
	RawLine    string
	Input      inventory.MovementInput
	ParseError string
}

type ProductReader interface {
	ReadProducts(ctx context.Context, path string) ([]inventory.Product, error)
}

type EventReader interface {
	EventFiles(eventsDir string) ([]string, error)
	ReadEvents(ctx context.Context, path string, handle func(EventLine) error) error
}

type Repository interface {
	UpsertProduct(ctx context.Context, product inventory.Product) error
	StoreMovement(ctx context.Context, movement inventory.Movement) (bool, error)
	RecordIngestError(ctx context.Context, sourceFile string, lineNumber int, rawLine, reason string) error
}
