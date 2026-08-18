package runqueue

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrJobNotFound       = errors.New("job was not found")
	ErrInvalidTransition = errors.New("invalid job lifecycle transition")
)

type JobStatus string

const (
	JobStatusQueued       JobStatus = "queued"
	JobStatusLeased       JobStatus = "leased"
	JobStatusAcknowledged JobStatus = "acknowledged"
	JobStatusRunning      JobStatus = "running"
	JobStatusCompleted    JobStatus = "completed"
	JobStatusFailed       JobStatus = "failed"
	JobStatusRejected     JobStatus = "rejected"
	JobStatusExpired      JobStatus = "expired"
)

type StageStatus string

const (
	StageStatusPending   StageStatus = "pending"
	StageStatusRunning   StageStatus = "running"
	StageStatusPassed    StageStatus = "passed"
	StageStatusFailed    StageStatus = "failed"
	StageStatusNotTested StageStatus = "not_tested"
)

var stageOrder = []string{
	"execution",
	"endpoint_telemetry",
	"siem_ingestion",
	"field_validation",
	"detection",
	"alert",
	"cleanup",
}

type detailPolicy struct {
	stage  string
	status StageStatus
}

var detailPolicies = map[string]detailPolicy{
	"awaiting_endpoint_event":     {stage: "endpoint_telemetry", status: StageStatusRunning},
	"endpoint_event_delayed":      {stage: "endpoint_telemetry", status: StageStatusRunning},
	"endpoint_event_missing":      {stage: "endpoint_telemetry", status: StageStatusFailed},
	"endpoint_evidence_limit":     {stage: "endpoint_telemetry", status: StageStatusFailed},
	"endpoint_query_failed":       {stage: "endpoint_telemetry", status: StageStatusFailed},
	"endpoint_xml_invalid":        {stage: "endpoint_telemetry", status: StageStatusFailed},
	"endpoint_observer_failed":    {stage: "endpoint_telemetry", status: StageStatusFailed},
	"siem_ingestion_delayed":      {stage: "siem_ingestion", status: StageStatusRunning},
	"siem_event_missing":          {stage: "siem_ingestion", status: StageStatusFailed},
	"required_field_missing":      {stage: "field_validation", status: StageStatusFailed},
	"detection_result_absent":     {stage: "detection", status: StageStatusFailed},
	"alert_delivery_delayed":      {stage: "alert", status: StageStatusRunning},
	"cleanup_verification_failed": {stage: "cleanup", status: StageStatusFailed},
}

type StageSnapshot struct {
	Stage      string      `json:"stage"`
	Status     StageStatus `json:"status"`
	LatencyMS  *int64      `json:"latency_ms,omitempty"`
	DetailCode string      `json:"detail_code,omitempty"`
}

