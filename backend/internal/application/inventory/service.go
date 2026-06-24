package inventory

import (
	"context"
	"errors"

	domain "takehome/internal/domain/inventory"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (service *Service) ListProductStock(ctx context.Context) ([]domain.ProductStock, error) {
	if service.repo == nil {
		return nil, errors.New("inventory repository is required")
	}

	return service.repo.ListProductStock(ctx)
}

func (service *Service) ListProductMovements(ctx context.Context, sku string) ([]domain.MovementHistoryItem, bool, error) {
	if service.repo == nil {
		return nil, false, errors.New("inventory repository is required")
	}

	exists, err := service.repo.ProductExists(ctx, sku)
	if err != nil || !exists {
		return nil, exists, err
	}

	movements, err := service.repo.ListProductMovements(ctx, sku)
	return movements, true, err
}
