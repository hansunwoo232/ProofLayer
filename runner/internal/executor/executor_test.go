package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/audit"
	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	"github.com/hansunwoo232/ProofLayer/runner/internal/job"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

const (
	testRunnerID      = "550e8400-e29b-41d4-a716-446655440000"
	testEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	testHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
)

type memoryRecorder struct {
	events []audit.Event
	err    error
}

func (recorder *memoryRecorder) Append(event audit.Event) error {
	if recorder.err != nil {
		return recorder.err
	}
	recorder.events = append(recorder.events, event)
	return nil
}

func testExecutor(recorder audit.Recorder, handler MarkerHandler) *Executor {
	now := time.Now().UTC()
	executor := New(identity.RunnerIdentity{
		SchemaVersion: identity.SchemaVersion,
		RunnerID:      testRunnerID,
		EnvironmentID: testEnvironmentID,
		HostID:        testHostID,
		IdentityKeyID: "runner-key-01",
		RegisteredAt:  now.Add(-time.Minute),
		State:         identity.StateActive,
	}, scenario.BuiltInCatalog(), recorder)
	executor.handler = handler
	return executor
}

func testRequest() job.ExecutionRequest {
	return job.ExecutionRequest{
		CorrelationID:   "PL-0123456789ABCDEF0123456789ABCDEF",
		EnvironmentID:   testEnvironmentID,
		HostID:          testHostID,
		ScenarioID:      "windows-process-marker",
		ScenarioVersion: "0.1.0",
		Parameters:      map[string]any{},
	}
}

func TestExecuteApprovedHandler(t *testing.T) {
	recorder := &memoryRecorder{}
	executor := testExecutor(recorder, func(_ context.Context, correlationID string) error {
		if correlationID != testRequest().CorrelationID {
			t.Fatalf("correlation ID = %q", correlationID)
		}
		return nil
	})
	result := executor.Execute(context.Background(), testRequest())
	if result.Status != StatusPassed || result.ErrorCode != "" {
		t.Fatalf("result = %+v", result)
	}
	if len(recorder.events) != 2 {
		t.Fatalf("audit events = %d, want 2", len(recorder.events))
	}
}

func TestExecuteRejectsInvalidRequestBeforeHandler(t *testing.T) {
	recorder := &memoryRecorder{}
	called := false
	executor := testExecutor(recorder, func(context.Context, string) error {
		called = true
		return nil
	})
	request := testRequest()
	request.Parameters["command"] = "whoami"
	result := executor.Execute(context.Background(), request)
	if result.ErrorCode != ErrorInvalidRequest {
		t.Fatalf("error code = %q", result.ErrorCode)
	}
	if called {
		t.Fatal("handler called for invalid request")
	}
}

func TestExecuteReportsStableFailureCodes(t *testing.T) {
	tests := []struct {
		name     string
		handler  MarkerHandler
		expected string
	}{
		{name: "unsupported", handler: func(context.Context, string) error { return ErrUnsupportedPlatform }, expected: ErrorUnsupportedPlatform},
		{name: "failed", handler: func(context.Context, string) error { return errors.New("start failed") }, expected: ErrorExecutionFailed},
		{name: "timeout", handler: func(ctx context.Context, _ string) error { <-ctx.Done(); return ctx.Err() }, expected: ErrorExecutionTimeout},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := testExecutor(&memoryRecorder{}, test.handler)
			if test.name == "timeout" {
				executor.limits.Runtime = time.Millisecond
			}
			result := executor.Execute(context.Background(), testRequest())
			if result.ErrorCode != test.expected {
				t.Fatalf("error code = %q, want %q", result.ErrorCode, test.expected)
			}
		})
	}
}

func TestExecuteFailsClosedWhenAuditIsUnavailable(t *testing.T) {
	called := false
	executor := testExecutor(&memoryRecorder{err: errors.New("disk unavailable")}, func(context.Context, string) error {
		called = true
		return nil
	})
	result := executor.Execute(context.Background(), testRequest())
	if result.ErrorCode != ErrorAuditWriteFailed {
		t.Fatalf("error code = %q", result.ErrorCode)
	}
	if called {
		t.Fatal("handler called without accepted audit event")
	}
}
