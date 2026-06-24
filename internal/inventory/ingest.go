package inventory

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type IngestOptions struct {
	ProductsCSV string
	EventsDir   string
}

type IngestSummary struct {
	ProductsLoaded int
	FilesProcessed int
	EventsInserted int
	Duplicates     int
	InvalidLines   int
}

type rawEvent struct {
	EventID    string `json:"event_id"`
	SKU        string `json:"sku"`
	Type       string `json:"type"`
	Quantity   int    `json:"quantity"`
	OccurredAt string `json:"occurred_at"`
}

type movement struct {
	EventID    string
	SKU        string
	Type       string
	Quantity   int
	OccurredAt time.Time
}

func RunIngest(ctx context.Context, database *gorm.DB, opts IngestOptions) (IngestSummary, error) {
	if database == nil {
		return IngestSummary{}, errors.New("database is required")
	}

	products, err := loadProducts(ctx, database, opts.ProductsCSV)
	if err != nil {
		return IngestSummary{}, err
	}

	summary := IngestSummary{ProductsLoaded: len(products)}
	if err := processEventFiles(ctx, database, opts.EventsDir, products, &summary); err != nil {
		return summary, err
	}

	return summary, nil
}

func loadProducts(ctx context.Context, database *gorm.DB, path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open products csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read products csv header: %w", err)
	}
	if len(header) != 2 || header[0] != "sku" || header[1] != "name" {
		return nil, fmt.Errorf("unexpected products csv header: %s", strings.Join(header, ","))
	}

	products := make(map[string]struct{})
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read products csv: %w", err)
		}
		if len(record) != 2 {
			return nil, fmt.Errorf("invalid products csv record: %v", record)
		}

		sku := strings.TrimSpace(record[0])
		name := strings.TrimSpace(record[1])
		if sku == "" || name == "" {
			return nil, fmt.Errorf("invalid product record: sku and name are required")
		}

		if err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`
				INSERT INTO products (sku, name)
				VALUES (?, ?)
				ON CONFLICT (sku) DO UPDATE SET name = EXCLUDED.name
			`, sku, name).Error; err != nil {
				return err
			}

			return tx.Exec(`
				INSERT INTO product_stock (sku)
				VALUES (?)
				ON CONFLICT (sku) DO NOTHING
			`, sku).Error
		}); err != nil {
			return nil, fmt.Errorf("store product %s: %w", sku, err)
		}

		products[sku] = struct{}{}
	}

	return products, nil
}

func processEventFiles(ctx context.Context, database *gorm.DB, eventsDir string, products map[string]struct{}, summary *IngestSummary) error {
	paths, err := filepath.Glob(filepath.Join(eventsDir, "*.ndjson"))
	if err != nil {
		return fmt.Errorf("list event files: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no event files found in %s", eventsDir)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
	)

	for _, path := range paths {
		path := path
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := processEventFile(ctx, database, path, products, summary, &mu); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel()
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return firstErr
}

func processEventFile(ctx context.Context, database *gorm.DB, path string, products map[string]struct{}, summary *IngestSummary, mu *sync.Mutex) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open event file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		if err := ctx.Err(); err != nil {
			return err
		}

		rawLine := scanner.Text()
		event, reason := parseMovement(rawLine, products)
		if reason != "" {
			if err := recordIngestError(ctx, database, path, lineNumber, rawLine, reason); err != nil {
				return err
			}
			addInvalidLine(summary, mu)
			continue
		}

		inserted, err := storeMovement(ctx, database, event)
		if err != nil {
			return err
		}
		addProcessedEvent(summary, mu, inserted)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan event file %s: %w", path, err)
	}

	mu.Lock()
	summary.FilesProcessed++
	mu.Unlock()

	return nil
}

func parseMovement(rawLine string, products map[string]struct{}) (movement, string) {
	var event rawEvent
	if err := json.Unmarshal([]byte(rawLine), &event); err != nil {
		return movement{}, "malformed json"
	}
	if strings.TrimSpace(event.EventID) == "" {
		return movement{}, "event_id is required"
	}
	if _, ok := products[event.SKU]; !ok {
		return movement{}, "unknown sku"
	}
	if event.Type != "IN" && event.Type != "OUT" {
		return movement{}, "invalid movement type"
	}
	if event.Quantity <= 0 {
		return movement{}, "quantity must be positive"
	}

	occurredAt, err := time.Parse(time.RFC3339, event.OccurredAt)
	if err != nil {
		return movement{}, "invalid occurred_at"
	}

	return movement{
		EventID:    event.EventID,
		SKU:        event.SKU,
		Type:       event.Type,
		Quantity:   event.Quantity,
		OccurredAt: occurredAt,
	}, ""
}

func storeMovement(ctx context.Context, database *gorm.DB, event movement) (bool, error) {
	inserted := false
	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
			INSERT INTO inventory_movements (event_id, sku, movement_type, quantity, occurred_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (event_id) DO NOTHING
		`, event.EventID, event.SKU, event.Type, event.Quantity, event.OccurredAt)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		delta := event.Quantity
		if event.Type == "OUT" {
			delta = -delta
		}

		if err := tx.Exec(`
			UPDATE product_stock
			SET quantity = quantity + ?, updated_at = now()
			WHERE sku = ?
		`, delta, event.SKU).Error; err != nil {
			return err
		}

		inserted = true
		return nil
	})

	return inserted, err
}

func recordIngestError(ctx context.Context, database *gorm.DB, sourceFile string, lineNumber int, rawLine, reason string) error {
	return database.WithContext(ctx).Exec(`
		INSERT INTO ingest_errors (source_file, line_number, raw_line, reason)
		VALUES (?, ?, ?, ?)
	`, sourceFile, lineNumber, rawLine, reason).Error
}

func addInvalidLine(summary *IngestSummary, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	summary.InvalidLines++
}

func addProcessedEvent(summary *IngestSummary, mu *sync.Mutex, inserted bool) {
	mu.Lock()
	defer mu.Unlock()

	if inserted {
		summary.EventsInserted++
		return
	}
	summary.Duplicates++
}
