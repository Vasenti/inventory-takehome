package files

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
	"sort"
	"strings"

	"takehome/internal/application/ingest"
	"takehome/internal/domain/inventory"
)

type ProductCSVReader struct{}

func (ProductCSVReader) ReadProducts(ctx context.Context, path string) ([]inventory.Product, error) {
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

	var products []inventory.Product
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

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

		product := inventory.Product{
			SKU:  strings.TrimSpace(record[0]),
			Name: strings.TrimSpace(record[1]),
		}
		if product.SKU == "" || product.Name == "" {
			return nil, fmt.Errorf("invalid product record: sku and name are required")
		}

		products = append(products, product)
	}

	return products, nil
}

type NDJSONEventReader struct{}

func (NDJSONEventReader) EventFiles(eventsDir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(eventsDir, "*.ndjson"))
	if err != nil {
		return nil, fmt.Errorf("list event files: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func (NDJSONEventReader) ReadEvents(ctx context.Context, path string, handle func(ingest.EventLine) error) error {
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
		line := ingest.EventLine{
			SourceFile: path,
			LineNumber: lineNumber,
			RawLine:    rawLine,
		}

		if err := json.Unmarshal([]byte(rawLine), &line.Input); err != nil {
			line.ParseError = "malformed json"
		}

		if err := handle(line); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan event file %s: %w", path, err)
	}

	return nil
}
