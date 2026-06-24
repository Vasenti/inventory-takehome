package inventory_test

import (
	"testing"
	"time"

	"takehome/internal/domain/inventory"
)

func TestValidateMovementAcceptsValidMovements(t *testing.T) {
	knownProducts := map[string]struct{}{"SKU-0001": {}}

	tests := []struct {
		name         string
		movementType string
	}{
		{name: "in movement", movementType: string(inventory.MovementTypeIn)},
		{name: "out movement", movementType: string(inventory.MovementTypeOut)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := inventory.ValidateMovement(inventory.MovementInput{
				EventID:    "evt-0001",
				SKU:        "SKU-0001",
				Type:       tt.movementType,
				Quantity:   10,
				OccurredAt: "2026-06-01T02:12:46Z",
			}, knownProducts)

			if reason != "" {
				t.Fatalf("expected valid movement, got reason %q", reason)
			}
			if got.Type != inventory.MovementType(tt.movementType) {
				t.Fatalf("expected type %q, got %q", tt.movementType, got.Type)
			}
			if got.Quantity != 10 {
				t.Fatalf("expected quantity 10, got %d", got.Quantity)
			}
			if got.OccurredAt.IsZero() {
				t.Fatal("expected occurred_at to be parsed")
			}
		})
	}
}

func TestValidateMovementRejectsInvalidInput(t *testing.T) {
	knownProducts := map[string]struct{}{"SKU-0001": {}}

	tests := []struct {
		name   string
		input  inventory.MovementInput
		reason string
	}{
		{
			name: "missing event id",
			input: inventory.MovementInput{
				SKU:        "SKU-0001",
				Type:       string(inventory.MovementTypeIn),
				Quantity:   10,
				OccurredAt: "2026-06-01T02:12:46Z",
			},
			reason: "event_id is required",
		},
		{
			name: "unknown sku",
			input: inventory.MovementInput{
				EventID:    "evt-0001",
				SKU:        "SKU-9999",
				Type:       string(inventory.MovementTypeIn),
				Quantity:   10,
				OccurredAt: "2026-06-01T02:12:46Z",
			},
			reason: "unknown sku",
		},
		{
			name: "unknown movement type",
			input: inventory.MovementInput{
				EventID:    "evt-0001",
				SKU:        "SKU-0001",
				Type:       "UNKNOWN",
				Quantity:   10,
				OccurredAt: "2026-06-01T02:12:46Z",
			},
			reason: "invalid movement type",
		},
		{
			name: "zero quantity",
			input: inventory.MovementInput{
				EventID:    "evt-0001",
				SKU:        "SKU-0001",
				Type:       string(inventory.MovementTypeIn),
				Quantity:   0,
				OccurredAt: "2026-06-01T02:12:46Z",
			},
			reason: "quantity must be positive",
		},
		{
			name: "negative quantity",
			input: inventory.MovementInput{
				EventID:    "evt-0001",
				SKU:        "SKU-0001",
				Type:       string(inventory.MovementTypeIn),
				Quantity:   -1,
				OccurredAt: "2026-06-01T02:12:46Z",
			},
			reason: "quantity must be positive",
		},
		{
			name: "invalid occurred at",
			input: inventory.MovementInput{
				EventID:    "evt-0001",
				SKU:        "SKU-0001",
				Type:       string(inventory.MovementTypeIn),
				Quantity:   10,
				OccurredAt: "not-a-time",
			},
			reason: "invalid occurred_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, reason := inventory.ValidateMovement(tt.input, knownProducts)
			if reason != tt.reason {
				t.Fatalf("expected reason %q, got %q", tt.reason, reason)
			}
		})
	}
}

func TestValidateMovementParsesRFC3339Timestamp(t *testing.T) {
	knownProducts := map[string]struct{}{"SKU-0001": {}}
	expected := time.Date(2026, 6, 1, 2, 12, 46, 0, time.UTC)

	got, reason := inventory.ValidateMovement(inventory.MovementInput{
		EventID:    "evt-0001",
		SKU:        "SKU-0001",
		Type:       string(inventory.MovementTypeIn),
		Quantity:   10,
		OccurredAt: "2026-06-01T02:12:46Z",
	}, knownProducts)

	if reason != "" {
		t.Fatalf("expected valid movement, got reason %q", reason)
	}
	if !got.OccurredAt.Equal(expected) {
		t.Fatalf("expected occurred_at %s, got %s", expected, got.OccurredAt)
	}
}
