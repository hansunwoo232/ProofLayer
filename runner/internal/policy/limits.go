package policy

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidLimits = errors.New("invalid execution limits")

type ExecutionLimits struct {
	Runtime            time.Duration `json:"runtime"`
	CleanupRuntime     time.Duration `json:"cleanup_runtime"`
	MaximumOutputBytes int           `json:"maximum_output_bytes"`
	MaximumChildren    int           `json:"maximum_children"`
	NetworkAccess      string        `json:"network_access"`
}

func ApprovedProcessMarkerLimits() ExecutionLimits {
	return ExecutionLimits{
		Runtime:            30 * time.Second,
		CleanupRuntime:     10 * time.Second,
		MaximumOutputBytes: 4096,
		MaximumChildren:    1,
		NetworkAccess:      "none",
	}
}

func (limits ExecutionLimits) Validate() error {
	if limits.Runtime <= 0 || limits.Runtime > 30*time.Second {
		return invalid("runtime")
	}
	if limits.CleanupRuntime <= 0 || limits.CleanupRuntime > 10*time.Second {
		return invalid("cleanup_runtime")
	}
	if limits.MaximumOutputBytes < 0 || limits.MaximumOutputBytes > 4096 {
		return invalid("maximum_output_bytes")
	}
	if limits.MaximumChildren != 1 {
		return invalid("maximum_children")
	}
	if limits.NetworkAccess != "none" {
		return invalid("network_access")
	}
	return nil
}

func (limits ExecutionLimits) ExecutionContext(parent context.Context) (context.Context, context.CancelFunc, error) {
	if err := limits.Validate(); err != nil {
		return nil, nil, err
	}
	contextWithDeadline, cancel := context.WithTimeout(parent, limits.Runtime)
	return contextWithDeadline, cancel, nil
}

func invalid(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidLimits, field)
}
