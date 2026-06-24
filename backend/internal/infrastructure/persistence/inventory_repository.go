package persistence

import (
	"context"

	domain "takehome/internal/domain/inventory"

	"gorm.io/gorm"
)

type InventoryRepository struct {
	database *gorm.DB
}

func NewInventoryRepository(database *gorm.DB) *InventoryRepository {
	return &InventoryRepository{database: database}
}

func (repo *InventoryRepository) UpsertProduct(ctx context.Context, product domain.Product) error {
	return repo.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO products (sku, name)
			VALUES (?, ?)
			ON CONFLICT (sku) DO UPDATE SET name = EXCLUDED.name
		`, product.SKU, product.Name).Error; err != nil {
			return err
		}

		return tx.Exec(`
			INSERT INTO product_stock (sku)
			VALUES (?)
			ON CONFLICT (sku) DO NOTHING
		`, product.SKU).Error
	})
}

func (repo *InventoryRepository) StoreMovement(ctx context.Context, movement domain.Movement) (bool, error) {
	inserted := false
	err := repo.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
			INSERT INTO inventory_movements (event_id, sku, movement_type, quantity, occurred_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (event_id) DO NOTHING
		`, movement.EventID, movement.SKU, string(movement.Type), movement.Quantity, movement.OccurredAt)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		delta := movement.Quantity
		if movement.Type == domain.MovementTypeOut {
			delta = -delta
		}

		if err := tx.Exec(`
			UPDATE product_stock
			SET quantity = quantity + ?, updated_at = now()
			WHERE sku = ?
		`, delta, movement.SKU).Error; err != nil {
			return err
		}

		inserted = true
		return nil
	})

	return inserted, err
}

func (repo *InventoryRepository) RecordIngestError(ctx context.Context, sourceFile string, lineNumber int, rawLine, reason string) error {
	return repo.database.WithContext(ctx).Exec(`
		INSERT INTO ingest_errors (source_file, line_number, raw_line, reason)
		VALUES (?, ?, ?, ?)
	`, sourceFile, lineNumber, rawLine, reason).Error
}

func (repo *InventoryRepository) ListProductStock(ctx context.Context) ([]domain.ProductStock, error) {
	var rows []domain.ProductStock
	err := repo.database.WithContext(ctx).Raw(`
		SELECT p.sku, p.name, COALESCE(ps.quantity, 0) AS quantity
		FROM products p
		LEFT JOIN product_stock ps ON ps.sku = p.sku
		ORDER BY p.sku
	`).Scan(&rows).Error

	return rows, err
}

func (repo *InventoryRepository) ProductExists(ctx context.Context, sku string) (bool, error) {
	var exists bool
	err := repo.database.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM products
			WHERE sku = ?
		)
	`, sku).Scan(&exists).Error

	return exists, err
}

func (repo *InventoryRepository) ListProductMovements(ctx context.Context, sku string) ([]domain.MovementHistoryItem, error) {
	var rows []domain.MovementHistoryItem
	err := repo.database.WithContext(ctx).Raw(`
		SELECT event_id, sku, movement_type, quantity, occurred_at
		FROM inventory_movements
		WHERE sku = ?
		ORDER BY occurred_at DESC, event_id
	`, sku).Scan(&rows).Error

	return rows, err
}
