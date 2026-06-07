package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestNormalizeEquipmentSlot(t *testing.T) {
	tests := []struct {
		name string
		slot string
		want string
		err  error
	}{
		{name: "gear", slot: " gear ", want: EquipmentSlotGear},
		{name: "accessory", slot: EquipmentSlotAccessory, want: EquipmentSlotAccessory},
		{name: "tool", slot: EquipmentSlotTool, want: EquipmentSlotTool},
		{name: "invalid", slot: "hat", err: ErrInvalidEquipmentSlot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeEquipmentSlot(tt.slot)
			if !errors.Is(err, tt.err) {
				t.Fatalf("error = %v, want %v", err, tt.err)
			}
			if got != tt.want {
				t.Fatalf("slot = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStringValue(t *testing.T) {
	if got := StringValue(nil); got != "" {
		t.Fatalf("StringValue(nil) = %q, want empty string", got)
	}

	value := "pickaxe"
	if got := StringValue(&value); got != value {
		t.Fatalf("StringValue(&value) = %q, want %q", got, value)
	}
}

func TestInventoryMutationValidationDoesNotExecuteQueries(t *testing.T) {
	querier := &countingQuerier{}

	if err := IncrementStudentItem(context.Background(), querier, 0, "rock", 1); err == nil {
		t.Fatal("expected invalid increment error")
	}
	if _, err := ConsumeStudentItem(context.Background(), querier, 1, " ", 1); err == nil {
		t.Fatal("expected invalid consume error")
	}
	if querier.execCount != 0 {
		t.Fatalf("exec count = %d, want 0", querier.execCount)
	}
}

type countingQuerier struct {
	execCount int
}

func (querier *countingQuerier) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	querier.execCount++
	return pgconn.CommandTag{}, nil
}
