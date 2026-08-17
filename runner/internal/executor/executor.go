package executor

import (
	"context"
	"errors"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/audit"
	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	"github.com/hansunwoo232/ProofLayer/runner/internal/job"
	"github.com/hansunwoo232/ProofLayer/runner/internal/policy"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

type Status string

const (
	StatusPassed Status = "passed"
	StatusFailed Status = "failed"
)

const ResultSchemaVersion = "1.0"

const (
	ErrorInvalidIdentity     = "invalid_identity"
	ErrorInvalidRequest      = "invalid_request"
	ErrorAuditWriteFailed    = "audit_write_failed"
	ErrorUnsupportedPlatform = "unsupported_platform"
	ErrorExecutionTimeout    = "execution_timeout"
	ErrorExecutionFailed     = "execution_failed"
	ErrorCleanupFailed       = "cleanup_failed"
	ErrorArtifactRemaining   = "artifact_remaining"
)

type Result struct {
	SchemaVersion   string    `json:"schema_version"`
	Status          Status    `json:"status"`
	CorrelationID   string    `json:"correlation_id"`
	ScenarioID      string    `json:"scenario_id"`
	ScenarioVersion string    `json:"scenario_version"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
	LatencyMS       int64     `json:"latency_ms"`
	CleanupStatus   Status    `json:"cleanup_status"`
	ErrorCode       string    `json:"error_code,omitempty"`
}

type Executor struct {
	identity identity.RunnerIdentity
	catalog  scenario.Catalog
	recorder audit.Recorder
	limits   policy.ExecutionLimits
	handlers map[string]ScenarioHandler
	now      func() time.Time
}

func New(
	runnerIdentity identity.RunnerIdentity,
	catalog scenario.Catalog,
	recorder audit.Recorder,
) *Executor {
	return &Executor{
		identity: runnerIdentity,
		catalog:  catalog,
		recorder: recorder,
		limits:   policy.ApprovedProcessMarkerLimits(),
		handlers: builtInHandlers(),
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (executor *Executor) Execute(parent context.Context, request job.ExecutionRequest) Result {
	startedAt := executor.now()
	result := Result{
		SchemaVersion:   ResultSchemaVersion,
		Status:          StatusFailed,
		CorrelationID:   request.CorrelationID,
		ScenarioID:      request.ScenarioID,
		ScenarioVersion: request.ScenarioVersion,
		StartedAt:       startedAt,
		CleanupStatus:   StatusPassed,
	}

	if err := executor.identity.Validate(startedAt); err != nil {
		return executor.finish(result, ErrorInvalidIdentity)
	}
	definition, err := request.Validate(executor.identity, executor.catalog)
	if err != nil {
		return executor.finish(result, ErrorInvalidRequest)
	}
	handler, ok := executor.handlers[definition.Handler]
	if !ok {
		return executor.finish(result, ErrorInvalidRequest)
	}

	if err := executor.record(request, "job.validation", "passed", ""); err != nil {
		return executor.finish(result, ErrorAuditWriteFailed)
	}

	executionContext, cancel, err := executor.limits.ExecutionContext(parent)
	if err != nil {
		return executor.finish(result, ErrorInvalidRequest)
	}
	defer cancel()

	result.CleanupStatus = StatusFailed
	executionErr := handler.Execute(executionContext, request.CorrelationID)
	executionCode := executor.executionErrorCode(executionContext, executionErr)
	executionOutcome := "passed"
	if executionCode != "" {
		executionOutcome = "failed"
	}
	if err := executor.record(request, "scenario.execution", executionOutcome, executionCode); err != nil {
		executionCode = ErrorAuditWriteFailed
	}

	cleanupCode := executor.cleanup(parent, handler, request)
	if cleanupCode != "" {
		return executor.finish(result, cleanupCode)
	}
	result.CleanupStatus = StatusPassed
	if executionCode != "" {
		return executor.finish(result, executionCode)
	}
	result.Status = StatusPassed
	return executor.finish(result, "")
}

func (executor *Executor) executionErrorCode(executionContext context.Context, err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(executionContext.Err(), context.DeadlineExceeded) {
		return ErrorExecutionTimeout
	}
	if errors.Is(err, ErrUnsupportedPlatform) {
		return ErrorUnsupportedPlatform
	}
	return ErrorExecutionFailed
}

func (executor *Executor) cleanup(parent context.Context, handler ScenarioHandler, request job.ExecutionRequest) string {
	cleanupContext, cancel, err := executor.limits.CleanupContext(parent)
	if err != nil {
		return executor.recordCleanupFailure(request, ErrorCleanupFailed)
	}
	defer cancel()

	if err := handler.Cleanup(cleanupContext, request.CorrelationID); err != nil {
		return executor.recordCleanupFailure(request, ErrorCleanupFailed)
	}
	if err := handler.VerifyAbsent(cleanupContext, request.CorrelationID); err != nil {
		errorCode := ErrorCleanupFailed
		if errors.Is(err, ErrArtifactPresent) {
			errorCode = ErrorArtifactRemaining
		}
		return executor.recordCleanupFailure(request, errorCode)
	}
	if err := executor.record(request, "scenario.cleanup", "passed", ""); err != nil {
		return ErrorAuditWriteFailed
	}
	return ""
}

func (executor *Executor) recordCleanupFailure(request job.ExecutionRequest, errorCode string) string {
	if err := executor.record(request, "scenario.cleanup", "failed", errorCode); err != nil {
		return ErrorAuditWriteFailed
	}
	return errorCode
}

func (executor *Executor) finish(result Result, errorCode string) Result {
	result.ErrorCode = errorCode
	result.CompletedAt = executor.now()
	result.LatencyMS = result.CompletedAt.Sub(result.StartedAt).Milliseconds()
	return result
}

func (executor *Executor) record(request job.ExecutionRequest, eventType, outcome, errorCode string) error {
	return executor.recorder.Append(audit.Event{
		SchemaVersion: "1.0",
		Timestamp:     executor.now(),
		EventType:     eventType,
		Outcome:       outcome,
		RunnerID:      executor.identity.RunnerID,
		CorrelationID: request.CorrelationID,
		ErrorCode:     errorCode,
	})
}
