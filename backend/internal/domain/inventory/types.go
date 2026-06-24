package inventory

import (
	"strings"
	"time"
)

type MovementType string

const (
	MovementTypeIn  MovementType = "IN"
	MovementTypeOut MovementType = "OUT"
)

type Product struct {
	SKU  string
	Name string
}

type MovementInput struct {
	EventID    string `json:"event_id"`
	SKU        string `json:"sku"`
	Type       string `json:"type"`
	Quantity   int    `json:"quantity"`
	OccurredAt string `json:"occurred_at"`
}

type Movement struct {
	EventID    string
	SKU        string
	Type       MovementType
	Quantity   int
	OccurredAt time.Time
}

type ProductStock struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type MovementHistoryItem struct {
	EventID      string    `json:"event_id"`
	SKU          string    `json:"sku"`
	MovementType string    `json:"type"`
	Quantity     int       `json:"quantity"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func ValidateMovement(input MovementInput, knownProducts map[string]struct{}) (Movement, string) {
	if strings.TrimSpace(input.EventID) == "" {
		return Movement{}, "event_id is required"
	}
	if _, ok := knownProducts[input.SKU]; !ok {
		return Movement{}, "unknown sku"
	}
	if input.Type != string(MovementTypeIn) && input.Type != string(MovementTypeOut) {
		return Movement{}, "invalid movement type"
	}
	if input.Quantity <= 0 {
		return Movement{}, "quantity must be positive"
	}

	occurredAt, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil {
		return Movement{}, "invalid occurred_at"
	}

	return Movement{
		EventID:    input.EventID,
		SKU:        input.SKU,
		Type:       MovementType(input.Type),
		Quantity:   input.Quantity,
		OccurredAt: occurredAt,
	}, ""
}
