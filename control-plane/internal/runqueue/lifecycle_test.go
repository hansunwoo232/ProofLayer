package runqueue

import (
	"errors"
	"testing"
)

func leaseAndAcknowledge(t *testing.T, queue *Queue) Receipt {
	t.Helper()
	receipt, err := queue.Enqueue(testIdempotency, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := queue.Lease(testEnvironmentID, testHostID); !ok {
		t.Fatal("job lease failed")
	}
	if err := queue.Acknowledge(testEnvironmentID, testHostID, receipt.JobID, true); err != nil {
		t.Fatal(err)
	}
	return receipt
}

func updateStage(t *testing.T, queue *Queue, jobID, stage string, status StageStatus, latency int64, detail string) {
	t.Helper()
	if err := queue.UpdateStage(testEnvironmentID, testHostID, jobID, StageUpdate{
		Stage: stage, Status: status, LatencyMS: latency, DetailCode: detail,
	}); err != nil {
		t.Fatalf("update %s to %s: %v", stage, status, err)
	}
}

func TestLifecyclePublishesDelayedStageAndFailedCompletion(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	receipt := leaseAndAcknowledge(t, queue)

	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, StageUpdate{
		Stage: "endpoint_telemetry", Status: StageStatusRunning,
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("out-of-order update error = %v", err)
	}
	updateStage(t, queue, receipt.JobID, "execution", StageStatusPassed, 74, "")
	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, StageUpdate{
		Stage: "execution", Status: StageStatusFailed, LatencyMS: 75,
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("terminal rewrite error = %v", err)
	}
	updateStage(t, queue, receipt.JobID, "endpoint_telemetry", StageStatusRunning, 0, "endpoint_event_delayed")

	snapshot, ok := queue.Status(receipt.JobID)
	if !ok {
		t.Fatal("job status missing")
	}
	if snapshot.Status != JobStatusRunning || snapshot.Terminal {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if snapshot.Stages[1].Status != StageStatusRunning || snapshot.Stages[1].DetailCode != "endpoint_event_delayed" {
		t.Fatalf("endpoint stage = %+v", snapshot.Stages[1])
	}

	updateStage(t, queue, receipt.JobID, "endpoint_telemetry", StageStatusPassed, 4200, "")
	updateStage(t, queue, receipt.JobID, "siem_ingestion", StageStatusPassed, 1680, "")
	updateStage(t, queue, receipt.JobID, "field_validation", StageStatusFailed, 8, "required_field_missing")

	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, StageUpdate{
		Stage: "detection", Status: StageStatusPassed, LatencyMS: 12,
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("downstream PASS after failure error = %v", err)
	}
	updateStage(t, queue, receipt.JobID, "detection", StageStatusNotTested, 0, "")
	updateStage(t, queue, receipt.JobID, "alert", StageStatusNotTested, 0, "")
	updateStage(t, queue, receipt.JobID, "cleanup", StageStatusPassed, 31, "")
	if err := queue.Complete(testEnvironmentID, testHostID, receipt.JobID); err != nil {
		t.Fatal(err)
	}

	snapshot, ok = queue.Status(receipt.JobID)
	if !ok || snapshot.Status != JobStatusFailed || !snapshot.Terminal {
		t.Fatalf("terminal snapshot = %+v, exists = %v", snapshot, ok)
	}
	if err := queue.Complete(testEnvironmentID, testHostID, receipt.JobID); err != nil {
		t.Fatalf("idempotent completion failed: %v", err)
	}
}

func TestLifecycleCompletesOnlyAfterEveryStage(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	receipt := leaseAndAcknowledge(t, queue)
	for _, stage := range stageOrder {
		updateStage(t, queue, receipt.JobID, stage, StageStatusPassed, 1, "")
	}
	if err := queue.Complete(testEnvironmentID, testHostID, receipt.JobID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := queue.Status(receipt.JobID)
	if snapshot.Status != JobStatusCompleted || !snapshot.Terminal {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestLifecycleAllowsOptionalAlertStageToRemainNotTested(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	receipt := leaseAndAcknowledge(t, queue)
	for _, stage := range stageOrder[:5] {
		updateStage(t, queue, receipt.JobID, stage, StageStatusPassed, 1, "")
	}
	updateStage(t, queue, receipt.JobID, "alert", StageStatusNotTested, 0, "")
	updateStage(t, queue, receipt.JobID, "cleanup", StageStatusPassed, 1, "")
	if err := queue.Complete(testEnvironmentID, testHostID, receipt.JobID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := queue.Status(receipt.JobID)
	if snapshot.Status != JobStatusCompleted || !snapshot.Terminal {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestLifecycleRejectsWrongIdentityAndInvalidDetail(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	receipt, err := queue.Enqueue(testIdempotency, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := queue.Lease(testEnvironmentID, testHostID); !ok {
		t.Fatal("job lease failed")
	}
	if err := queue.Acknowledge(testOperatorID, testHostID, receipt.JobID, true); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("wrong identity error = %v", err)
	}
	if err := queue.Acknowledge(testEnvironmentID, testHostID, receipt.JobID, true); err != nil {
		t.Fatal(err)
	}
	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, StageUpdate{
		Stage: "execution", Status: StageStatusRunning, DetailCode: "arbitrary_runner_text",
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("invalid detail error = %v", err)
	}
	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, StageUpdate{
		Stage: "execution", Status: StageStatusRunning, DetailCode: "endpoint_event_delayed",
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("wrong-stage detail error = %v", err)
	}
	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, StageUpdate{
		Stage: "execution", Status: StageStatusNotTested,
	}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("unjustified not-tested error = %v", err)
	}
}

func TestRejectedLeaseIsTerminalAndIdempotent(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	receipt, err := queue.Enqueue(testIdempotency, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := queue.Lease(testEnvironmentID, testHostID); !ok {
		t.Fatal("job lease failed")
	}
	if err := queue.Acknowledge(testEnvironmentID, testHostID, receipt.JobID, false); err != nil {
		t.Fatal(err)
	}
	if err := queue.Acknowledge(testEnvironmentID, testHostID, receipt.JobID, false); err != nil {
		t.Fatalf("idempotent rejection failed: %v", err)
	}
	snapshot, _ := queue.Status(receipt.JobID)
	if snapshot.Status != JobStatusRejected || !snapshot.Terminal {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	for _, stage := range snapshot.Stages {
		if stage.Status != StageStatusNotTested {
			t.Fatalf("stage = %+v", stage)
		}
	}
}
