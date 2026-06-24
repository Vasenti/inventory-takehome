package inventory

import (
	"context"

	domain "takehome/internal/domain/inventory"
)

type Repository interface {
	ListProductStock(ctx context.Context) ([]domain.ProductStock, error)
	ProductExists(ctx context.Context, sku string) (bool, error)
	ListProductMovements(ctx context.Context, sku string) ([]domain.MovementHistoryItem, error)
}
