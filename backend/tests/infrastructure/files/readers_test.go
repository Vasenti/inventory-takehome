package files_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"takehome/internal/application/ingest"
	"takehome/internal/infrastructure/files"
)

func TestProductCSVReaderReadsProducts(t *testing.T) {
	path := writeFile(t, "products.csv", "sku,name\nSKU-0001,Small box\nSKU-0002,Large box\n")
	reader := files.ProductCSVReader{}

	products, err := reader.ReadProducts(context.Background(), path)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	if products[0].SKU != "SKU-0001" || products[0].Name != "Small box" {
		t.Fatalf("unexpected first product: %#v", products[0])
	}
}

func TestProductCSVReaderRejectsInvalidHeader(t *testing.T) {
	path := writeFile(t, "products.csv", "id,title\nSKU-0001,Small box\n")
	reader := files.ProductCSVReader{}

	_, err := reader.ReadProducts(context.Background(), path)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProductCSVReaderRejectsEmptyProductFields(t *testing.T) {
	path := writeFile(t, "products.csv", "sku,name\nSKU-0001,\n")
	reader := files.ProductCSVReader{}

	_, err := reader.ReadProducts(context.Background(), path)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNDJSONEventReaderListsEventFilesInOrder(t *testing.T) {
	dir := t.TempDir()
	writeFileInDir(t, dir, "part-002.ndjson", "")
	writeFileInDir(t, dir, "part-001.ndjson", "")
	writeFileInDir(t, dir, "notes.txt", "")
	reader := files.NDJSONEventReader{}

	paths, err := reader.EventFiles(dir)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 ndjson files, got %d", len(paths))
	}
	if filepath.Base(paths[0]) != "part-001.ndjson" || filepath.Base(paths[1]) != "part-002.ndjson" {
		t.Fatalf("expected sorted ndjson files, got %#v", paths)
	}
}

func TestNDJSONEventReaderReadsEventsAndPreservesLineMetadata(t *testing.T) {
	path := writeFile(t, "part-000.ndjson", ""+
		"{\"event_id\":\"evt-0001\",\"sku\":\"SKU-0001\",\"type\":\"IN\",\"quantity\":10,\"occurred_at\":\"2026-06-01T02:12:46Z\"}\n"+
		"{\"event_id\":\"evt-bad\",\"sku\":\"SKU-0001\",\"type\":\"IN\",\"quantity\":\n")
	reader := files.NDJSONEventReader{}

	var lines []ingest.EventLine
	err := reader.ReadEvents(context.Background(), path, func(line ingest.EventLine) error {
		lines = append(lines, line)
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].SourceFile != path || lines[0].LineNumber != 1 {
		t.Fatalf("unexpected first line metadata: %#v", lines[0])
	}
	if lines[0].Input.EventID != "evt-0001" || lines[0].ParseError != "" {
		t.Fatalf("unexpected parsed event: %#v", lines[0])
	}
	if lines[1].LineNumber != 2 || lines[1].RawLine == "" {
		t.Fatalf("unexpected malformed line metadata: %#v", lines[1])
	}
	if lines[1].ParseError != "malformed json" {
		t.Fatalf("expected malformed json parse error, got %q", lines[1].ParseError)
	}
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	dir := t.TempDir()
	return writeFileInDir(t, dir, name, content)
}

func writeFileInDir(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}
