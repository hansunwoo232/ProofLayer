package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileRecorderAppendsOneJSONEventPerLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner-audit.jsonl")
	recorder, err := NewFileRecorder(path)
	if err != nil {
		t.Fatal(err)
	}
	event := Event{
		SchemaVersion: "1.0",
		Timestamp:     time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC),
		EventType:     "job.validation",
		Outcome:       "passed",
		RunnerID:      "550e8400-e29b-41d4-a716-446655440000",
		CorrelationID: "PL-0123456789ABCDEF0123456789ABCDEF",
	}
	if err := recorder.Append(event); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("missing audit line")
	}
	var decoded Event
	if err := json.Unmarshal(scanner.Bytes(), &decoded); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if decoded.CorrelationID != event.CorrelationID {
		t.Fatalf("correlation ID = %q", decoded.CorrelationID)
	}
	if scanner.Scan() {
		t.Fatal("unexpected second audit line")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("audit permissions = %o", info.Mode().Perm())
	}
}

func TestFileRecorderRejectsInvalidEventAndRelativePath(t *testing.T) {
	if _, err := NewFileRecorder("relative.jsonl"); err == nil {
		t.Fatal("relative path accepted")
	}
	recorder, err := NewFileRecorder(filepath.Join(t.TempDir(), "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Append(Event{}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("invalid event error = %v", err)
	}
}
