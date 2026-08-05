package audit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
)

var (
	ErrInvalidEvent  = errors.New("invalid audit event")
	eventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_.]{2,79}$`)
)

type Event struct {
	SchemaVersion string    `json:"schema_version"`
	Timestamp     time.Time `json:"timestamp"`
	EventType     string    `json:"event_type"`
	Outcome       string    `json:"outcome"`
	RunnerID      string    `json:"runner_id"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	ErrorCode     string    `json:"error_code,omitempty"`
}

func (event Event) Validate() error {
	if event.SchemaVersion != "1.0" || event.Timestamp.IsZero() {
		return ErrInvalidEvent
	}
	if !eventTypePattern.MatchString(event.EventType) {
		return ErrInvalidEvent
	}
	if event.Outcome != "passed" && event.Outcome != "failed" && event.Outcome != "denied" {
		return ErrInvalidEvent
	}
	if event.RunnerID == "" {
		return ErrInvalidEvent
	}
	if event.CorrelationID != "" && !correlation.Valid(event.CorrelationID) {
		return ErrInvalidEvent
	}
	if len(event.ErrorCode) > 80 {
		return ErrInvalidEvent
	}
	return nil
}

type FileRecorder struct {
	path string
}

type Recorder interface {
	Append(Event) error
}

func NewFileRecorder(path string) (*FileRecorder, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("audit path must be absolute")
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("audit path must be a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect audit path: %w", err)
	}
	return &FileRecorder{path: path}, nil
}

func (recorder *FileRecorder) Append(event Event) error {
	if err := event.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode audit event: %w", err)
	}
	payload = append(payload, '\n')

	file, err := os.OpenFile(recorder.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("secure audit file: %w", err)
	}
	if _, err := file.Write(payload); err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync audit event: %w", err)
	}
	return nil
}
