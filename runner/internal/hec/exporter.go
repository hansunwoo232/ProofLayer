package hec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
	"github.com/hansunwoo232/ProofLayer/runner/internal/observer"
)

const maximumResponseBytes = 16 << 10

var (
	ErrInvalidConfig = errors.New("invalid HEC exporter configuration")
	ErrInvalidEvent  = errors.New("invalid HEC event metadata")
	ErrRejected      = errors.New("HEC rejected the event")
	tokenPattern     = regexp.MustCompile(`^[A-Fa-f0-9]{32,128}$`)
)

type Config struct {
	Endpoint   string
	Token      string
	HTTPClient *http.Client
}

type Exporter struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

func New(config Config) (*Exporter, error) {
	target, err := url.Parse(config.Endpoint)
	if err != nil || target.Scheme != "https" || target.Host == "" ||
		target.Path != "/services/collector/event" || target.User != nil ||
		target.RawQuery != "" || target.Fragment != "" || !tokenPattern.MatchString(config.Token) {
		return nil, ErrInvalidConfig
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	} else {
		clone := *client
		if clone.Timeout <= 0 || clone.Timeout > 30*time.Second {
			clone.Timeout = 15 * time.Second
		}
		client = &clone
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Exporter{endpoint: target.String(), token: config.Token, httpClient: client}, nil
}

func (exporter *Exporter) Export(
	ctx context.Context,
	correlationID string,
	evidence observer.Evidence,
) error {
	if !correlation.Valid(correlationID) || evidence.Provider != "Microsoft-Windows-Sysmon" ||
		evidence.EventID != 1 || evidence.RecordID == 0 || evidence.TimeCreatedUTC.IsZero() {
		return ErrInvalidEvent
	}
	payload := envelope{
		Time:       float64(evidence.TimeCreatedUTC.UnixMilli()) / 1000,
		Host:       "prooflayer-windows-lab",
		Source:     "prooflayer:windows-lab",
		SourceType: "prooflayer:sysmon",
		Index:      "prooflayer_test",
		Event: event{
			SchemaVersion:     "1.0",
			CorrelationID:     correlationID,
			EventKind:         "endpoint_process",
			Provider:          evidence.Provider,
			EventID:           evidence.EventID,
			RecordID:          evidence.RecordID,
			EndpointEventTime: evidence.TimeCreatedUTC.UTC(),
			ObservedAt:        evidence.ObservedAtUTC.UTC(),
			Host:              namedValue{Name: "prooflayer-windows-lab"},
			Process: processValue{
				Name:        "cmd.exe",
				CommandLine: "prooflayer-safe-marker " + correlationID,
			},
			User: namedValue{Name: "prooflayer-synthetic-user"},
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ErrInvalidEvent
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, exporter.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return ErrInvalidConfig
	}
	request.Header.Set("Authorization", "Splunk "+exporter.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := exporter.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: unavailable", ErrRejected)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes+1))
	if err != nil || len(body) > maximumResponseBytes || response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: http_%d", ErrRejected, response.StatusCode)
	}
	var result struct {
		Text string `json:"text"`
		Code int    `json:"code"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil || result.Code != 0 || result.Text == "" {
		return ErrRejected
	}
	return nil
}

type envelope struct {
	Time       float64 `json:"time"`
	Host       string  `json:"host"`
	Source     string  `json:"source"`
	SourceType string  `json:"sourcetype"`
	Index      string  `json:"index"`
	Event      event   `json:"event"`
}

type event struct {
	SchemaVersion     string       `json:"schema_version"`
	CorrelationID     string       `json:"correlation_id"`
	EventKind         string       `json:"event_kind"`
	Provider          string       `json:"provider"`
	EventID           int          `json:"event_id"`
	RecordID          uint64       `json:"record_id"`
	EndpointEventTime time.Time    `json:"endpoint_event_time"`
	ObservedAt        time.Time    `json:"observed_at"`
	Host              namedValue   `json:"host"`
	Process           processValue `json:"process"`
	User              namedValue   `json:"user"`
}

type namedValue struct {
	Name string `json:"name"`
}

type processValue struct {
	Name        string `json:"name"`
	CommandLine string `json:"command_line"`
}
