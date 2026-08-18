package controlplane

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
)

const (
	testRunnerID      = "6ba7b812-9dad-41d1-80b4-00c04fd430c8"
	testEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	testHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
	testOperatorID    = "7ba7b811-9dad-41d1-80b4-00c04fd430c8"
	testJobID         = "8ba7b811-9dad-41d1-80b4-00c04fd430c8"
	testToken         = "runner_token_0123456789ABCDEF0123456789ABCDEF"
	testKeyID         = "local-poc-key-01"
)

var testNow = time.Date(2026, time.August, 18, 18, 0, 0, 0, time.UTC)

func testIdentity() identity.RunnerIdentity {
	return identity.RunnerIdentity{
		SchemaVersion: identity.SchemaVersion,
		RunnerID:      testRunnerID,
		EnvironmentID: testEnvironmentID,
		HostID:        testHostID,
		IdentityKeyID: "runner-identity-01",
		RegisteredAt:  testNow.Add(-time.Hour),
		State:         identity.StateActive,
	}
}

func signedTestJob(privateKey ed25519.PrivateKey) Job {
	job := Job{
		SchemaVersion:   SchemaVersion,
		JobID:           testJobID,
		CorrelationID:   "PL-0123456789ABCDEF0123456789ABCDEF",
		EnvironmentID:   testEnvironmentID,
		HostID:          testHostID,
		ScenarioID:      "windows-process-marker",
		ScenarioVersion: "0.1.0",
		RequestedBy:     testOperatorID,
		RequestedAt:     testNow.Add(-time.Second),
		ExpiresAt:       testNow.Add(time.Minute),
		Nonce:           "0123456789ABCDEF0123456789ABCDEF",
		Parameters:      map[string]any{},
	}
	payload, err := json.Marshal(unsigned(job))
	if err != nil {
		panic(err)
	}
	job.Signature = Signature{
		Algorithm: "Ed25519",
		KeyID:     testKeyID,
		Value:     base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, payload)),
	}
	return job
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func handlerClient(handler http.Handler) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Result(), nil
	})}
}

func testClient(t *testing.T, handler http.Handler, publicKey ed25519.PublicKey) *Client {
	t.Helper()
	client, err := New(Config{
		BaseURL:          "http://127.0.0.1:8787",
		BearerToken:      testToken,
		Identity:         testIdentity(),
		SigningKeyID:     testKeyID,
		SigningPublicKey: publicKey,
		HTTPClient:       handlerClient(handler),
		Now:              func() time.Time { return testNow },
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func writeTestJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func TestClientLeasesAndVerifiesBoundSignedJob(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	job := signedTestJob(privateKey)
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/runners/"+testRunnerID+"/jobs:lease" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer "+testToken {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		writeTestJSON(writer, http.StatusOK, job)
	})

	client := testClient(t, handler, publicKey)
	leased, ok, err := client.Lease(context.Background())
	if err != nil || !ok {
		t.Fatalf("lease = %+v, ok = %v, error = %v", leased, ok, err)
	}
	request := leased.ExecutionRequest()
	if request.CorrelationID != job.CorrelationID || request.ScenarioID != job.ScenarioID || len(request.Parameters) != 0 {
		t.Fatalf("execution request = %+v", request)
	}
	if _, _, err := client.Lease(context.Background()); !errors.Is(err, ErrJobReplayed) {
		t.Fatalf("replay error = %v", err)
	}
}

func TestClientRejectsTamperedAndMismatchedJobs(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*Job)
		want   error
	}{
		{name: "tampered scenario", mutate: func(job *Job) { job.ScenarioID = "windows-registry-run-key-canary" }, want: ErrJobSignature},
		{name: "wrong host", mutate: func(job *Job) { job.HostID = testOperatorID }, want: ErrJobIdentity},
		{name: "expired", mutate: func(job *Job) { job.ExpiresAt = testNow }, want: ErrJobExpired},
	} {
		t.Run(test.name, func(t *testing.T) {
			job := signedTestJob(privateKey)
			test.mutate(&job)
			handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writeTestJSON(writer, http.StatusOK, job)
			})
			client := testClient(t, handler, publicKey)
			if _, _, err := client.Lease(context.Background()); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestClientSendsOnlyBoundLifecycleRequests(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var requests []string
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+testToken {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		mu.Lock()
		requests = append(requests, request.Method+" "+request.URL.Path)
		mu.Unlock()
		writeTestJSON(writer, http.StatusOK, map[string]string{"schema_version": SchemaVersion})
	})
	client := testClient(t, handler, publicKey)

	ctx := context.Background()
	if err := client.Acknowledge(ctx, testJobID, true); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateStage(ctx, testJobID, StageUpdate{Stage: "execution", Status: "passed", LatencyMS: 21}); err != nil {
		t.Fatal(err)
	}
	if err := client.Complete(ctx, testJobID); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"POST /v1/runners/" + testRunnerID + "/jobs/" + testJobID + ":ack",
		"PUT /v1/runners/" + testRunnerID + "/jobs/" + testJobID + "/stages/execution",
		"POST /v1/runners/" + testRunnerID + "/jobs/" + testJobID + ":complete",
	}
	if strings.Join(requests, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
	if err := client.UpdateStage(ctx, testJobID, StageUpdate{Stage: "execution", Status: "failed", DetailCode: "arbitrary_text"}); !errors.Is(err, ErrInvalidUpdate) {
		t.Fatalf("unsafe detail error = %v", err)
	}
}

func TestClientReportsStableRemoteError(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeTestJSON(writer, http.StatusConflict, map[string]string{
			"schema_version": SchemaVersion,
			"code":           "JOB_TRANSITION_REJECTED",
		})
	})
	client := testClient(t, handler, publicKey)
	err = client.Complete(context.Background(), testJobID)
	var remote *RemoteError
	if !errors.As(err, &remote) || remote.StatusCode != http.StatusConflict || remote.Code != "JOB_TRANSITION_REJECTED" {
		t.Fatalf("remote error = %#v", err)
	}
}

func TestClientDisablesRedirectsThatCouldMoveRunnerCredentials(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", "https://other.example/v1/runners")
		writer.WriteHeader(http.StatusTemporaryRedirect)
	})
	client := testClient(t, handler, publicKey)
	err = client.Complete(context.Background(), testJobID)
	var remote *RemoteError
	if !errors.As(err, &remote) || remote.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("redirect error = %#v", err)
	}
}

func TestClientConfigurationRejectsInsecureOrUnboundedSettings(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	base := Config{
		BaseURL:          "http://127.0.0.1:8787",
		BearerToken:      testToken,
		Identity:         testIdentity(),
		SigningKeyID:     testKeyID,
		SigningPublicKey: publicKey,
		Now:              func() time.Time { return testNow },
	}
	for _, mutate := range []func(*Config){
		func(config *Config) { config.BaseURL = "http://10.0.2.2:8787" },
		func(config *Config) { config.BaseURL = "file:///tmp/control-plane" },
		func(config *Config) { config.BearerToken = "short" },
		func(config *Config) { config.Identity.State = identity.StateRevoked },
		func(config *Config) { config.Identity.HostID = strings.ToUpper(config.Identity.HostID) },
		func(config *Config) { config.SigningPublicKey = nil },
	} {
		config := base
		mutate(&config)
		if _, err := New(config); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("configuration error = %v for %+v", err, config)
		}
	}
}
