package worker

import (
	"context"
	"errors"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/controlplane"
	"github.com/hansunwoo232/ProofLayer/runner/internal/executor"
	"github.com/hansunwoo232/ProofLayer/runner/internal/fieldvalidator"
	"github.com/hansunwoo232/ProofLayer/runner/internal/job"
	"github.com/hansunwoo232/ProofLayer/runner/internal/observer"
	"github.com/hansunwoo232/ProofLayer/runner/internal/splunk"
)

const ResultSchemaVersion = "1.0"

var ErrInvalidDependencies = errors.New("invalid worker dependencies")

type ControlPlane interface {
	Lease(context.Context) (controlplane.Job, bool, error)
	Acknowledge(context.Context, string, bool) error
	UpdateStage(context.Context, string, controlplane.StageUpdate) error
	Complete(context.Context, string) error
}

type ScenarioExecutor interface {
	Execute(context.Context, job.ExecutionRequest) executor.Result
}

type EndpointObserver interface {
	Observe(context.Context, string, time.Time) (observer.Evidence, error)
}

type HECExporter interface {
	Export(context.Context, string, observer.Evidence) error
}

type SIEMObserver interface {
	splunk.ExactSearcher
	SearchDetection(context.Context, string, splunk.SearchWindow, splunk.DetectionPlan) (splunk.DetectionEvidence, error)
}

