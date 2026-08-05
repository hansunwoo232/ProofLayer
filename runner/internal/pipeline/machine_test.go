package pipeline

import (
	"errors"
	"testing"
	"time"
)

func TestHappyPathWithOptionalAlertSkipped(t *testing.T) {
	machine, startedAt := newMachine(t, Options{DetectionRequired: true, AlertRequired: false})
	pass(t, machine, StageExecution, startedAt.Add(time.Second))
	pass(t, machine, StageEndpointTelemetry, startedAt.Add(2*time.Second))
	pass(t, machine, StageSIEMIngestion, startedAt.Add(3*time.Second))
	pass(t, machine, StageFieldValidation, startedAt.Add(4*time.Second))
	pass(t, machine, StageDetection, startedAt.Add(5*time.Second))
	if err := machine.SkipOptional(StageAlert); err != nil {
		t.Fatal(err)
	}
	pass(t, machine, StageCleanup, startedAt.Add(6*time.Second))

	snapshot := machine.Snapshot()
	if snapshot.OverallStatus != OverallPassed || snapshot.CompletedAt == nil {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	assertStage(t, snapshot, StageAlert, StatusNotTested)
	assertStage(t, snapshot, StageCleanup, StatusPassed)
}

func TestUpstreamFailureMarksDependentsNotTestedButKeepsCleanupPending(t *testing.T) {
	machine, startedAt := newMachine(t, Options{DetectionRequired: true, AlertRequired: true})
	pass(t, machine, StageExecution, startedAt.Add(time.Second))
	pass(t, machine, StageEndpointTelemetry, startedAt.Add(2*time.Second))
	pass(t, machine, StageSIEMIngestion, startedAt.Add(3*time.Second))
	if err := machine.Start(StageFieldValidation, startedAt.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := machine.Complete(StageFieldValidation, StatusFailed, startedAt.Add(4*time.Second+50*time.Millisecond), "REQUIRED_FIELD_MISSING"); err != nil {
		t.Fatal(err)
	}

	snapshot := machine.Snapshot()
	assertStage(t, snapshot, StageDetection, StatusNotTested)
	assertStage(t, snapshot, StageAlert, StatusNotTested)
	assertStage(t, snapshot, StageCleanup, StatusPending)
	if snapshot.OverallStatus != OverallFailed {
		t.Fatalf("overall status = %s", snapshot.OverallStatus)
	}

	pass(t, machine, StageCleanup, startedAt.Add(5*time.Second))
	if machine.Snapshot().OverallStatus != OverallFailed {
		t.Fatal("cleanup pass hid the upstream failure")
	}
}

func TestCleanupFailureAlwaysFailsRun(t *testing.T) {
	machine, startedAt := newMachine(t, Options{})
	pass(t, machine, StageExecution, startedAt.Add(time.Second))
	pass(t, machine, StageEndpointTelemetry, startedAt.Add(2*time.Second))
	pass(t, machine, StageSIEMIngestion, startedAt.Add(3*time.Second))
	pass(t, machine, StageFieldValidation, startedAt.Add(4*time.Second))
	if err := machine.SkipOptional(StageDetection); err != nil {
		t.Fatal(err)
	}
	if err := machine.SkipOptional(StageAlert); err != nil {
		t.Fatal(err)
	}
	if err := machine.Start(StageCleanup, startedAt.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := machine.Complete(StageCleanup, StatusFailed, startedAt.Add(6*time.Second), "CLEANUP_FAILED"); err != nil {
		t.Fatal(err)
	}
	if machine.Snapshot().OverallStatus != OverallFailed {
		t.Fatal("cleanup failure did not fail the run")
	}
}

func TestMachineRejectsOutOfOrderAndUnsafeTransitions(t *testing.T) {
	machine, startedAt := newMachine(t, Options{DetectionRequired: true})
	if err := machine.Start(StageSIEMIngestion, startedAt.Add(time.Second)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("out-of-order error = %v", err)
	}
	if err := machine.SkipOptional(StageExecution); !errors.Is(err, ErrRequiredStage) {
		t.Fatalf("required skip error = %v", err)
	}
	if err := machine.Start(StageExecution, startedAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := machine.Complete(StageExecution, StatusPassed, startedAt.Add(2*time.Second), "SHOULD_NOT_EXIST"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("pass-with-error error = %v", err)
	}
	if err := machine.Complete(StageExecution, StatusFailed, startedAt.Add(2*time.Second), ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("failure-without-error error = %v", err)
	}
}

func TestDegradedStageProducesDegradedRun(t *testing.T) {
	machine, startedAt := newMachine(t, Options{})
	if err := machine.Start(StageExecution, startedAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := machine.Complete(StageExecution, StatusDegraded, startedAt.Add(2*time.Second), "EXECUTION_SLO_EXCEEDED"); err != nil {
		t.Fatal(err)
	}
	pass(t, machine, StageEndpointTelemetry, startedAt.Add(3*time.Second))
	pass(t, machine, StageSIEMIngestion, startedAt.Add(4*time.Second))
	pass(t, machine, StageFieldValidation, startedAt.Add(5*time.Second))
	if err := machine.SkipOptional(StageDetection); err != nil {
		t.Fatal(err)
	}
	if err := machine.SkipOptional(StageAlert); err != nil {
		t.Fatal(err)
	}
	pass(t, machine, StageCleanup, startedAt.Add(6*time.Second))
	if machine.Snapshot().OverallStatus != OverallDegraded {
		t.Fatalf("overall status = %s", machine.Snapshot().OverallStatus)
	}
}

func TestSnapshotIsDefensiveCopy(t *testing.T) {
	machine, startedAt := newMachine(t, Options{})
	pass(t, machine, StageExecution, startedAt.Add(time.Second))
	first := machine.Snapshot()
	first.Stages[0].Status = StatusFailed
	*first.Stages[0].StartedAt = time.Time{}
	second := machine.Snapshot()
	if second.Stages[0].Status != StatusPassed || second.Stages[0].StartedAt.IsZero() {
		t.Fatal("snapshot mutation changed machine state")
	}
}

func newMachine(t *testing.T, options Options) (*Machine, time.Time) {
	t.Helper()
	startedAt := time.Date(2026, time.August, 5, 20, 0, 0, 0, time.UTC)
	machine, err := New(options, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	return machine, startedAt
}

func pass(t *testing.T, machine *Machine, stage Stage, start time.Time) {
	t.Helper()
	if err := machine.Start(stage, start); err != nil {
		t.Fatalf("start %s: %v", stage, err)
	}
	if err := machine.Complete(stage, StatusPassed, start.Add(100*time.Millisecond), ""); err != nil {
		t.Fatalf("complete %s: %v", stage, err)
	}
}

func assertStage(t *testing.T, snapshot Snapshot, stage Stage, expected StageStatus) {
	t.Helper()
	for _, result := range snapshot.Stages {
		if result.Stage == stage {
			if result.Status != expected {
				t.Fatalf("stage %s status = %s, want %s", stage, result.Status, expected)
			}
			return
		}
	}
	t.Fatalf("stage %s was missing", stage)
}
