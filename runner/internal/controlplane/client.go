package controlplane

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

const maximumResponseBody = 64 << 10

var (
	ErrInvalidConfig = errors.New("invalid Control Plane client configuration")
	ErrInvalidUpdate = errors.New("invalid lifecycle stage update")
	ErrRemote        = errors.New("Control Plane rejected the Runner request")

	bearerTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)
	detailPolicies     = map[string]struct {
		stage  string
		status string
	}{
		"awaiting_endpoint_event":     {stage: "endpoint_telemetry", status: "running"},
		"endpoint_event_delayed":      {stage: "endpoint_telemetry", status: "running"},
		"endpoint_event_missing":      {stage: "endpoint_telemetry", status: "failed"},
		"siem_ingestion_delayed":      {stage: "siem_ingestion", status: "running"},
		"siem_event_missing":          {stage: "siem_ingestion", status: "failed"},
		"required_field_missing":      {stage: "field_validation", status: "failed"},
		"detection_result_absent":     {stage: "detection", status: "failed"},
		"alert_delivery_delayed":      {stage: "alert", status: "running"},
		"cleanup_verification_failed": {stage: "cleanup", status: "failed"},
	}
)

type Config struct {
	BaseURL          string
	BearerToken      string
	Identity         identity.RunnerIdentity
	SigningKeyID     string
	SigningPublicKey ed25519.PublicKey
	HTTPClient       *http.Client
	Now              func() time.Time
}

type Client struct {
	baseURL     string
	token       string
	identity    identity.RunnerIdentity
	httpClient  *http.Client
	verifier    *Verifier
	now         func() time.Time
	stageNames  map[string]bool
	stageStatus map[string]bool
}

type StageUpdate struct {
	Stage      string `json:"-"`
	Status     string `json:"status"`
	LatencyMS  int64  `json:"latency_ms"`
	DetailCode string `json:"detail_code,omitempty"`
}

type RemoteError struct {
	StatusCode int
	Code       string
}

func (err *RemoteError) Error() string {
	return fmt.Sprintf("%s: HTTP %d (%s)", ErrRemote, err.StatusCode, err.Code)
}

func (err *RemoteError) Unwrap() error {
	return ErrRemote
}

func New(config Config) (*Client, error) {
	now := config.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	parsed, err := url.Parse(config.BaseURL)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.Path != "" || parsed.Port() == "" || !validTransport(parsed) ||
		!bearerTokenPattern.MatchString(config.BearerToken) || config.Identity.Validate(now()) != nil ||
		config.Identity.State != identity.StateActive || !canonicalIdentity(config.Identity) {
		return nil, ErrInvalidConfig
	}
	verifier, err := NewVerifier(config.Identity, scenario.BuiltInCatalog(), config.SigningKeyID, config.SigningPublicKey)
	if err != nil {
		return nil, ErrInvalidConfig
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	} else {
		clone := *httpClient
		if clone.Timeout <= 0 || clone.Timeout > 30*time.Second {
			clone.Timeout = 10 * time.Second
		}
		httpClient = &clone
	}
	httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{
		baseURL:    strings.TrimSuffix(config.BaseURL, "/"),
		token:      config.BearerToken,
		identity:   config.Identity,
		httpClient: httpClient,
		verifier:   verifier,
		now:        now,
		stageNames: map[string]bool{
			"execution": true, "endpoint_telemetry": true, "siem_ingestion": true,
			"field_validation": true, "detection": true, "alert": true, "cleanup": true,
		},
		stageStatus: map[string]bool{"running": true, "passed": true, "failed": true, "not_tested": true},
	}, nil
}

func (client *Client) Lease(ctx context.Context) (Job, bool, error) {
	var job Job
	status, err := client.doJSON(ctx, http.MethodPost, client.runnerPath("jobs:lease"), versionDocument(), &job)
	if err != nil {
		return Job{}, false, err
	}
	if status == http.StatusNoContent {
		return Job{}, false, nil
	}
	if status != http.StatusOK {
		return Job{}, false, &RemoteError{StatusCode: status, Code: "UNEXPECTED_RESPONSE"}
	}
	if err := client.verifier.VerifyAndConsume(job, client.now()); err != nil {
		return Job{}, false, err
	}
	return job, true, nil
}

