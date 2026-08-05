package pipeline

import (
	"errors"
	"fmt"
	"time"
)

type Stage string

const (
	StageExecution         Stage = "execution"
	StageEndpointTelemetry Stage = "endpoint_telemetry"
	StageSIEMIngestion     Stage = "siem_ingestion"
	StageFieldValidation   Stage = "field_validation"
	StageDetection         Stage = "detection"
	StageAlert             Stage = "alert"
	StageCleanup           Stage = "cleanup"
)

var orderedStages = []Stage{
	StageExecution,
	StageEndpointTelemetry,
	StageSIEMIngestion,
	StageFieldValidation,
	StageDetection,
	StageAlert,
	StageCleanup,
}

type StageStatus string

const (
	StatusPending   StageStatus = "pending"
	StatusRunning   StageStatus = "running"
	StatusPassed    StageStatus = "passed"
	StatusFailed    StageStatus = "failed"
	StatusDegraded  StageStatus = "degraded"
	StatusNotTested StageStatus = "not_tested"
)

type OverallStatus string

const (
	OverallPending  OverallStatus = "pending"
	OverallRunning  OverallStatus = "running"
	OverallPassed   OverallStatus = "passed"
	OverallFailed   OverallStatus = "failed"
	OverallDegraded OverallStatus = "degraded"
)

var (
	ErrInvalidTransition = errors.New("invalid pipeline transition")
	ErrInvalidTimestamp  = errors.New("invalid pipeline timestamp")
	ErrRequiredStage     = errors.New("required stage cannot be skipped")
)

type Options struct {
	DetectionRequired bool
	AlertRequired     bool
}

type StageResult struct {
	Stage       Stage       `json:"stage"`
	Status      StageStatus `json:"status"`
	Required    bool        `json:"required"`
	StartedAt   *time.Time  `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at"`
	LatencyMS   *int64      `json:"latency_ms"`
	ErrorCode   string      `json:"error_code,omitempty"`
}

type Snapshot struct {
	OverallStatus OverallStatus `json:"overall_status"`
	StartedAt     time.Time     `json:"started_at"`
	CompletedAt   *time.Time    `json:"completed_at"`
	Stages        []StageResult `json:"stages"`
}

type Machine struct {
	startedAt time.Time
	stages    []StageResult
}

func New(options Options, startedAt time.Time) (*Machine, error) {
	if startedAt.IsZero() {
		return nil, ErrInvalidTimestamp
	}
	required := map[Stage]bool{
		StageExecution:         true,
		StageEndpointTelemetry: true,
		StageSIEMIngestion:     true,
		StageFieldValidation:   true,
		StageDetection:         options.DetectionRequired,
		StageAlert:             options.AlertRequired,
		StageCleanup:           true,
	}
	stages := make([]StageResult, 0, len(orderedStages))
	for _, stage := range orderedStages {
		stages = append(stages, StageResult{
			Stage:    stage,
			Status:   StatusPending,
			Required: required[stage],
		})
	}
	return &Machine{startedAt: startedAt.UTC(), stages: stages}, nil
}

func (machine *Machine) Start(stage Stage, at time.Time) error {
	index, ok := stageIndex(stage)
	if !ok {
		return transition(stage, "unknown_stage")
	}
	if at.IsZero() || at.Before(machine.startedAt) {
		return ErrInvalidTimestamp
	}
	if machine.stages[index].Status != StatusPending {
		return transition(stage, "not_pending")
	}
	if !machine.ready(index) {
		return transition(stage, "out_of_order")
	}
	timestamp := at.UTC()
	machine.stages[index].Status = StatusRunning
	machine.stages[index].StartedAt = &timestamp
	return nil
}