type Result struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	JobID         string `json:"job_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
}

type Worker struct {
	controlPlane ControlPlane
	executor     ScenarioExecutor
	endpoint     EndpointObserver
	exporter     HECExporter
	siem         SIEMObserver
	now          func() time.Time
	polling      splunk.PollingPolicy
}

func New(
	controlPlane ControlPlane,
	scenarioExecutor ScenarioExecutor,
	endpoint EndpointObserver,
	exporter HECExporter,
	siem SIEMObserver,
) (*Worker, error) {
	if controlPlane == nil || scenarioExecutor == nil || endpoint == nil || exporter == nil || siem == nil {
		return nil, ErrInvalidDependencies
	}
	return &Worker{
		controlPlane: controlPlane,
		executor:     scenarioExecutor,
		endpoint:     endpoint,
		exporter:     exporter,
		siem:         siem,
		now:          func() time.Time { return time.Now().UTC() },
		polling:      splunk.ApprovedPollingPolicy(),
	}, nil
}

func (worker *Worker) RunOnce(ctx context.Context) (Result, error) {
	leased, ok, err := worker.controlPlane.Lease(ctx)
	if err != nil {
		return Result{}, err
	}
	if !ok {
		return Result{SchemaVersion: ResultSchemaVersion, Status: "idle"}, nil
	}
	result := Result{
		SchemaVersion: ResultSchemaVersion,
		Status:        "failed", JobID: leased.JobID, CorrelationID: leased.CorrelationID,
	}
	if leased.ScenarioID != "windows-process-marker" || leased.ScenarioVersion != "0.1.0" {
		if err := worker.controlPlane.Acknowledge(ctx, leased.JobID, false); err != nil {
			return result, err
		}
		result.Status = "rejected"
		result.ErrorCode = "scenario_not_supported_by_worker"
		return result, nil
	}
	if err := worker.controlPlane.Acknowledge(ctx, leased.JobID, true); err != nil {
		return result, err
	}

	if err := worker.stage(ctx, leased.JobID, "execution", "running", 0, ""); err != nil {
		return result, err
	}
	execution := worker.executor.Execute(ctx, leased.ExecutionRequest())
	if execution.Status != executor.StatusPassed {
		if err := worker.stage(ctx, leased.JobID, "execution", "failed", execution.LatencyMS, ""); err != nil {
			return result, err
		}
		result.ErrorCode = execution.ErrorCode
		if err := worker.finishFailed(ctx, leased.JobID, 0, execution); err != nil {
			return result, err
		}
		return result, nil
	}
	if err := worker.stage(ctx, leased.JobID, "execution", "passed", execution.LatencyMS, ""); err != nil {
		return result, err
	}

	if err := worker.stage(ctx, leased.JobID, "endpoint_telemetry", "running", 0, "awaiting_endpoint_event"); err != nil {
		return result, err
	}
	endpointEvidence, err := worker.endpoint.Observe(ctx, leased.CorrelationID, execution.StartedAt)
	if err != nil {
		detailCode := endpointFailureCode(err)
		if updateErr := worker.stage(ctx, leased.JobID, "endpoint_telemetry", "failed", elapsedMS(execution.StartedAt, worker.now()), detailCode); updateErr != nil {
			return result, updateErr
		}
		result.ErrorCode = endpointResultCode(err)
		if finishErr := worker.finishFailed(ctx, leased.JobID, 1, execution); finishErr != nil {
			return result, finishErr
		}
		return result, nil
	}
	if err := worker.stage(ctx, leased.JobID, "endpoint_telemetry", "passed", elapsedMS(execution.StartedAt, endpointEvidence.ObservedAtUTC), ""); err != nil {
		return result, err
	}

	if err := worker.stage(ctx, leased.JobID, "siem_ingestion", "running", 0, "siem_ingestion_delayed"); err != nil {
		return result, err
	}
	if err := worker.exporter.Export(ctx, leased.CorrelationID, endpointEvidence); err != nil {
		return worker.failSIEM(ctx, result, leased.JobID, execution)
	}
	window := splunk.SearchWindow{
		Earliest: execution.StartedAt.Add(-2 * time.Minute),
		Latest:   worker.now().Add(2 * time.Minute),
	}
	siemEvidence, _, err := splunk.PollExact(ctx, worker.siem, leased.CorrelationID, window, worker.polling)
	if err != nil {
		return worker.failSIEM(ctx, result, leased.JobID, execution)
	}
	if err := worker.stage(ctx, leased.JobID, "siem_ingestion", "passed", nonNegative(siemEvidence.IngestionLatencyMS), ""); err != nil {
		return result, err
	}

	fieldStartedAt := worker.now()
	if err := worker.stage(ctx, leased.JobID, "field_validation", "running", 0, ""); err != nil {
		return result, err
	}
	fieldResult, err := fieldvalidator.Validate(siemEvidence.FieldPresence)
	if err != nil || fieldResult.Status != fieldvalidator.StatusPassed {
		if updateErr := worker.stage(ctx, leased.JobID, "field_validation", "failed", elapsedMS(fieldStartedAt, worker.now()), "required_field_missing"); updateErr != nil {
			return result, updateErr
		}
		result.ErrorCode = "required_field_missing"
		if finishErr := worker.finishFailed(ctx, leased.JobID, 3, execution); finishErr != nil {
			return result, finishErr
		}
		return result, nil
	}
	if err := worker.stage(ctx, leased.JobID, "field_validation", "passed", elapsedMS(fieldStartedAt, worker.now()), ""); err != nil {
		return result, err
	}

	detectionStartedAt := worker.now()
	if err := worker.stage(ctx, leased.JobID, "detection", "running", 0, ""); err != nil {
		return result, err
	}
	detection, err := worker.siem.SearchDetection(ctx, leased.CorrelationID, window, splunk.BuiltInInlineDetectionPlan())
	if err != nil || !detection.Detected || detection.Status != splunk.DetectionStatusPassed {
		if updateErr := worker.stage(ctx, leased.JobID, "detection", "failed", elapsedMS(detectionStartedAt, worker.now()), "detection_result_absent"); updateErr != nil {
			return result, updateErr
		}
		result.ErrorCode = "detection_result_absent"
		if finishErr := worker.finishFailed(ctx, leased.JobID, 4, execution); finishErr != nil {
			return result, finishErr
		}
		return result, nil
	}
	if err := worker.stage(ctx, leased.JobID, "detection", "passed", elapsedMS(detectionStartedAt, worker.now()), ""); err != nil {
		return result, err
	}
	if err := worker.stage(ctx, leased.JobID, "alert", "not_tested", 0, ""); err != nil {
		return result, err
	}
	if err := worker.publishCleanup(ctx, leased.JobID, execution); err != nil {
		return result, err
	}
	if err := worker.controlPlane.Complete(ctx, leased.JobID); err != nil {
		return result, err
	}
	result.Status = "completed"
	return result, nil
}

func endpointFailureCode(err error) string {
	switch {
	case errors.Is(err, observer.ErrEventNotFound):
		return "endpoint_event_missing"
	case errors.Is(err, observer.ErrEvidenceLimit):
		return "endpoint_evidence_limit"
	case errors.Is(err, observer.ErrWindowsEventQuery):
		return "endpoint_query_failed"
	case errors.Is(err, observer.ErrWindowsEventXML):
		return "endpoint_xml_invalid"
	default:
		return "endpoint_observer_failed"
	}
}

func endpointResultCode(err error) string {
	switch {
	case errors.Is(err, observer.ErrWindowsEventEncoding):
		return "endpoint_xml_encoding_invalid"
	case errors.Is(err, observer.ErrWindowsEventDeclaration):
		return "endpoint_xml_declaration_invalid"
	case errors.Is(err, observer.ErrWindowsEventDocument):
		return "endpoint_xml_document_missing"
	case errors.Is(err, observer.ErrWindowsEventRecord):
		return "endpoint_xml_records_invalid"
	case errors.Is(err, observer.ErrWindowsEventTimestamp):
		return "endpoint_xml_timestamps_invalid"
	default:
		return endpointFailureCode(err)
	}
}

func (worker *Worker) failSIEM(ctx context.Context, result Result, jobID string, execution executor.Result) (Result, error) {
	if err := worker.stage(ctx, jobID, "siem_ingestion", "failed", 0, "siem_event_missing"); err != nil {
		return result, err
	}
	result.ErrorCode = "siem_event_missing"
	if err := worker.finishFailed(ctx, jobID, 2, execution); err != nil {
		return result, err
	}
	return result, nil
}

func (worker *Worker) finishFailed(ctx context.Context, jobID string, failedIndex int, execution executor.Result) error {
	stages := []string{"execution", "endpoint_telemetry", "siem_ingestion", "field_validation", "detection", "alert"}
	for _, stage := range stages[failedIndex+1:] {
		if err := worker.stage(ctx, jobID, stage, "not_tested", 0, ""); err != nil {
			return err
		}
	}
	if err := worker.publishCleanup(ctx, jobID, execution); err != nil {
		return err
	}
	return worker.controlPlane.Complete(ctx, jobID)
}

func (worker *Worker) publishCleanup(ctx context.Context, jobID string, execution executor.Result) error {
	if execution.CleanupStatus == executor.StatusPassed {
		return worker.stage(ctx, jobID, "cleanup", "passed", 0, "")
	}
	return worker.stage(ctx, jobID, "cleanup", "failed", 0, "cleanup_verification_failed")
}

func (worker *Worker) stage(ctx context.Context, jobID, name, status string, latency int64, detail string) error {
	return worker.controlPlane.UpdateStage(ctx, jobID, controlplane.StageUpdate{
		Stage: name, Status: status, LatencyMS: nonNegative(latency), DetailCode: detail,
	})
}

func elapsedMS(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
