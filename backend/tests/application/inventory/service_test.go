package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	application "takehome/internal/application/inventory"
	domain "takehome/internal/domain/inventory"
)

func TestServiceListProductStockReturnsRepositoryRows(t *testing.T) {
	expected := []domain.ProductStock{
		{SKU: "SKU-0001", Name: "Small box", Quantity: 12},
		{SKU: "SKU-0002", Name: "Large box", Quantity: -3},
	}
	repo := &fakeRepository{stock: expected}
	service := application.NewService(repo)

	got, err := service.ListProductStock(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), len(got))
	}
	if got[0] != expected[0] || got[1] != expected[1] {
		t.Fatalf("expected stock rows %#v, got %#v", expected, got)
	}
	if repo.stockCalls != 1 {
		t.Fatalf("expected stock repository to be called once, got %d", repo.stockCalls)
	}
}

func TestServiceListProductStockPropagatesRepositoryError(t *testing.T) {
	expectedErr := errors.New("stock query failed")
	service := application.NewService(&fakeRepository{stockErr: expectedErr})

	_, err := service.ListProductStock(context.Background())

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestServiceListProductStockRequiresRepository(t *testing.T) {
	service := application.NewService(nil)

	_, err := service.ListProductStock(context.Background())

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceListProductMovementsReturnsHistoryWhenProductExists(t *testing.T) {
	expected := []domain.MovementHistoryItem{
		{
			EventID:      "evt-0001",
			SKU:          "SKU-0001",
			MovementType: string(domain.MovementTypeIn),
			Quantity:     10,
			OccurredAt:   time.Date(2026, 6, 1, 2, 12, 46, 0, time.UTC),
		},
	}
	repo := &fakeRepository{
		productExists: true,
		movements:     expected,
	}
	service := application.NewService(repo)

	got, exists, err := service.ListProductMovements(context.Background(), "SKU-0001")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatal("expected product to exist")
	}
	if len(got) != len(expected) || got[0] != expected[0] {
		t.Fatalf("expected movements %#v, got %#v", expected, got)
	}
	if repo.existsCalls != 1 {
		t.Fatalf("expected product existence check once, got %d", repo.existsCalls)
	}
	if repo.movementsCalls != 1 {
		t.Fatalf("expected movement query once, got %d", repo.movementsCalls)
	}
}

func TestServiceListProductMovementsSkipsHistoryWhenProductDoesNotExist(t *testing.T) {
	repo := &fakeRepository{productExists: false}
	service := application.NewService(repo)

	got, exists, err := service.ListProductMovements(context.Background(), "SKU-404")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatal("expected product not to exist")
	}
	if got != nil {
		t.Fatalf("expected nil movements, got %#v", got)
	}
	if repo.movementsCalls != 0 {
		t.Fatalf("expected movement query to be skipped, got %d calls", repo.movementsCalls)
	}
}

func TestServiceListProductMovementsPropagatesExistenceError(t *testing.T) {
	expectedErr := errors.New("exists query failed")
	repo := &fakeRepository{existsErr: expectedErr}
	service := application.NewService(repo)

	_, exists, err := service.ListProductMovements(context.Background(), "SKU-0001")

	if exists {
		t.Fatal("expected exists=false when existence query fails")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if repo.movementsCalls != 0 {
		t.Fatalf("expected movement query to be skipped, got %d calls", repo.movementsCalls)
	}
}

func TestServiceListProductMovementsPropagatesHistoryError(t *testing.T) {
	expectedErr := errors.New("movement query failed")
	repo := &fakeRepository{
		productExists: true,
		movementsErr:  expectedErr,
	}
	service := application.NewService(repo)

	_, exists, err := service.ListProductMovements(context.Background(), "SKU-0001")

	if !exists {
		t.Fatal("expected product to exist")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestServiceListProductMovementsRequiresRepository(t *testing.T) {
	service := application.NewService(nil)

	_, exists, err := service.ListProductMovements(context.Background(), "SKU-0001")

	if exists {
		t.Fatal("expected exists=false")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeRepository struct {
	stock          []domain.ProductStock
	stockErr       error
	productExists  bool
	existsErr      error
	movements      []domain.MovementHistoryItem
	movementsErr   error
	stockCalls     int
	existsCalls    int
	movementsCalls int
}

func (repo *fakeRepository) ListProductStock(context.Context) ([]domain.ProductStock, error) {
	repo.stockCalls++
	return repo.stock, repo.stockErr
}

func (repo *fakeRepository) ProductExists(context.Context, string) (bool, error) {
	repo.existsCalls++
	return repo.productExists, repo.existsErr
}

func (repo *fakeRepository) ListProductMovements(context.Context, string) ([]domain.MovementHistoryItem, error) {
	repo.movementsCalls++
	return repo.movements, repo.movementsErr
}
