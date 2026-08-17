package executor

import (
	"context"
	"errors"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
)

var ErrUnsupportedPlatform = errors.New("process marker is unsupported on this platform")

func RunProcessMarker(ctx context.Context, correlationID string) error {
	if !correlation.Valid(correlationID) {
		return errors.New("invalid correlation ID")
	}
	return runProcessMarker(ctx, correlationID)
}

func processMarkerHandler() ScenarioHandler {
	return handlerFunctions{
		execute:      RunProcessMarker,
		cleanup:      noCleanup,
		verifyAbsent: noCleanup,
	}
}
