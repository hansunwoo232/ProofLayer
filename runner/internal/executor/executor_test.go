package executor

import (
	"context"
	"encoding/json"
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

func testExecutor(recorder audit.Recorder, handler ScenarioHandler) *Executor {
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
	executor.handlers["builtin.emit_process_marker"] = handler
	return executor
}

func testHandler(execute func(context.Context, string) error) ScenarioHandler {
	return handlerFunctions{
		execute:      execute,
		cleanup:      noCleanup,
		verifyAbsent: noCleanup,
	}
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
	executor := testExecutor(recorder, testHandler(func(_ context.Context, correlationID string) error {
		if correlationID != testRequest().CorrelationID {
			t.Fatalf("correlation ID = %q", correlationID)
		}
		return nil
	}))
	result := executor.Execute(context.Background(), testRequest())
	if result.Status != StatusPassed || result.ErrorCode != "" {
		t.Fatalf("result = %+v", result)
	}
	if result.SchemaVersion != ResultSchemaVersion {
		t.Fatalf("schema version = %q, want %q", result.SchemaVersion, ResultSchemaVersion)
	}
	if len(recorder.events) != 3 {
		t.Fatalf("audit events = %d, want 3", len(recorder.events))
	}
	if recorder.events[2].EventType != "scenario.cleanup" || recorder.events[2].Outcome != "passed" {
		t.Fatalf("cleanup audit event = %+v", recorder.events[2])
	}
}

func TestResultJSONIncludesSchemaVersion(t *testing.T) {
	result := testExecutor(&memoryRecorder{}, testHandler(func(context.Context, string) error {
		return nil
	})).Execute(context.Background(), testRequest())
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	if document["schema_version"] != ResultSchemaVersion {
		t.Fatalf("encoded schema version = %#v", document["schema_version"])
	}
	if _, ok := document["error_code"]; ok {
		t.Fatal("successful result included an error code")
	}
}

func TestExecuteRejectsInvalidRequestBeforeHandler(t *testing.T) {
	recorder := &memoryRecorder{}
	called := false
	executor := testExecutor(recorder, testHandler(func(context.Context, string) error {
		called = true
		return nil
	}))
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
		handler  ScenarioHandler
		expected string
	}{
		{name: "unsupported", handler: testHandler(func(context.Context, string) error { return ErrUnsupportedPlatform }), expected: ErrorUnsupportedPlatform},
		{name: "failed", handler: testHandler(func(context.Context, string) error { return errors.New("start failed") }), expected: ErrorExecutionFailed},
		{name: "timeout", handler: testHandler(func(ctx context.Context, _ string) error { <-ctx.Done(); return ctx.Err() }), expected: ErrorExecutionTimeout},
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
	executor := testExecutor(&memoryRecorder{err: errors.New("disk unavailable")}, testHandler(func(context.Context, string) error {
		called = true
		return nil
	}))
	result := executor.Execute(context.Background(), testRequest())
	if result.ErrorCode != ErrorAuditWriteFailed {
		t.Fatalf("error code = %q", result.ErrorCode)
	}
	if called {
		t.Fatal("handler called without accepted audit event")
	}
}

func TestExecuteRunsCleanupAfterExecutionFailure(t *testing.T) {
	cleanupCalled := false
	verificationCalled := false
	handler := handlerFunctions{
		execute: func(context.Context, string) error { return errors.New("execution failed") },
		cleanup: func(context.Context, string) error {
			cleanupCalled = true
			return nil
		},
		verifyAbsent: func(context.Context, string) error {
			verificationCalled = true
			return nil
		},
	}
	result := testExecutor(&memoryRecorder{}, handler).Execute(context.Background(), testRequest())
	if result.ErrorCode != ErrorExecutionFailed || result.CleanupStatus != StatusPassed {
		t.Fatalf("result = %+v", result)
	}
	if !cleanupCalled || !verificationCalled {
		t.Fatal("cleanup and absence verification must run after execution failure")
	}
}

func TestExecuteFailsWhenCleanupIsIncomplete(t *testing.T) {
	tests := []struct {
		name      string
		cleanup   func(context.Context, string) error
		verify    func(context.Context, string) error
		errorCode string
	}{
		{
			name:      "cleanup operation failed",
			cleanup:   func(context.Context, string) error { return errors.New("delete failed") },
			verify:    noCleanup,
			errorCode: ErrorCleanupFailed,
		},
		{
			name:      "artifact remains",
			cleanup:   noCleanup,
			verify:    func(context.Context, string) error { return ErrArtifactPresent },
			errorCode: ErrorArtifactRemaining,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &memoryRecorder{}
			handler := handlerFunctions{
				execute:      func(context.Context, string) error { return nil },
				cleanup:      test.cleanup,
				verifyAbsent: test.verify,
			}
			result := testExecutor(recorder, handler).Execute(context.Background(), testRequest())
			if result.Status != StatusFailed || result.CleanupStatus != StatusFailed || result.ErrorCode != test.errorCode {
				t.Fatalf("result = %+v", result)
			}
			lastEvent := recorder.events[len(recorder.events)-1]
			if lastEvent.EventType != "scenario.cleanup" || lastEvent.Outcome != "failed" || lastEvent.ErrorCode != test.errorCode {
				t.Fatalf("cleanup audit event = %+v", lastEvent)
			}
		})
	}
}

func TestCleanupIgnoresParentCancellationButKeepsDeadline(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	cleanupContextObserved := false
	handler := handlerFunctions{
		execute: func(ctx context.Context, _ string) error { return ctx.Err() },
		cleanup: func(ctx context.Context, _ string) error {
			if ctx.Err() != nil {
				t.Fatalf("cleanup inherited cancellation: %v", ctx.Err())
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("cleanup deadline missing")
			}
			cleanupContextObserved = true
			return nil
		},
		verifyAbsent: noCleanup,
	}
	result := testExecutor(&memoryRecorder{}, handler).Execute(parent, testRequest())
	if result.ErrorCode != ErrorExecutionFailed || !cleanupContextObserved {
		t.Fatalf("result = %+v, cleanup observed = %t", result, cleanupContextObserved)
	}
}
