package ingest_test

import (
	"context"
	"errors"
	"testing"

	"takehome/internal/application/ingest"
	"takehome/internal/domain/inventory"
)

func TestServiceRunProcessesProductsAndEvents(t *testing.T) {
	repo := &fakeRepository{insertResults: []bool{true, false}}
	service := ingest.NewService(
		&fakeProductReader{products: []inventory.Product{{SKU: "SKU-0001", Name: "Small box"}}},
		&fakeEventReader{
			files: []string{"part-000.ndjson", "part-001.ndjson"},
			lines: map[string][]ingest.EventLine{
				"part-000.ndjson": {
					validLine("part-000.ndjson", 1, "evt-0001", "SKU-0001", "IN", 10),
					{SourceFile: "part-000.ndjson", LineNumber: 2, RawLine: "{", ParseError: "malformed json"},
				},
				"part-001.ndjson": {
					validLine("part-001.ndjson", 1, "evt-0002", "SKU-0001", "OUT", 3),
					validLine("part-001.ndjson", 2, "evt-0003", "SKU-9999", "IN", 1),
				},
			},
		},
		repo,
	)

	summary, err := service.Run(context.Background(), ingest.Options{
		ProductsCSV: "products.csv",
		EventsDir:   "events",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary.ProductsLoaded != 1 {
		t.Fatalf("expected 1 product loaded, got %d", summary.ProductsLoaded)
	}
	if summary.FilesProcessed != 2 {
		t.Fatalf("expected 2 files processed, got %d", summary.FilesProcessed)
	}
	if summary.EventsInserted != 1 {
		t.Fatalf("expected 1 inserted event, got %d", summary.EventsInserted)
	}
	if summary.Duplicates != 1 {
		t.Fatalf("expected 1 duplicate, got %d", summary.Duplicates)
	}
	if summary.InvalidLines != 2 {
		t.Fatalf("expected 2 invalid lines, got %d", summary.InvalidLines)
	}
	if len(repo.products) != 1 {
		t.Fatalf("expected 1 product upsert, got %d", len(repo.products))
	}
	if len(repo.movements) != 2 {
		t.Fatalf("expected 2 movement store attempts, got %d", len(repo.movements))
	}
	if len(repo.ingestErrors) != 2 {
		t.Fatalf("expected 2 ingest errors, got %d", len(repo.ingestErrors))
	}
}

func TestServiceRunReturnsProductReaderError(t *testing.T) {
	expectedErr := errors.New("read products failed")
	service := ingest.NewService(&fakeProductReader{err: expectedErr}, &fakeEventReader{}, &fakeRepository{})

	_, err := service.Run(context.Background(), ingest.Options{})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestServiceRunReturnsProductUpsertError(t *testing.T) {
	expectedErr := errors.New("upsert product failed")
	service := ingest.NewService(
		&fakeProductReader{products: []inventory.Product{{SKU: "SKU-0001", Name: "Small box"}}},
		&fakeEventReader{},
		&fakeRepository{upsertErr: expectedErr},
	)

	_, err := service.Run(context.Background(), ingest.Options{})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestServiceRunReturnsEventListingError(t *testing.T) {
	expectedErr := errors.New("list events failed")
	service := ingest.NewService(
		&fakeProductReader{products: []inventory.Product{{SKU: "SKU-0001", Name: "Small box"}}},
		&fakeEventReader{filesErr: expectedErr},
		&fakeRepository{},
	)

	_, err := service.Run(context.Background(), ingest.Options{})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestServiceRunReturnsErrorWhenNoEventFilesExist(t *testing.T) {
	service := ingest.NewService(
		&fakeProductReader{products: []inventory.Product{{SKU: "SKU-0001", Name: "Small box"}}},
		&fakeEventReader{},
		&fakeRepository{},
	)

	_, err := service.Run(context.Background(), ingest.Options{EventsDir: "events"})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceRunReturnsPartialSummaryOnMovementStoreError(t *testing.T) {
	expectedErr := errors.New("store movement failed")
	service := ingest.NewService(
		&fakeProductReader{products: []inventory.Product{{SKU: "SKU-0001", Name: "Small box"}}},
		&fakeEventReader{
			files: []string{"part-000.ndjson"},
			lines: map[string][]ingest.EventLine{
				"part-000.ndjson": {validLine("part-000.ndjson", 1, "evt-0001", "SKU-0001", "IN", 10)},
			},
		},
		&fakeRepository{storeErr: expectedErr},
	)

	summary, err := service.Run(context.Background(), ingest.Options{EventsDir: "events"})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if summary.ProductsLoaded != 1 {
		t.Fatalf("expected partial summary with 1 product loaded, got %d", summary.ProductsLoaded)
	}
	if summary.EventsInserted != 0 {
		t.Fatalf("expected 0 inserted events, got %d", summary.EventsInserted)
	}
}

func TestServiceRunReturnsErrorRecordingFailure(t *testing.T) {
	expectedErr := errors.New("record ingest error failed")
	service := ingest.NewService(
		&fakeProductReader{products: []inventory.Product{{SKU: "SKU-0001", Name: "Small box"}}},
		&fakeEventReader{
			files: []string{"part-000.ndjson"},
			lines: map[string][]ingest.EventLine{
				"part-000.ndjson": {{SourceFile: "part-000.ndjson", LineNumber: 1, RawLine: "{", ParseError: "malformed json"}},
			},
		},
		&fakeRepository{recordErr: expectedErr},
	)

	_, err := service.Run(context.Background(), ingest.Options{EventsDir: "events"})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestServiceRunRequiresDependencies(t *testing.T) {
	service := ingest.NewService(nil, nil, nil)

	_, err := service.Run(context.Background(), ingest.Options{})

	if err == nil {
		t.Fatal("expected error")
	}
}

func validLine(sourceFile string, lineNumber int, eventID, sku, movementType string, quantity int) ingest.EventLine {
	return ingest.EventLine{
		SourceFile: sourceFile,
		LineNumber: lineNumber,
		RawLine:    "{}",
		Input: inventory.MovementInput{
			EventID:    eventID,
			SKU:        sku,
			Type:       movementType,
			Quantity:   quantity,
			OccurredAt: "2026-06-01T02:12:46Z",
		},
	}
}

type fakeProductReader struct {
	products []inventory.Product
	err      error
}

func (reader *fakeProductReader) ReadProducts(context.Context, string) ([]inventory.Product, error) {
	return reader.products, reader.err
}

type fakeEventReader struct {
	files    []string
	filesErr error
	lines    map[string][]ingest.EventLine
}

func (reader *fakeEventReader) EventFiles(string) ([]string, error) {
	return reader.files, reader.filesErr
}

func (reader *fakeEventReader) ReadEvents(_ context.Context, path string, handle func(ingest.EventLine) error) error {
	for _, line := range reader.lines[path] {
		if err := handle(line); err != nil {
			return err
		}
	}
	return nil
}

type recordedIngestError struct {
	sourceFile string
	lineNumber int
	rawLine    string
	reason     string
}

type fakeRepository struct {
	products      []inventory.Product
	movements     []inventory.Movement
	ingestErrors  []recordedIngestError
	insertResults []bool
	upsertErr     error
	storeErr      error
	recordErr     error
}

func (repo *fakeRepository) UpsertProduct(_ context.Context, product inventory.Product) error {
	if repo.upsertErr != nil {
		return repo.upsertErr
	}
	repo.products = append(repo.products, product)
	return nil
}

func (repo *fakeRepository) StoreMovement(_ context.Context, movement inventory.Movement) (bool, error) {
	if repo.storeErr != nil {
		return false, repo.storeErr
	}
	repo.movements = append(repo.movements, movement)
	if len(repo.insertResults) == 0 {
		return true, nil
	}
	inserted := repo.insertResults[0]
	repo.insertResults = repo.insertResults[1:]
	return inserted, nil
}

func (repo *fakeRepository) RecordIngestError(_ context.Context, sourceFile string, lineNumber int, rawLine, reason string) error {
	if repo.recordErr != nil {
		return repo.recordErr
	}
	repo.ingestErrors = append(repo.ingestErrors, recordedIngestError{
		sourceFile: sourceFile,
		lineNumber: lineNumber,
		rawLine:    rawLine,
		reason:     reason,
	})
	return nil
}
