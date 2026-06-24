package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	application "takehome/internal/application/inventory"
	domain "takehome/internal/domain/inventory"
	transport "takehome/internal/infrastructure/http"
)

func TestInventoryHandlerHealth(t *testing.T) {
	app := transport.NewInventoryHandler(application.NewService(&fakeRepository{})).Routes()

	response := performRequest(t, app, http.MethodGet, "/healthz")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
}

func TestInventoryHandlerListProductStock(t *testing.T) {
	expected := []domain.ProductStock{{SKU: "SKU-0001", Name: "Small box", Quantity: 12}}
	app := transport.NewInventoryHandler(application.NewService(&fakeRepository{stock: expected})).Routes()

	response := performRequest(t, app, http.MethodGet, "/products/stock")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var got []domain.ProductStock
	decodeJSON(t, response, &got)
	if len(got) != 1 || got[0] != expected[0] {
		t.Fatalf("expected stock %#v, got %#v", expected, got)
	}
}

func TestInventoryHandlerListProductStockReturnsServerError(t *testing.T) {
	app := transport.NewInventoryHandler(application.NewService(&fakeRepository{
		stockErr: errors.New("stock query failed"),
	})).Routes()

	response := performRequest(t, app, http.MethodGet, "/products/stock")

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.StatusCode)
	}
}

func TestInventoryHandlerListProductMovements(t *testing.T) {
	expected := []domain.MovementHistoryItem{
		{
			EventID:      "evt-0001",
			SKU:          "SKU-0001",
			MovementType: string(domain.MovementTypeIn),
			Quantity:     10,
			OccurredAt:   time.Date(2026, 6, 1, 2, 12, 46, 0, time.UTC),
		},
	}
	app := transport.NewInventoryHandler(application.NewService(&fakeRepository{
		productExists: true,
		movements:     expected,
	})).Routes()

	response := performRequest(t, app, http.MethodGet, "/products/SKU-0001/movements")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var got []domain.MovementHistoryItem
	decodeJSON(t, response, &got)
	if len(got) != 1 || got[0].EventID != expected[0].EventID {
		t.Fatalf("expected movements %#v, got %#v", expected, got)
	}
}

func TestInventoryHandlerListProductMovementsReturnsNotFound(t *testing.T) {
	app := transport.NewInventoryHandler(application.NewService(&fakeRepository{
		productExists: false,
	})).Routes()

	response := performRequest(t, app, http.MethodGet, "/products/SKU-404/movements")

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.StatusCode)
	}
}

func TestInventoryHandlerListProductMovementsReturnsServerError(t *testing.T) {
	app := transport.NewInventoryHandler(application.NewService(&fakeRepository{
		productExists: true,
		movementsErr:  errors.New("movement query failed"),
	})).Routes()

	response := performRequest(t, app, http.MethodGet, "/products/SKU-0001/movements")

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.StatusCode)
	}
}

func performRequest(t *testing.T, app interface {
	Test(*http.Request, ...int) (*http.Response, error)
}, method, path string) *http.Response {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}

	return response
}

func decodeJSON(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

type fakeRepository struct {
	stock         []domain.ProductStock
	stockErr      error
	productExists bool
	existsErr     error
	movements     []domain.MovementHistoryItem
	movementsErr  error
}

func (repo *fakeRepository) ListProductStock(context.Context) ([]domain.ProductStock, error) {
	return repo.stock, repo.stockErr
}

func (repo *fakeRepository) ProductExists(context.Context, string) (bool, error) {
	return repo.productExists, repo.existsErr
}

func (repo *fakeRepository) ListProductMovements(context.Context, string) ([]domain.MovementHistoryItem, error) {
	return repo.movements, repo.movementsErr
}
