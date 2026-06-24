package ingest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"takehome/internal/domain/inventory"
)

type Service struct {
	products ProductReader
	events   EventReader
	repo     Repository
}

func NewService(products ProductReader, events EventReader, repo Repository) *Service {
	return &Service{
		products: products,
		events:   events,
		repo:     repo,
	}
}

func (service *Service) Run(ctx context.Context, opts Options) (Summary, error) {
	if service.products == nil || service.events == nil || service.repo == nil {
		return Summary{}, errors.New("ingest service dependencies are required")
	}

	products, err := service.products.ReadProducts(ctx, opts.ProductsCSV)
	if err != nil {
		return Summary{}, err
	}

	knownProducts := make(map[string]struct{}, len(products))
	for _, product := range products {
		if err := service.repo.UpsertProduct(ctx, product); err != nil {
			return Summary{}, fmt.Errorf("store product %s: %w", product.SKU, err)
		}
		knownProducts[product.SKU] = struct{}{}
	}

	summary := Summary{ProductsLoaded: len(products)}
	if err := service.processEventFiles(ctx, opts.EventsDir, knownProducts, &summary); err != nil {
		return summary, err
	}

	return summary, nil
}

func (service *Service) processEventFiles(ctx context.Context, eventsDir string, knownProducts map[string]struct{}, summary *Summary) error {
	paths, err := service.events.EventFiles(eventsDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no event files found in %s", eventsDir)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Files are processed concurrently, while database concurrency remains
	// bounded by the PostgreSQL pool configured in infrastructure/database.
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
			if err := service.processEventFile(ctx, path, knownProducts, summary, &mu); err != nil {
				mu.Lock()
				if firstErr == nil {
					// Operational errors cancel the whole ingest; invalid event
					// lines are recorded and skipped inside processEventFile.
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

func (service *Service) processEventFile(ctx context.Context, path string, knownProducts map[string]struct{}, summary *Summary, mu *sync.Mutex) error {
	err := service.events.ReadEvents(ctx, path, func(line EventLine) error {
		if line.ParseError != "" {
			if err := service.repo.RecordIngestError(ctx, line.SourceFile, line.LineNumber, line.RawLine, line.ParseError); err != nil {
				return err
			}
			addInvalidLine(summary, mu)
			return nil
		}

		movement, reason := inventory.ValidateMovement(line.Input, knownProducts)
		if reason != "" {
			if err := service.repo.RecordIngestError(ctx, line.SourceFile, line.LineNumber, line.RawLine, reason); err != nil {
				return err
			}
			addInvalidLine(summary, mu)
			return nil
		}

		inserted, err := service.repo.StoreMovement(ctx, movement)
		if err != nil {
			return err
		}
		addProcessedEvent(summary, mu, inserted)
		return nil
	})
	if err != nil {
		return err
	}

	mu.Lock()
	summary.FilesProcessed++
	mu.Unlock()

	return nil
}

func addInvalidLine(summary *Summary, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	summary.InvalidLines++
}

func addProcessedEvent(summary *Summary, mu *sync.Mutex, inserted bool) {
	mu.Lock()
	defer mu.Unlock()

	if inserted {
		summary.EventsInserted++
		return
	}
	summary.Duplicates++
}
