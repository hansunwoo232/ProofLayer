package job

import (
	"errors"
	"testing"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

const (
	testEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	testHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
)

func activeIdentity() identity.RunnerIdentity {
	return identity.RunnerIdentity{
		SchemaVersion: identity.SchemaVersion,
		RunnerID:      "550e8400-e29b-41d4-a716-446655440000",
		EnvironmentID: testEnvironmentID,
		HostID:        testHostID,
		IdentityKeyID: "runner-key-01",
		RegisteredAt:  time.Now().Add(-time.Minute),
		State:         identity.StateActive,
	}
}

func validRequest() ExecutionRequest {
	return ExecutionRequest{
		CorrelationID:   "PL-0123456789ABCDEF0123456789ABCDEF",
		EnvironmentID:   testEnvironmentID,
		HostID:          testHostID,
		ScenarioID:      "windows-process-marker",
		ScenarioVersion: "0.1.0",
		Parameters:      map[string]any{},
	}
}

func TestExecutionRequestAcceptsApprovedInput(t *testing.T) {
	definition, err := validRequest().Validate(activeIdentity(), scenario.BuiltInCatalog())
	if err != nil {
		t.Fatalf("approved request rejected: %v", err)
	}
	if definition.Handler != "builtin.emit_process_marker" {
		t.Fatalf("handler = %q", definition.Handler)
	}
}

func TestExecutionRequestRejectsUnsafeOrMismatchedInput(t *testing.T) {
	tests := []ExecutionRequest{
		func() ExecutionRequest { value := validRequest(); value.CorrelationID = "bad"; return value }(),
		func() ExecutionRequest {
			value := validRequest()
			value.HostID = "550e8400-e29b-41d4-a716-446655440000"
			return value
		}(),
		func() ExecutionRequest { value := validRequest(); value.ScenarioID = "arbitrary-command"; return value }(),
		func() ExecutionRequest { value := validRequest(); value.ScenarioVersion = "0.2.0"; return value }(),
		func() ExecutionRequest { value := validRequest(); value.Parameters["command"] = "whoami"; return value }(),
	}

	for index, request := range tests {
		_, err := request.Validate(activeIdentity(), scenario.BuiltInCatalog())
		if !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}