type StatusSnapshot struct {
	SchemaVersion string          `json:"schema_version"`
	JobID         string          `json:"job_id"`
	CorrelationID string          `json:"correlation_id"`
	Status        JobStatus       `json:"status"`
	UpdatedAt     time.Time       `json:"updated_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
	Terminal      bool            `json:"terminal"`
	Stages        []StageSnapshot `json:"stages"`
}

type StageUpdate struct {
	Stage      string
	Status     StageStatus
	LatencyMS  int64
	DetailCode string
}

type lifecycleRecord struct {
	Job       Job
	Status    JobStatus
	UpdatedAt time.Time
	Terminal  bool
	Stages    []StageSnapshot
}

func newLifecycleRecord(job Job) *lifecycleRecord {
	stages := make([]StageSnapshot, len(stageOrder))
	for index, stage := range stageOrder {
		stages[index] = StageSnapshot{Stage: stage, Status: StageStatusPending}
	}
	return &lifecycleRecord{
		Job:       cloneJob(job),
		Status:    JobStatusQueued,
		UpdatedAt: job.RequestedAt,
		Stages:    stages,
	}
}

func (queue *Queue) Acknowledge(environmentID, hostID, jobID string, accepted bool) error {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneExpiredLocked(queue.now().UTC())

	record, err := queue.authorizedRecordLocked(environmentID, hostID, jobID)
	if err != nil {
		return err
	}
	if accepted {
		if record.Status == JobStatusAcknowledged || record.Status == JobStatusRunning ||
			record.Status == JobStatusCompleted || record.Status == JobStatusFailed {
			return nil
		}
		if record.Status != JobStatusLeased {
			return ErrInvalidTransition
		}
		record.Status = JobStatusAcknowledged
	} else {
		if record.Status == JobStatusRejected {
			return nil
		}
		if record.Status != JobStatusLeased {
			return ErrInvalidTransition
		}
		record.Status = JobStatusRejected
		record.Terminal = true
		markPendingNotTested(record)
	}
	record.UpdatedAt = queue.now().UTC()
	return nil
}

func (queue *Queue) UpdateStage(environmentID, hostID, jobID string, update StageUpdate) error {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	record, err := queue.authorizedRecordLocked(environmentID, hostID, jobID)
	if err != nil {
		return err
	}
	if record.Status != JobStatusAcknowledged && record.Status != JobStatusRunning {
		return ErrInvalidTransition
	}
	index := stageIndex(update.Stage)
	if index < 0 || !validStageStatus(update.Status) || update.LatencyMS < 0 || update.LatencyMS > 300_000 {
		return ErrInvalidTransition
	}
	if !validDetail(update) {
		return ErrInvalidTransition
	}
	if !priorStagesTerminal(record.Stages, index) {
		return ErrInvalidTransition
	}
	priorFailed := priorStageFailed(record.Stages, index)
	if update.Status == StageStatusNotTested &&
		((!priorFailed && update.Stage != "alert") || update.Stage == "cleanup") {
		return ErrInvalidTransition
	}
	if priorFailed && update.Stage != "cleanup" && update.Status != StageStatusNotTested {
		return ErrInvalidTransition
	}
	current := &record.Stages[index]
	if isTerminalStageStatus(current.Status) {
		if current.Status == update.Status && current.DetailCode == update.DetailCode &&
			latencyValue(current.LatencyMS) == update.LatencyMS {
			return nil
		}
		return ErrInvalidTransition
	}
	if current.Status == StageStatusRunning && update.Status == StageStatusPending {
		return ErrInvalidTransition
	}
	latency := update.LatencyMS
	current.Status = update.Status
	current.DetailCode = update.DetailCode
	if update.Status == StageStatusRunning || update.Status == StageStatusPending {
		current.LatencyMS = nil
	} else {
		current.LatencyMS = &latency
	}
	record.Status = JobStatusRunning
	record.UpdatedAt = queue.now().UTC()
	return nil
}

func (queue *Queue) Complete(environmentID, hostID, jobID string) error {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	record, err := queue.authorizedRecordLocked(environmentID, hostID, jobID)
	if err != nil {
		return err
	}
	if record.Status == JobStatusCompleted || record.Status == JobStatusFailed {
		return nil
	}
	if record.Status != JobStatusRunning {
		return ErrInvalidTransition
	}
	for _, stage := range record.Stages {
		if !isTerminalStageStatus(stage.Status) {
			return ErrInvalidTransition
		}
	}
	cleanup := record.Stages[len(record.Stages)-1]
	if cleanup.Status != StageStatusPassed && cleanup.Status != StageStatusFailed {
		return ErrInvalidTransition
	}
	record.Status = JobStatusCompleted
	for _, stage := range record.Stages {
		if stage.Status == StageStatusFailed {
			record.Status = JobStatusFailed
			break
		}
	}
	record.Terminal = true
	record.UpdatedAt = queue.now().UTC()
	return nil
}

func (queue *Queue) Status(jobID string) (StatusSnapshot, bool) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneExpiredLocked(queue.now().UTC())

	record, ok := queue.lifecycle[strings.ToLower(jobID)]
	if !ok {
		return StatusSnapshot{}, false
	}
	return snapshot(record), true
}

func (queue *Queue) authorizedRecordLocked(environmentID, hostID, jobID string) (*lifecycleRecord, error) {
	record, ok := queue.lifecycle[strings.ToLower(jobID)]
	if !ok {
		return nil, ErrJobNotFound
	}
	if record.Job.EnvironmentID != strings.ToLower(environmentID) || record.Job.HostID != strings.ToLower(hostID) {
		return nil, fmt.Errorf("%w: identity_binding", ErrInvalidRequest)
	}
	return record, nil
}

func snapshot(record *lifecycleRecord) StatusSnapshot {
	stages := make([]StageSnapshot, len(record.Stages))
	for index, stage := range record.Stages {
		stages[index] = stage
		if stage.LatencyMS != nil {
			value := *stage.LatencyMS
			stages[index].LatencyMS = &value
		}
	}
	return StatusSnapshot{
		SchemaVersion: SchemaVersion,
		JobID:         record.Job.JobID,
		CorrelationID: record.Job.CorrelationID,
		Status:        record.Status,
		UpdatedAt:     record.UpdatedAt,
		ExpiresAt:     record.Job.ExpiresAt,
		Terminal:      record.Terminal,
		Stages:        stages,
	}
}

func stageIndex(stage string) int {
	for index, candidate := range stageOrder {
		if candidate == stage {
			return index
		}
	}
	return -1
}

func validStageStatus(status StageStatus) bool {
	return status == StageStatusRunning || status == StageStatusPassed ||
		status == StageStatusFailed || status == StageStatusNotTested
}

func validDetail(update StageUpdate) bool {
	if update.DetailCode == "" {
		return true
	}
	policy, ok := detailPolicies[update.DetailCode]
	return ok && policy.stage == update.Stage && policy.status == update.Status
}

func priorStagesTerminal(stages []StageSnapshot, index int) bool {
	for _, stage := range stages[:index] {
		if !isTerminalStageStatus(stage.Status) {
			return false
		}
	}
	return true
}

func priorStageFailed(stages []StageSnapshot, index int) bool {
	for _, stage := range stages[:index] {
		if stage.Status == StageStatusFailed {
			return true
		}
	}
	return false
}

func isTerminalStageStatus(status StageStatus) bool {
	return status == StageStatusPassed || status == StageStatusFailed || status == StageStatusNotTested
}

func latencyValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func markPendingNotTested(record *lifecycleRecord) {
	zero := int64(0)
	for index := range record.Stages {
		if record.Stages[index].Status == StageStatusPending {
			record.Stages[index].Status = StageStatusNotTested
			record.Stages[index].LatencyMS = &zero
		}
	}
}
