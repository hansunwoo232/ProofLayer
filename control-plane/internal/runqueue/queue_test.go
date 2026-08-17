package runqueue

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"sync"
	"testing"
	"time"
)

const (
	testEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	testHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
	testOperatorID    = "7ba7b811-9dad-41d1-80b4-00c04fd430c8"
	testIdempotency   = "run_test_0123456789ABCDEF012345"
)

func newTestQueue(t *testing.T, capacity int) (*Queue, ed25519.PublicKey) {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
	queue, err := New(Config{
		Capacity:          capacity,
		EnvironmentID:     testEnvironmentID,
		HostID:            testHostID,
		RequestedBy:       testOperatorID,
		SigningKeyID:      "local-poc-key-01",
		SigningPrivateKey: privateKey,
		Now: func() time.Time {
			return time.Date(2026, time.August, 17, 20, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return queue, privateKey.Public().(ed25519.PublicKey)
}

func validRequest() CreateRequest {
	return CreateRequest{
		SchemaVersion:   SchemaVersion,
		EnvironmentID:   testEnvironmentID,
		HostID:          testHostID,
		ScenarioID:      "windows-process-marker",
		ScenarioVersion: "0.1.0",
	}
}

func TestEnqueueCreatesSignedHostBoundJob(t *testing.T) {
	queue, publicKey := newTestQueue(t, 4)
	receipt, err := queue.Enqueue(testIdempotency, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "queued" || receipt.Replayed || queue.Depth() != 1 {
		t.Fatalf("receipt = %+v, depth = %d", receipt, queue.Depth())
	}
	job, ok := queue.Lease(testEnvironmentID, testHostID)
	if !ok {
		t.Fatal("bound Runner could not lease its job")
	}
	if job.JobID != receipt.JobID || job.CorrelationID != receipt.CorrelationID {
		t.Fatalf("job = %+v, receipt = %+v", job, receipt)
	}
	if !Verify(job, publicKey) {
		t.Fatal("job signature did not verify")
	}
	if job.ExpiresAt.Sub(job.RequestedAt) != jobLifetime {
		t.Fatalf("job lifetime = %s", job.ExpiresAt.Sub(job.RequestedAt))
	}
	if len(job.Parameters) != 0 {
		t.Fatalf("parameters = %#v", job.Parameters)
	}
}

func TestDuplicateSubmissionReturnsOneJob(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	const callers = 12
	receipts := make(chan Receipt, callers)
	errorsFound := make(chan error, callers)
	var waitGroup sync.WaitGroup
	for range callers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			receipt, err := queue.Enqueue(testIdempotency, validRequest())
			if err != nil {
				errorsFound <- err
				return
			}
			receipts <- receipt
		}()
	}
	waitGroup.Wait()
	close(receipts)
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
	var jobID string
	for receipt := range receipts {
		if jobID == "" {
			jobID = receipt.JobID
		}
		if receipt.JobID != jobID {
			t.Fatalf("duplicate created job %q, want %q", receipt.JobID, jobID)
		}
	}
	if queue.Depth() != 1 {
		t.Fatalf("queue depth = %d, want 1", queue.Depth())
	}
	if _, ok := queue.Lease(testEnvironmentID, testHostID); !ok {
		t.Fatal("first lease failed")
	}
	if _, ok := queue.Lease(testEnvironmentID, testHostID); ok {
		t.Fatal("same job leased twice")
	}
}

func TestIdempotencyKeyCannotChangeRequest(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	if _, err := queue.Enqueue(testIdempotency, validRequest()); err != nil {
		t.Fatal(err)
	}
	changed := validRequest()
	changed.ScenarioID = "windows-registry-run-key-canary"
	if _, err := queue.Enqueue(testIdempotency, changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v", err)
	}
}

func TestQueueRejectsUnsafeRequestsAndCapacityOverflow(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CreateRequest)
	}{
		{name: "schema", mutate: func(request *CreateRequest) { request.SchemaVersion = "2.0" }},
		{name: "environment", mutate: func(request *CreateRequest) { request.EnvironmentID = testOperatorID }},
		{name: "host", mutate: func(request *CreateRequest) { request.HostID = testOperatorID }},
		{name: "scenario", mutate: func(request *CreateRequest) { request.ScenarioID = "arbitrary-command" }},
		{name: "version", mutate: func(request *CreateRequest) { request.ScenarioVersion = "9.9.9" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queue, _ := newTestQueue(t, 2)
			request := validRequest()
			test.mutate(&request)
			if _, err := queue.Enqueue(testIdempotency, request); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	queue, _ := newTestQueue(t, 1)
	if _, err := queue.Enqueue(testIdempotency, validRequest()); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.Enqueue("run_test_FEDCBA9876543210012345", validRequest()); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("error = %v", err)
	}
}

func TestLeaseIsRestrictedToBoundIdentity(t *testing.T) {
	queue, _ := newTestQueue(t, 2)
	if _, err := queue.Enqueue(testIdempotency, validRequest()); err != nil {
		t.Fatal(err)
	}
	if _, ok := queue.Lease(testEnvironmentID, testOperatorID); ok {
		t.Fatal("wrong host leased job")
	}
	if _, ok := queue.Lease(testOperatorID, testHostID); ok {
		t.Fatal("wrong environment leased job")
	}
	if queue.Depth() != 1 {
		t.Fatalf("unauthorized lease removed job")
	}
}

func TestExpiredJobCannotBeLeased(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x24}, ed25519.SeedSize))
	now := time.Date(2026, time.August, 17, 20, 0, 0, 0, time.UTC)
	queue, err := New(Config{
		Capacity:          2,
		EnvironmentID:     testEnvironmentID,
		HostID:            testHostID,
		RequestedBy:       testOperatorID,
		SigningKeyID:      "local-poc-key-01",
		SigningPrivateKey: privateKey,
		Now:               func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := queue.Enqueue(testIdempotency, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(jobLifetime)
	if _, ok := queue.Lease(testEnvironmentID, testHostID); ok {
		t.Fatal("expired job was leased")
	}
	if queue.Depth() != 0 {
		t.Fatalf("expired job remained in queue")
	}
	snapshot, ok := queue.Status(receipt.JobID)
	if !ok || snapshot.Status != JobStatusExpired || !snapshot.Terminal {
		t.Fatalf("expired snapshot = %+v, exists = %v", snapshot, ok)
	}
}

func TestExpiredLeaseCannotBeAcknowledged(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x25}, ed25519.SeedSize))
	now := time.Date(2026, time.August, 17, 20, 0, 0, 0, time.UTC)
	queue, err := New(Config{
		Capacity:          2,
		EnvironmentID:     testEnvironmentID,
		HostID:            testHostID,
		RequestedBy:       testOperatorID,
		SigningKeyID:      "local-poc-key-01",
		SigningPrivateKey: privateKey,
		Now:               func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := queue.Enqueue(testIdempotency, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := queue.Lease(testEnvironmentID, testHostID); !ok {
		t.Fatal("job lease failed")
	}
	now = now.Add(jobLifetime)
	if err := queue.Acknowledge(testEnvironmentID, testHostID, receipt.JobID, true); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expired acknowledgement error = %v", err)
	}
	snapshot, _ := queue.Status(receipt.JobID)
	if snapshot.Status != JobStatusExpired || !snapshot.Terminal {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}
