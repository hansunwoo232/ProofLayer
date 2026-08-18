package httpapi

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/runqueue"
)

const (
	testOrigin        = "http://127.0.0.1:8787"
	testEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	testHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
	testRunnerID      = "6ba7b812-9dad-41d1-80b4-00c04fd430c8"
	testRunnerToken   = "runner_token_0123456789ABCDEF0123456789ABCDEF"
	testIdempotency   = "run_test_0123456789ABCDEF012345"
)

func newTestServer(t *testing.T) (*Server, *runqueue.Queue) {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	queue, err := runqueue.New(runqueue.Config{
		Capacity:          4,
		EnvironmentID:     testEnvironmentID,
		HostID:            testHostID,
		RequestedBy:       "7ba7b811-9dad-41d1-80b4-00c04fd430c8",
		SigningKeyID:      "local-poc-key-01",
		SigningPrivateKey: privateKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	dashboard := filepath.Join("..", "..", "..", "dashboard")
	server, err := New(queue, testOrigin, dashboard)
	if err != nil {
		t.Fatal(err)
	}
	server.csrfToken = func() (string, error) {
		return "csrf_0123456789ABCDEF0123456789ABCDEF01234567", nil
	}
	return server, queue
}

func validBody() string {
	return `{"schema_version":"1.0","environment_id":"` + testEnvironmentID + `","host_id":"` + testHostID + `","scenario_id":"windows-process-marker","scenario_version":"0.1.0"}`
}

func runnerServer(t *testing.T) (*Server, *runqueue.Queue) {
	t.Helper()
	base, queue := newTestServer(t)
	server, err := NewWithRunner(queue, testOrigin, filepath.Join("..", "..", "..", "dashboard"), RunnerBinding{
		RunnerID:      testRunnerID,
		EnvironmentID: testEnvironmentID,
		HostID:        testHostID,
		BearerToken:   testRunnerToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	server.csrfToken = base.csrfToken
	return server, queue
}

func callRunner(handler http.Handler, method, path, body, runnerID, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if runnerID != "" {
		request.URL.Path = strings.Replace(request.URL.Path, testRunnerID, runnerID, 1)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func session(t *testing.T, handler http.Handler) (*http.Cookie, string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/session", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session status = %d", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie = %+v", cookies)
	}
	var document map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	return cookies[0], document["csrf_token"]
}

func postJob(handler http.Handler, body, idempotency, origin string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/v1/test-jobs", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", origin)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Idempotency-Key", idempotency)
	request.Header.Set("X-ProofLayer-CSRF", csrf)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func getJobStatus(handler http.Handler, jobID string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/test-jobs/"+jobID, nil)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("X-ProofLayer-CSRF", csrf)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestSessionAndCreateJob(t *testing.T) {
	server, queue := newTestServer(t)
	handler := server.Handler()
	cookie, token := session(t, handler)

	response := postJob(handler, validBody(), testIdempotency, testOrigin, cookie, token)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body.String())
	}
	var receipt runqueue.Receipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "queued" || receipt.Replayed || queue.Depth() != 1 {
		t.Fatalf("receipt = %+v, depth = %d", receipt, queue.Depth())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers missing")
	}

	duplicate := postJob(handler, validBody(), testIdempotency, testOrigin, cookie, token)
	if duplicate.Code != http.StatusOK {
		t.Fatalf("duplicate status = %d", duplicate.Code)
	}
	if err := json.Unmarshal(duplicate.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if !receipt.Replayed || queue.Depth() != 1 {
		t.Fatalf("duplicate receipt = %+v, depth = %d", receipt, queue.Depth())
	}
}

func TestRequestIntegrityIsRequired(t *testing.T) {
	server, _ := newTestServer(t)
	handler := server.Handler()
	cookie, token := session(t, handler)
	tests := []struct {
		name   string
		origin string
		cookie *http.Cookie
		token  string
	}{
		{name: "missing origin", origin: "", cookie: cookie, token: token},
		{name: "foreign origin", origin: "https://attacker.example", cookie: cookie, token: token},
		{name: "missing cookie", origin: testOrigin, token: token},
		{name: "wrong token", origin: testOrigin, cookie: cookie, token: "csrf_WRONG_WRONG_WRONG_WRONG_WRONG_WRONG"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := postJob(handler, validBody(), testIdempotency, test.origin, test.cookie, test.token)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d", response.Code)
			}
		})
	}
}

func TestBodyAndContentValidation(t *testing.T) {
	server, queue := newTestServer(t)
	handler := server.Handler()
	cookie, token := session(t, handler)

	unknown := strings.TrimSuffix(validBody(), "}") + `,"command":"whoami"}`
	if response := postJob(handler, unknown, testIdempotency, testOrigin, cookie, token); response.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d", response.Code)
	}
	if queue.Depth() != 0 {
		t.Fatal("invalid request was queued")
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/test-jobs", bytes.NewBufferString(validBody()))
	request.Header.Set("Content-Type", "text/plain")
	request.Header.Set("Origin", testOrigin)
	request.Header.Set("X-ProofLayer-CSRF", token)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("content type status = %d", response.Code)
	}

	tooLarge := `{"padding":"` + strings.Repeat("x", maximumBody) + `"}`
	if response := postJob(handler, tooLarge, testIdempotency, testOrigin, cookie, token); response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large body status = %d", response.Code)
	}
}

