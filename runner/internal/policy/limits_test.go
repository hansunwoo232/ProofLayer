package policy

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestApprovedProcessMarkerLimits(t *testing.T) {
	limits := ApprovedProcessMarkerLimits()
	if err := limits.Validate(); err != nil {
		t.Fatalf("approved limits rejected: %v", err)
	}
	ctx, cancel, err := limits.ExecutionContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 30*time.Second {
		t.Fatal("execution deadline not enforced")
	}
}

func TestLimitsRejectRelaxation(t *testing.T) {
	tests := []ExecutionLimits{
		func() ExecutionLimits {
			value := ApprovedProcessMarkerLimits()
			value.Runtime = 31 * time.Second
			return value
		}(),
		func() ExecutionLimits {
			value := ApprovedProcessMarkerLimits()
			value.CleanupRuntime = 11 * time.Second
			return value
		}(),
		func() ExecutionLimits {
			value := ApprovedProcessMarkerLimits()
			value.MaximumOutputBytes = 4097
			return value
		}(),
		func() ExecutionLimits {
			value := ApprovedProcessMarkerLimits()
			value.MaximumChildren = 2
			return value
		}(),
		func() ExecutionLimits {
			value := ApprovedProcessMarkerLimits()
			value.NetworkAccess = "outbound"
			return value
		}(),
	}
	for index, limits := range tests {
		if err := limits.Validate(); !errors.Is(err, ErrInvalidLimits) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}
