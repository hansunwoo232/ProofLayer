package observer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
)

var (
	ErrInvalidPolicy     = errors.New("invalid observation policy")
	ErrEventNotFound     = errors.New("endpoint event not found before deadline")
	ErrObservationFailed = errors.New("endpoint observation failed")
)

type Policy struct {
	Timeout         time.Duration
	PollInterval    time.Duration
	MaximumAttempts int
}

func ApprovedSysmonPolicy() Policy {
	return Policy{
		Timeout:         15 * time.Second,
		PollInterval:    500 * time.Millisecond,
		MaximumAttempts: 30,
	}
}

func (policy Policy) Validate() error {
	if policy.Timeout <= 0 || policy.Timeout > 15*time.Second {
		return ErrInvalidPolicy
	}
	if policy.PollInterval < 10*time.Millisecond || policy.PollInterval > time.Second {
		return ErrInvalidPolicy
	}
	if policy.MaximumAttempts < 1 || policy.MaximumAttempts > 30 {
		return ErrInvalidPolicy
	}
	return nil
}

type Event struct {
	Provider       string
	EventID        int
	RecordID       uint64
	TimeCreatedUTC time.Time
	DataValues     []string
}

type Evidence struct {
	Provider       string    `json:"provider"`
	EventID        int       `json:"event_id"`
	RecordID       uint64    `json:"record_id"`
	TimeCreatedUTC time.Time `json:"time_created_utc"`
	Attempts       int       `json:"attempts"`
	ObservedAtUTC  time.Time `json:"observed_at_utc"`
}

type Source interface {
	RecentProcessEvents(context.Context, time.Time) ([]Event, error)
}

type SysmonObserver struct {
	source Source
	policy Policy
	now    func() time.Time
}

func NewSysmonObserver(source Source, policy Policy) (*SysmonObserver, error) {
	if source == nil {
		return nil, fmt.Errorf("%w: source", ErrInvalidPolicy)
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &SysmonObserver{
		source: source,
		policy: policy,
		now:    func() time.Time { return time.Now().UTC() },
	}, nil
}

func (observer *SysmonObserver) Observe(
	parent context.Context,
	correlationID string,
	executionStartedAt time.Time,
) (Evidence, error) {
	if !correlation.Valid(correlationID) {
		return Evidence{}, fmt.Errorf("%w: correlation_id", ErrObservationFailed)
	}
	if executionStartedAt.IsZero() {
		return Evidence{}, fmt.Errorf("%w: execution_started_at", ErrObservationFailed)
	}

	ctx, cancel := context.WithTimeout(parent, observer.policy.Timeout)
	defer cancel()
	for attempt := 1; attempt <= observer.policy.MaximumAttempts; attempt++ {
		events, err := observer.source.RecentProcessEvents(ctx, executionStartedAt.Add(-2*time.Second))
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return Evidence{}, ErrEventNotFound
			}
			return Evidence{}, fmt.Errorf("%w: %w", ErrObservationFailed, err)
		}
		for _, event := range events {
			if event.Provider != "Microsoft-Windows-Sysmon" || event.EventID != 1 {
				continue
			}
			if event.TimeCreatedUTC.Before(executionStartedAt.Add(-2 * time.Second)) {
				continue
			}
			if contains(event.DataValues, correlationID) {
				return Evidence{
					Provider:       event.Provider,
					EventID:        event.EventID,
					RecordID:       event.RecordID,
					TimeCreatedUTC: event.TimeCreatedUTC,
					Attempts:       attempt,
					ObservedAtUTC:  observer.now(),
				}, nil
			}
		}

		if attempt == observer.policy.MaximumAttempts {
			break
		}
		timer := time.NewTimer(observer.policy.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return Evidence{}, ErrEventNotFound
		case <-timer.C:
		}
	}
	return Evidence{}, ErrEventNotFound
}

func contains(values []string, correlationID string) bool {
	for _, value := range values {
		if value == correlationID {
			return true
		}
		if len(value) >= len(correlationID) {
			for index := 0; index+len(correlationID) <= len(value); index++ {
				if value[index:index+len(correlationID)] == correlationID {
					return true
				}
			}
		}
	}
	return false
}