func (machine *Machine) Complete(stage Stage, status StageStatus, at time.Time, errorCode string) error {
	index, ok := stageIndex(stage)
	if !ok {
		return transition(stage, "unknown_stage")
	}
	result := &machine.stages[index]
	if result.Status != StatusRunning || result.StartedAt == nil {
		return transition(stage, "not_running")
	}
	if status != StatusPassed && status != StatusFailed && status != StatusDegraded {
		return transition(stage, "invalid_terminal_status")
	}
	if at.IsZero() || at.Before(*result.StartedAt) {
		return ErrInvalidTimestamp
	}
	if status == StatusPassed && errorCode != "" {
		return transition(stage, "pass_with_error")
	}
	if status == StatusFailed && errorCode == "" {
		return transition(stage, "failure_without_error")
	}

	completedAt := at.UTC()
	latency := completedAt.Sub(*result.StartedAt).Milliseconds()
	result.Status = status
	result.CompletedAt = &completedAt
	result.LatencyMS = &latency
	result.ErrorCode = errorCode
	if status == StatusFailed && stage != StageCleanup {
		machine.propagateNotTested(index + 1)
	}
	return nil
}

func (machine *Machine) SkipOptional(stage Stage) error {
	index, ok := stageIndex(stage)
	if !ok {
		return transition(stage, "unknown_stage")
	}
	result := &machine.stages[index]
	if result.Required {
		return fmt.Errorf("%w: %s", ErrRequiredStage, stage)
	}
	if result.Status != StatusPending || !machine.ready(index) {
		return transition(stage, "cannot_skip")
	}
	result.Status = StatusNotTested
	return nil
}

func (machine *Machine) Snapshot() Snapshot {
	stages := make([]StageResult, len(machine.stages))
	copy(stages, machine.stages)
	for index := range stages {
		stages[index].StartedAt = copyTime(stages[index].StartedAt)
		stages[index].CompletedAt = copyTime(stages[index].CompletedAt)
		if stages[index].LatencyMS != nil {
			value := *stages[index].LatencyMS
			stages[index].LatencyMS = &value
		}
	}
	snapshot := Snapshot{
		OverallStatus: machine.overallStatus(),
		StartedAt:     machine.startedAt,
		Stages:        stages,
	}
	cleanup := machine.stages[len(machine.stages)-1]
	if cleanup.Status == StatusPassed || cleanup.Status == StatusFailed || cleanup.Status == StatusDegraded {
		snapshot.CompletedAt = copyTime(cleanup.CompletedAt)
	}
	return snapshot
}

func (machine *Machine) ready(index int) bool {
	if index == len(machine.stages)-1 {
		for prior := 0; prior < index; prior++ {
			if machine.stages[prior].Status == StatusPending || machine.stages[prior].Status == StatusRunning {
				return false
			}
		}
		return true
	}
	for prior := 0; prior < index; prior++ {
		status := machine.stages[prior].Status
		if status != StatusPassed && status != StatusDegraded && status != StatusNotTested {
			return false
		}
	}
	return true
}

func (machine *Machine) propagateNotTested(start int) {
	for index := start; index < len(machine.stages)-1; index++ {
		if machine.stages[index].Status == StatusPending {
			machine.stages[index].Status = StatusNotTested
		}
	}
}

func (machine *Machine) overallStatus() OverallStatus {
	cleanup := machine.stages[len(machine.stages)-1]
	if cleanup.Status == StatusFailed {
		return OverallFailed
	}
	for _, result := range machine.stages {
		if result.Required && result.Status == StatusFailed {
			return OverallFailed
		}
	}
	if cleanup.Status == StatusPending || cleanup.Status == StatusRunning {
		if machine.stages[0].Status == StatusPending {
			return OverallPending
		}
		return OverallRunning
	}
	for _, result := range machine.stages {
		if result.Required && (result.Status == StatusPending || result.Status == StatusRunning || result.Status == StatusNotTested) {
			return OverallRunning
		}
		if result.Status == StatusDegraded {
			return OverallDegraded
		}
	}
	return OverallPassed
}

func stageIndex(stage Stage) (int, bool) {
	for index, candidate := range orderedStages {
		if stage == candidate {
			return index, true
		}
	}
	return 0, false
}

func transition(stage Stage, reason string) error {
	return fmt.Errorf("%w: %s:%s", ErrInvalidTransition, stage, reason)
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
