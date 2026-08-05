package splunk

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidPollingPolicy = errors.New("invalid Splunk polling policy")

type ExactSearcher interface {
	SearchExact(context.Context, string, SearchWindow) (CorrelationEvidence, error)
}

type PollingPolicy struct {
	Timeout         time.Duration
	Interval        time.Duration
	MaximumAttempts int
}

func ApprovedPollingPolicy() PollingPolicy {
	return PollingPolicy{Timeout: 60 * time.Second, Interval: 2 * time.Second, MaximumAttempts: 30}
}

func (policy PollingPolicy) Validate() error {
	if policy.Timeout <= 0 || policy.Timeout > time.Minute {
		return ErrInvalidPollingPolicy
	}
	if policy.Interval < 250*time.Millisecond || policy.Interval > 5*time.Second {
		return ErrInvalidPollingPolicy
	}
	if policy.MaximumAttempts < 1 || policy.MaximumAttempts > 120 {
		return ErrInvalidPollingPolicy
	}
	return nil
}

func PollExact(
	ctx context.Context,
	searcher ExactSearcher,
	correlationID string,
	window SearchWindow,
	policy PollingPolicy,
) (CorrelationEvidence, int, error) {
	if searcher == nil {
		return CorrelationEvidence{}, 0, ErrInvalidPollingPolicy
	}
	if err := policy.Validate(); err != nil {
		return CorrelationEvidence{}, 0, err
	}
	pollContext, cancel := context.WithTimeout(ctx, policy.Timeout)
	defer cancel()
	for attempt := 1; attempt <= policy.MaximumAttempts; attempt++ {
		evidence, err := searcher.SearchExact(pollContext, correlationID, window)
		if err == nil {
			return evidence, attempt, nil
		}
		if !errors.Is(err, ErrEventNotFound) {
			return CorrelationEvidence{}, attempt, err
		}
		if attempt == policy.MaximumAttempts {
			break
		}
		timer := time.NewTimer(policy.Interval)
		select {
		case <-pollContext.Done():
			timer.Stop()
			return CorrelationEvidence{}, attempt, ErrEventNotFound
		case <-timer.C:
		}
	}
	return CorrelationEvidence{}, policy.MaximumAttempts, ErrEventNotFound
}