func TestStaticWireframeIsServedWithSecurityHeaders(t *testing.T) {
	server, _ := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/result-screen-wireframe.html", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "ProofLayer") {
		t.Fatal("wireframe content missing")
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("content security policy missing")
	}
}

func TestJobStatusShowsLiveDelayedStage(t *testing.T) {
	server, queue := newTestServer(t)
	handler := server.Handler()
	cookie, token := session(t, handler)
	created := postJob(handler, validBody(), testIdempotency, testOrigin, cookie, token)
	var receipt runqueue.Receipt
	if err := json.Unmarshal(created.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if _, ok := queue.Lease(testEnvironmentID, testHostID); !ok {
		t.Fatal("job lease failed")
	}
	if err := queue.Acknowledge(testEnvironmentID, testHostID, receipt.JobID, true); err != nil {
		t.Fatal(err)
	}
	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, runqueue.StageUpdate{
		Stage: "execution", Status: runqueue.StageStatusPassed, LatencyMS: 74,
	}); err != nil {
		t.Fatal(err)
	}
	if err := queue.UpdateStage(testEnvironmentID, testHostID, receipt.JobID, runqueue.StageUpdate{
		Stage: "endpoint_telemetry", Status: runqueue.StageStatusRunning, DetailCode: "endpoint_event_delayed",
	}); err != nil {
		t.Fatal(err)
	}

	response := getJobStatus(handler, receipt.JobID, cookie, token)
	if response.Code != http.StatusOK {
		t.Fatalf("status response = %d, body = %s", response.Code, response.Body.String())
	}
	var snapshot runqueue.StatusSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != runqueue.JobStatusRunning || snapshot.Terminal {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if snapshot.Stages[0].Status != runqueue.StageStatusPassed ||
		snapshot.Stages[1].Status != runqueue.StageStatusRunning ||
		snapshot.Stages[1].DetailCode != "endpoint_event_delayed" {
		t.Fatalf("stages = %+v", snapshot.Stages[:2])
	}

	if response := getJobStatus(handler, receipt.JobID, cookie, "wrong-token-with-more-than-thirty-two-characters"); response.Code != http.StatusForbidden {
		t.Fatalf("invalid status session = %d", response.Code)
	}
	if response := getJobStatus(handler, "00000000-0000-4000-8000-000000000000", cookie, token); response.Code != http.StatusNotFound {
		t.Fatalf("unknown job status = %d", response.Code)
	}
}

func TestServerRejectsAmbiguousLoopbackOrigin(t *testing.T) {
	_, queue := newTestServer(t)
	for _, origin := range []string{
		"https://127.0.0.1:8787",
		"http://localhost:8787",
		"http://127.0.0.1:8787/path",
		"http://127.0.0.1",
	} {
		if _, err := New(queue, origin, filepath.Join("..", "..", "..", "dashboard")); err == nil {
			t.Fatalf("ambiguous origin %q was accepted", origin)
		}
	}
}

func TestRunnerTransportRequiresBoundCredentialBeforeLease(t *testing.T) {
	server, queue := runnerServer(t)
	handler := server.Handler()
	if _, err := queue.Enqueue(testIdempotency, runqueue.CreateRequest{
		SchemaVersion:   runqueue.SchemaVersion,
		EnvironmentID:   testEnvironmentID,
		HostID:          testHostID,
		ScenarioID:      "windows-process-marker",
		ScenarioVersion: "0.1.0",
	}); err != nil {
		t.Fatal(err)
	}

	path := "/v1/runners/" + testRunnerID + "/jobs:lease"
	for _, test := range []struct {
		name     string
		runnerID string
		token    string
	}{
		{name: "missing token", runnerID: testRunnerID},
		{name: "wrong token", runnerID: testRunnerID, token: testRunnerToken + "x"},
		{name: "wrong runner", runnerID: testHostID, token: testRunnerToken},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := callRunner(handler, http.MethodPost, path, `{"schema_version":"1.0"}`, test.runnerID, test.token)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if response.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("bearer challenge missing")
			}
		})
	}
	if queue.Depth() != 1 {
		t.Fatalf("unauthorized request changed queue depth to %d", queue.Depth())
	}

	response := callRunner(handler, http.MethodPost, path, `{"schema_version":"1.0"}`, testRunnerID, testRunnerToken)
	if response.Code != http.StatusOK {
		t.Fatalf("lease status = %d, body = %s", response.Code, response.Body.String())
	}
	var job runqueue.Job
	if err := json.Unmarshal(response.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if job.EnvironmentID != testEnvironmentID || job.HostID != testHostID || job.Signature.Value == "" {
		t.Fatalf("leased job = %+v", job)
	}
	if queue.Depth() != 0 {
		t.Fatalf("queue depth = %d", queue.Depth())
	}

	empty := callRunner(handler, http.MethodPost, path, `{"schema_version":"1.0"}`, testRunnerID, testRunnerToken)
	if empty.Code != http.StatusNoContent || empty.Body.Len() != 0 {
		t.Fatalf("empty lease = %d, body = %q", empty.Code, empty.Body.String())
	}
}

