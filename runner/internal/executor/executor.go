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

const (
	ErrorInvalidIdentity     = "invalid_identity"
	ErrorInvalidRequest      = "invalid_request"
	ErrorAuditWriteFailed    = "audit_write_failed"
	ErrorUnsupportedPlatform = "unsupported_platform"
	ErrorExecutionTimeout    = "execution_timeout"
	ErrorExecutionFailed     = "execution_failed"
)

type Result struct {
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

type MarkerHandler func(context.Context, string) error

type Executor struct {
	identity identity.RunnerIdentity
	catalog  scenario.Catalog
	recorder audit.Recorder
	limits   policy.ExecutionLimits
	handler  MarkerHandler
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
		handler:  RunProcessMarker,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (executor *Executor) Execute(parent context.Context, request job.ExecutionRequest) Result {
	startedAt := executor.now()
	result := Result{
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
	if definition.Handler != "builtin.emit_process_marker" {
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

	err = executor.handler(executionContext, request.CorrelationID)
	if err != nil {
		errorCode := ErrorExecutionFailed
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(executionContext.Err(), context.DeadlineExceeded) {
			errorCode = ErrorExecutionTimeout
		} else if errors.Is(err, ErrUnsupportedPlatform) {
			errorCode = ErrorUnsupportedPlatform
		}
		if auditErr := executor.record(request, "scenario.execution", "failed", errorCode); auditErr != nil {
			errorCode = ErrorAuditWriteFailed
		}
		return executor.finish(result, errorCode)
	}

	if err := executor.record(request, "scenario.execution", "passed", ""); err != nil {
		return executor.finish(result, ErrorAuditWriteFailed)
	}
	result.Status = StatusPassed
	return executor.finish(result, "")
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
