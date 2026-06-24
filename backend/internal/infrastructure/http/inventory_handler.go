package http

import (
	"strings"

	application "takehome/internal/application/inventory"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type InventoryHandler struct {
	service *application.Service
}

func NewInventoryHandler(service *application.Service) *InventoryHandler {
	return &InventoryHandler{service: service}
}

func (handler *InventoryHandler) Routes() *fiber.App {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173,http://127.0.0.1:5173",
		AllowMethods: "GET,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept",
	}))
	app.Get("/healthz", handler.health)
	app.Get("/products/stock", handler.listProductStock)
	app.Get("/products/:sku/movements", handler.listProductMovements)

	return app
}

func (handler *InventoryHandler) health(c *fiber.Ctx) error {
	return c.JSON(map[string]string{"status": "ok"})
}

func (handler *InventoryHandler) listProductStock(c *fiber.Ctx) error {
	rows, err := handler.service.ListProductStock(c.Context())
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "query product stock")
	}

	return c.JSON(rows)
}

func (handler *InventoryHandler) listProductMovements(c *fiber.Ctx) error {
	sku := strings.TrimSpace(c.Params("sku"))
	if sku == "" {
		return writeError(c, fiber.StatusBadRequest, "sku is required")
	}

	rows, exists, err := handler.service.ListProductMovements(c.Context(), sku)
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, "query product movements")
	}
	if !exists {
		return writeError(c, fiber.StatusNotFound, "product not found")
	}

	return c.JSON(rows)
}

func writeError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(map[string]string{"error": message})
}
