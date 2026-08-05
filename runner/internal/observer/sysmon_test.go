package observer

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testCorrelationID = "PL-0123456789ABCDEF0123456789ABCDEF"

type sequenceSource struct {
	responses [][]Event
	err       error
	calls     int
}

func (source *sequenceSource) RecentProcessEvents(context.Context, time.Time) ([]Event, error) {
	source.calls++
	if source.err != nil {
		return nil, source.err
	}
	index := source.calls - 1
	if index >= len(source.responses) {
		index = len(source.responses) - 1
	}
	if index < 0 {
		return nil, nil
	}
	return source.responses[index], nil
}

func testPolicy() Policy {
	return Policy{Timeout: 100 * time.Millisecond, PollInterval: 10 * time.Millisecond, MaximumAttempts: 3}
}

func TestObserveFindsExactCorrelationAfterRetry(t *testing.T) {
	startedAt := time.Now().UTC()
	source := &sequenceSource{responses: [][]Event{
		{},
		{{
			Provider:       "Microsoft-Windows-Sysmon",
			EventID:        1,
			RecordID:       42,
			TimeCreatedUTC: startedAt.Add(time.Millisecond),
			DataValues:     []string{"cmd.exe /c echo " + testCorrelationID + " >NUL"},
		}},
	}}
	observer, err := NewSysmonObserver(source, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := observer.Observe(context.Background(), testCorrelationID, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.RecordID != 42 || evidence.Attempts != 2 {
		t.Fatalf("evidence = %+v", evidence)
	}
}

func TestObserveReturnsNotFoundAfterBoundedAttempts(t *testing.T) {
	source := &sequenceSource{responses: [][]Event{{}}}
	observer, err := NewSysmonObserver(source, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	_, err = observer.Observe(context.Background(), testCorrelationID, time.Now().UTC())
	if !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("error = %v", err)
	}
	if source.calls != 3 {
		t.Fatalf("query attempts = %d, want 3", source.calls)
	}
}

func TestObserveRejectsInvalidCorrelationBeforeQuery(t *testing.T) {
	source := &sequenceSource{}
	observer, err := NewSysmonObserver(source, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	_, err = observer.Observe(context.Background(), "invalid", time.Now().UTC())
	if !errors.Is(err, ErrObservationFailed) {
		t.Fatalf("error = %v", err)
	}
	if source.calls != 0 {
		t.Fatal("source queried for invalid correlation ID")
	}
}

func TestObserveDoesNotRetrySourceFailure(t *testing.T) {
	source := &sequenceSource{err: errors.New("access denied")}
	observer, err := NewSysmonObserver(source, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	_, err = observer.Observe(context.Background(), testCorrelationID, time.Now().UTC())
	if !errors.Is(err, ErrObservationFailed) {
		t.Fatalf("error = %v", err)
	}
	if source.calls != 1 {
		t.Fatalf("query attempts = %d, want 1", source.calls)
	}
}

func TestPolicyRejectsRelaxation(t *testing.T) {
	tests := []Policy{
		{Timeout: 16 * time.Second, PollInterval: 500 * time.Millisecond, MaximumAttempts: 30},
		{Timeout: 15 * time.Second, PollInterval: 2 * time.Second, MaximumAttempts: 30},
		{Timeout: 15 * time.Second, PollInterval: 500 * time.Millisecond, MaximumAttempts: 31},
	}
	for index, policy := range tests {
		if err := policy.Validate(); !errors.Is(err, ErrInvalidPolicy) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}