func TestRunnerTransportDrivesOrderedLifecycle(t *testing.T) {
	server, queue := runnerServer(t)
	handler := server.Handler()
	receipt, err := queue.Enqueue(testIdempotency, runqueue.CreateRequest{
		SchemaVersion:   runqueue.SchemaVersion,
		EnvironmentID:   testEnvironmentID,
		HostID:          testHostID,
		ScenarioID:      "windows-process-marker",
		ScenarioVersion: "0.1.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	leasePath := "/v1/runners/" + testRunnerID + "/jobs:lease"
	if response := callRunner(handler, http.MethodPost, leasePath, `{"schema_version":"1.0"}`, testRunnerID, testRunnerToken); response.Code != http.StatusOK {
		t.Fatalf("lease status = %d", response.Code)
	}

	jobPath := "/v1/runners/" + testRunnerID + "/jobs/" + receipt.JobID
	ack := callRunner(handler, http.MethodPost, jobPath+":ack", `{"schema_version":"1.0","accepted":true}`, testRunnerID, testRunnerToken)
	if ack.Code != http.StatusOK {
		t.Fatalf("ack status = %d, body = %s", ack.Code, ack.Body.String())
	}
	for _, stage := range []string{
		"execution", "endpoint_telemetry", "siem_ingestion", "field_validation", "detection", "alert", "cleanup",
	} {
		body := `{"schema_version":"1.0","status":"passed","latency_ms":1}`
		response := callRunner(handler, http.MethodPut, jobPath+"/stages/"+stage, body, testRunnerID, testRunnerToken)
		if response.Code != http.StatusOK {
			t.Fatalf("stage %s status = %d, body = %s", stage, response.Code, response.Body.String())
		}
	}
	complete := callRunner(handler, http.MethodPost, jobPath+":complete", `{"schema_version":"1.0"}`, testRunnerID, testRunnerToken)
	if complete.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body = %s", complete.Code, complete.Body.String())
	}
	snapshot, ok := queue.Status(receipt.JobID)
	if !ok || snapshot.Status != runqueue.JobStatusCompleted || !snapshot.Terminal {
		t.Fatalf("snapshot = %+v, exists = %v", snapshot, ok)
	}
}

func TestRunnerTransportRejectsMalformedAndOutOfOrderUpdates(t *testing.T) {
	server, queue := runnerServer(t)
	handler := server.Handler()
	receipt, err := queue.Enqueue(testIdempotency, runqueue.CreateRequest{
		SchemaVersion:   runqueue.SchemaVersion,
		EnvironmentID:   testEnvironmentID,
		HostID:          testHostID,
		ScenarioID:      "windows-process-marker",
		ScenarioVersion: "0.1.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	leasePath := "/v1/runners/" + testRunnerID + "/jobs:lease"
	if response := callRunner(handler, http.MethodPost, leasePath, `{"schema_version":"1.0","unknown":true}`, testRunnerID, testRunnerToken); response.Code != http.StatusBadRequest {
		t.Fatalf("unknown lease field status = %d, body = %s", response.Code, response.Body.String())
	}
	if queue.Depth() != 1 {
		t.Fatal("malformed lease consumed a job")
	}
	if response := callRunner(handler, http.MethodPost, leasePath, `{"schema_version":"1.0"}`, testRunnerID, testRunnerToken); response.Code != http.StatusOK {
		t.Fatalf("valid lease status = %d", response.Code)
	}
	jobPath := "/v1/runners/" + testRunnerID + "/jobs/" + receipt.JobID
	update := callRunner(handler, http.MethodPut, jobPath+"/stages/execution", `{"schema_version":"1.0","status":"passed","latency_ms":1}`, testRunnerID, testRunnerToken)
	if update.Code != http.StatusConflict {
		t.Fatalf("pre-ack update status = %d, body = %s", update.Code, update.Body.String())
	}
	wrongMethod := callRunner(handler, http.MethodGet, jobPath+":ack", `{"schema_version":"1.0","accepted":true}`, testRunnerID, testRunnerToken)
	if wrongMethod.Code != http.StatusMethodNotAllowed || wrongMethod.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("wrong method response = %d, allow = %q", wrongMethod.Code, wrongMethod.Header().Get("Allow"))
	}
}

func TestRunnerBindingValidationFailsClosed(t *testing.T) {
	_, queue := newTestServer(t)
	dashboard := filepath.Join("..", "..", "..", "dashboard")
	for _, binding := range []RunnerBinding{
		{RunnerID: "invalid", EnvironmentID: testEnvironmentID, HostID: testHostID, BearerToken: testRunnerToken},
		{RunnerID: testRunnerID, EnvironmentID: testEnvironmentID, HostID: testHostID, BearerToken: "short"},
	} {
		if _, err := NewWithRunner(queue, testOrigin, dashboard, binding); err == nil {
			t.Fatalf("invalid binding was accepted: %+v", binding)
		}
	}
}