func (client *Client) Acknowledge(ctx context.Context, jobID string, accepted bool) error {
	document := struct {
		SchemaVersion string `json:"schema_version"`
		Accepted      bool   `json:"accepted"`
	}{SchemaVersion: SchemaVersion, Accepted: accepted}
	status, err := client.doJSON(ctx, http.MethodPost, client.jobPath(jobID)+":ack", document, nil)
	return requireOK(status, err)
}

func (client *Client) UpdateStage(ctx context.Context, jobID string, update StageUpdate) error {
	if !client.stageNames[update.Stage] || !client.stageStatus[update.Status] || update.LatencyMS < 0 ||
		update.LatencyMS > 300_000 || !validDetail(update) {
		return ErrInvalidUpdate
	}
	document := struct {
		SchemaVersion string `json:"schema_version"`
		Status        string `json:"status"`
		LatencyMS     int64  `json:"latency_ms"`
		DetailCode    string `json:"detail_code,omitempty"`
	}{SchemaVersion: SchemaVersion, Status: update.Status, LatencyMS: update.LatencyMS, DetailCode: update.DetailCode}
	status, err := client.doJSON(ctx, http.MethodPut, client.jobPath(jobID)+"/stages/"+update.Stage, document, nil)
	return requireOK(status, err)
}

func (client *Client) Complete(ctx context.Context, jobID string) error {
	status, err := client.doJSON(ctx, http.MethodPost, client.jobPath(jobID)+":complete", versionDocument(), nil)
	return requireOK(status, err)
}

func (client *Client) runnerPath(suffix string) string {
	return "/v1/runners/" + client.identity.RunnerID + "/" + suffix
}

func (client *Client) jobPath(jobID string) string {
	return client.runnerPath("jobs/" + jobID)
}

func (client *Client) doJSON(
	ctx context.Context,
	method,
	path string,
	document,
	result any,
) (int, error) {
	payload, err := json.Marshal(document)
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.StatusCode, decodeRemoteError(response)
	}
	if response.StatusCode == http.StatusNoContent || result == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maximumResponseBody))
		return response.StatusCode, nil
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return response.StatusCode, errors.New("Control Plane response content type is invalid")
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maximumResponseBody+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(result); err != nil {
		return response.StatusCode, err
	}
	if err := ensureResponseEOF(decoder); err != nil {
		return response.StatusCode, err
	}
	return response.StatusCode, nil
}

func validTransport(target *url.URL) bool {
	if target.Scheme == "https" {
		return target.Hostname() != ""
	}
	if target.Scheme != "http" {
		return false
	}
	ip := net.ParseIP(target.Hostname())
	return ip != nil && ip.IsLoopback()
}

func canonicalIdentity(value identity.RunnerIdentity) bool {
	return value.RunnerID == strings.ToLower(value.RunnerID) &&
		value.EnvironmentID == strings.ToLower(value.EnvironmentID) &&
		value.HostID == strings.ToLower(value.HostID)
}

func validDetail(update StageUpdate) bool {
	if update.DetailCode == "" {
		return true
	}
	policy, ok := detailPolicies[update.DetailCode]
	return ok && policy.stage == update.Stage && policy.status == update.Status
}

func requireOK(status int, err error) error {
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return &RemoteError{StatusCode: status, Code: "UNEXPECTED_RESPONSE"}
	}
	return nil
}

func versionDocument() struct {
	SchemaVersion string `json:"schema_version"`
} {
	return struct {
		SchemaVersion string `json:"schema_version"`
	}{SchemaVersion: SchemaVersion}
}

func decodeRemoteError(response *http.Response) error {
	document := struct {
		SchemaVersion string `json:"schema_version"`
		Code          string `json:"code"`
	}{}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maximumResponseBody+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil || document.SchemaVersion != SchemaVersion || document.Code == "" {
		document.Code = "UNREADABLE_ERROR"
	}
	return &RemoteError{StatusCode: response.StatusCode, Code: document.Code}
}

func ensureResponseEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("Control Plane response contains more than one JSON value")
	}
	return nil
}
