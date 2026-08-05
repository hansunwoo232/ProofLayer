package splunk

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const searchCorrelationID = "PL-0123456789ABCDEF0123456789ABCDEF"

func TestSearchExactReturnsMinimumEvidence(t *testing.T) {
	window := SearchWindow{
		Earliest: time.Unix(1_754_416_800, 0).UTC(),
		Latest:   time.Unix(1_754_420_400, 0).UTC(),
	}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		search := form.Get("search")
		for _, required := range []string{
			"index=prooflayer_test",
			`source="prooflayer:windows-lab"`,
			"earliest=1754416800",
			"latest=1754420400",
			searchCorrelationID,
			"head 2",
			"table correlation_id,provider,event_id,record_id,endpoint_event_time,ingestion_latency_ms,host_name_present,process_name_present,process_command_line_present,user_name_present",
		} {
			if !strings.Contains(search, required) {
				t.Errorf("search missing %q: %s", required, search)
			}
		}
		payload := `{"preview":false,"result":{"correlation_id":"` + searchCorrelationID +
			`","provider":"Microsoft-Windows-Sysmon","event_id":"1","record_id":"2447","endpoint_event_time":"2026-08-05T18:41:41.3230000Z","ingestion_latency_ms":"2677","host_name_present":"1","process_name_present":"1","process_command_line_present":"1","user_name_present":"1"}}` + "\n"
		return response(http.StatusOK, payload), nil
	})}
	connector, err := New(Config{BaseURL: "https://splunk.test:8089", Username: ObserverUsername, Password: testPassword}, client)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := connector.SearchExact(context.Background(), searchCorrelationID, window)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.RecordID != 2447 || evidence.IngestionLatencyMS != 2677 {
		t.Fatalf("evidence = %+v", evidence)
	}
	if !evidence.FieldPresence["process.command_line"] {
		t.Fatal("process.command_line presence was not returned")
	}
}

func TestSearchExactClassifiesEmptyAndAmbiguousResults(t *testing.T) {
	window := SearchWindow{Earliest: time.Now().Add(-time.Hour), Latest: time.Now()}
	tests := []struct {
		name     string
		payload  string
		expected error
	}{
		{name: "empty", payload: "", expected: ErrEventNotFound},
		{name: "ambiguous", payload: validSearchRow(searchCorrelationID) + validSearchRow(searchCorrelationID), expected: ErrAmbiguousEvent},
		{name: "mismatch", payload: validSearchRow("PL-FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), expected: ErrInvalidSearchResult},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return response(http.StatusOK, test.payload), nil
			})}
			connector, err := New(Config{BaseURL: "https://splunk.test:8089", Username: ObserverUsername, Password: testPassword}, client)
			if err != nil {
				t.Fatal(err)
			}
			_, err = connector.SearchExact(context.Background(), searchCorrelationID, window)
			if !errors.Is(err, test.expected) {
				t.Fatalf("error = %v, want %v", err, test.expected)
			}
		})
	}
}

func TestSearchWindowRejectsUnboundedRange(t *testing.T) {
	now := time.Now().UTC()
	invalid := []SearchWindow{
		{},
		{Earliest: now, Latest: now},
		{Earliest: now, Latest: now.Add(24*time.Hour + time.Second)},
	}
	for index, window := range invalid {
		if err := window.Validate(); !errors.Is(err, ErrInvalidSearchWindow) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func validSearchRow(correlationID string) string {
	return `{"result":{"correlation_id":"` + correlationID +
		`","provider":"Microsoft-Windows-Sysmon","event_id":"1","record_id":"2447","endpoint_event_time":"2026-08-05T18:41:41.3230000Z","ingestion_latency_ms":"2677","host_name_present":"1","process_name_present":"1","process_command_line_present":"1","user_name_present":"1"}}` + "\n"
}

type delayedSearcher struct {
	missingAttempts int
	calls           int
}

func (searcher *delayedSearcher) SearchExact(context.Context, string, SearchWindow) (CorrelationEvidence, error) {
	searcher.calls++
	if searcher.calls <= searcher.missingAttempts {
		return CorrelationEvidence{}, ErrEventNotFound
	}
	return CorrelationEvidence{CorrelationID: searchCorrelationID, Provider: "Microsoft-Windows-Sysmon", EventID: 1}, nil
}

func TestPollExactFindsLateEvent(t *testing.T) {
	searcher := &delayedSearcher{missingAttempts: 2}
	policy := PollingPolicy{Timeout: time.Second, Interval: 250 * time.Millisecond, MaximumAttempts: 3}
	evidence, attempts, err := PollExact(
		context.Background(),
		searcher,
		searchCorrelationID,
		SearchWindow{Earliest: time.Now().Add(-time.Hour), Latest: time.Now()},
		policy,
	)
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 || evidence.CorrelationID != searchCorrelationID {
		t.Fatalf("attempts = %d, evidence = %+v", attempts, evidence)
	}
}

func TestPollExactStopsAfterBoundedMissingEvent(t *testing.T) {
	searcher := &delayedSearcher{missingAttempts: 10}
	policy := PollingPolicy{Timeout: time.Second, Interval: 250 * time.Millisecond, MaximumAttempts: 2}
	_, attempts, err := PollExact(
		context.Background(),
		searcher,
		searchCorrelationID,
		SearchWindow{Earliest: time.Now().Add(-time.Hour), Latest: time.Now()},
		policy,
	)
	if !errors.Is(err, ErrEventNotFound) || attempts != 2 {
		t.Fatalf("attempts = %d, error = %v", attempts, err)
	}
}
