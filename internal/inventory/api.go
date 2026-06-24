package inventory

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type API struct {
	database *gorm.DB
}

type ProductStockResponse struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type MovementResponse struct {
	EventID      string    `json:"event_id"`
	SKU          string    `json:"sku"`
	MovementType string    `json:"type"`
	Quantity     int       `json:"quantity"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func NewAPI(database *gorm.DB) *API {
	return &API{database: database}
}

func (api *API) Routes() *fiber.App {
	app := fiber.New()
	app.Get("/healthz", api.health)
	app.Get("/products/stock", api.listProductStock)
	app.Get("/products/:sku/movements", api.listProductMovements)

	return app
}

func (api *API) health(c *fiber.Ctx) error {
	return c.JSON(map[string]string{"status": "ok"})
}

func (api *API) listProductStock(c *fiber.Ctx) error {
	var rows []ProductStockResponse
	if err := api.database.WithContext(c.Context()).Raw(`
		SELECT p.sku, p.name, COALESCE(ps.quantity, 0) AS quantity
		FROM products p
		LEFT JOIN product_stock ps ON ps.sku = p.sku
		ORDER BY p.sku
	`).Scan(&rows).Error; err != nil {
		return writeError(c, fiber.StatusInternalServerError, "query product stock")
	}

	return c.JSON(rows)
}

func (api *API) listProductMovements(c *fiber.Ctx) error {
	sku := strings.TrimSpace(c.Params("sku"))
	if sku == "" {
		return writeError(c, fiber.StatusBadRequest, "sku is required")
	}

	var exists bool
	if err := api.database.WithContext(c.Context()).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM products
			WHERE sku = ?
		)
	`, sku).Scan(&exists).Error; err != nil {
		return writeError(c, fiber.StatusInternalServerError, "query product")
	}
	if !exists {
		return writeError(c, fiber.StatusNotFound, "product not found")
	}

	var rows []MovementResponse
	if err := api.database.WithContext(c.Context()).Raw(`
		SELECT event_id, sku, movement_type, quantity, occurred_at
		FROM inventory_movements
		WHERE sku = ?
		ORDER BY occurred_at DESC, event_id
	`, sku).Scan(&rows).Error; err != nil {
		return writeError(c, fiber.StatusInternalServerError, "query product movements")
	}

	return c.JSON(rows)
}

func writeError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(map[string]string{"error": message})
}
