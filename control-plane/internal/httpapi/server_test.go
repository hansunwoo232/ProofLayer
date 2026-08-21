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
	"time"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/localauth"
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

func localAuthServer(t *testing.T) (*Server, *runqueue.Queue) {
	t.Helper()
	_, queue := newTestServer(t)
	authentication, err := localauth.New(localauth.Config{
		Workspace: localauth.Workspace{
			ID:   "8ba7b810-9dad-41d1-80b4-00c04fd430c8",
			Slug: "prooflayer-lab",
			Name: "ProofLayer Lab",
		},
		User: localauth.User{
			ID:          "7ba7b811-9dad-41d1-80b4-00c04fd430c8",
			WorkspaceID: "8ba7b810-9dad-41d1-80b4-00c04fd430c8",
			Email:       "admin@prooflayer.local",
			DisplayName: "Local Administrator",
			Role:        localauth.RoleAdmin,
			Status:      localauth.StatusActive,
		},
		Password: "correct horse battery staple",
		PasswordParameters: localauth.PasswordParameters{
			MemoryKiB: 8 * 1024, Iterations: 1, Parallelism: 1, SaltBytes: 16, KeyBytes: 32,
		},
		IdleTimeout:     30 * time.Minute,
		AbsoluteTimeout: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewWithLocalAuth(queue, testOrigin, filepath.Join("..", "..", "..", "dashboard"), authentication)
	if err != nil {
		t.Fatal(err)
	}
	sequence := 0
	server.csrfToken = func() (string, error) {
		sequence++
		return "csrf_0123456789ABCDEF0123456789ABCDEF_" + string(rune('0'+sequence)), nil
	}
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
	var document struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	return cookies[0], document.CSRFToken
}

func login(
	handler http.Handler,
	email,
	password string,
	csrfCookie *http.Cookie,
	csrfToken string,
) *httptest.ResponseRecorder {
	body := `{"schema_version":"1.0","email":` + string(mustJSON(email)) + `,"password":` + string(mustJSON(password)) + `}`
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", testOrigin)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("X-ProofLayer-CSRF", csrfToken)
	request.AddCookie(csrfCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func mustJSON(value string) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func cookieNamed(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s was not set", name)
	return nil
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

func getWorkspaceAPI(handler http.Handler, path string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("X-ProofLayer-CSRF", csrf)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func postWorkspaceAPI(handler http.Handler, path, body string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", testOrigin)
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("X-ProofLayer-CSRF", csrf)
	request.AddCookie(cookie)
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
	for _, path := range []string{
		"/result-screen-wireframe.html", "/test-new.html", "/hosts.html", "/schedules.html", "/history.html", "/app.css", "/app.js",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, response.Code)
		}
		if response.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("%s content security policy missing", path)
		}
	}
}

func TestScenarioHostScheduleAndHistoryAPIs(t *testing.T) {
	server, queue := runnerServer(t)
	handler := server.Handler()
	csrfCookie, csrfToken := session(t, handler)

	scenarios := getWorkspaceAPI(handler, "/v1/scenarios", csrfCookie, csrfToken)
	if scenarios.Code != http.StatusOK || !strings.Contains(scenarios.Body.String(), `"risk_level":"guarded"`) ||
		!strings.Contains(scenarios.Body.String(), `"cleanup_required":true`) {
		t.Fatalf("scenarios = %d, body = %s", scenarios.Code, scenarios.Body.String())
	}
	hosts := getWorkspaceAPI(handler, "/v1/hosts", csrfCookie, csrfToken)
	if hosts.Code != http.StatusOK || !strings.Contains(hosts.Body.String(), `"status":"offline"`) {
		t.Fatalf("initial hosts = %d, body = %s", hosts.Code, hosts.Body.String())
	}
	leasePath := "/v1/runners/" + testRunnerID + "/jobs:lease"
	request := httptest.NewRequest(http.MethodPost, leasePath, strings.NewReader(`{"schema_version":"1.0"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testRunnerToken)
	request.Header.Set("X-ProofLayer-Runner-Version", "0.3.0")
	heartbeat := httptest.NewRecorder()
	handler.ServeHTTP(heartbeat, request)
	if heartbeat.Code != http.StatusNoContent {
		t.Fatalf("heartbeat = %d, body = %s", heartbeat.Code, heartbeat.Body.String())
	}
	hosts = getWorkspaceAPI(handler, "/v1/hosts", csrfCookie, csrfToken)
	if !strings.Contains(hosts.Body.String(), `"status":"online"`) ||
		!strings.Contains(hosts.Body.String(), `"runner_version":"0.3.0"`) ||
		!strings.Contains(hosts.Body.String(), `"last_seen_at"`) {
		t.Fatalf("online hosts body = %s", hosts.Body.String())
	}

	invalidHostBody := strings.Replace(validBody(), testHostID, "00000000-0000-4000-8000-000000000000", 1)
	if response := postJob(handler, invalidHostBody, "invalid_host_0123456789ABCDEF012345", testOrigin, csrfCookie, csrfToken); response.Code != http.StatusForbidden {
		t.Fatalf("invalid host status = %d, body = %s", response.Code, response.Body.String())
	}
	if queue.Depth() != 0 {
		t.Fatal("unauthorized host request was queued")
	}
	if response := postJob(handler, validBody(), testIdempotency, testOrigin, csrfCookie, csrfToken); response.Code != http.StatusCreated {
		t.Fatalf("valid create status = %d, body = %s", response.Code, response.Body.String())
	}
	registryBody := strings.Replace(validBody(), "windows-process-marker", "windows-registry-run-key-canary", 1)
	if response := postJob(handler, registryBody, "history_second_0123456789ABCDEF0123", testOrigin, csrfCookie, csrfToken); response.Code != http.StatusCreated {
		t.Fatalf("second create status = %d, body = %s", response.Code, response.Body.String())
	}

	history := getWorkspaceAPI(handler, "/v1/test-runs?page=1&page_size=1&scenario_id=windows-registry-run-key-canary", csrfCookie, csrfToken)
	if history.Code != http.StatusOK || !strings.Contains(history.Body.String(), `"total_items":1`) ||
		!strings.Contains(history.Body.String(), `"windows-registry-run-key-canary"`) {
		t.Fatalf("history = %d, body = %s", history.Code, history.Body.String())
	}
	invalidHistory := getWorkspaceAPI(handler, "/v1/test-runs?page=-1", csrfCookie, csrfToken)
	if invalidHistory.Code != http.StatusBadRequest {
		t.Fatalf("invalid history status = %d", invalidHistory.Code)
	}

	location, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	scheduledLocal := time.Now().In(location).Add(2 * time.Minute).Format("2006-01-02T15:04:05")
	scheduleBody := `{"schema_version":"1.0","host_id":"` + testHostID +
		`","scenario_id":"windows-process-marker","scenario_version":"0.1.0","scheduled_for_local":"` +
		scheduledLocal + `","time_zone":"Europe/Istanbul"}`
	scheduled := postWorkspaceAPI(handler, "/v1/schedules", scheduleBody, csrfCookie, csrfToken)
	if scheduled.Code != http.StatusCreated || !strings.Contains(scheduled.Body.String(), `"status":"planned"`) ||
		!strings.Contains(scheduled.Body.String(), `"scheduled_for_utc"`) {
		t.Fatalf("schedule = %d, body = %s", scheduled.Code, scheduled.Body.String())
	}
	conflict := postWorkspaceAPI(handler, "/v1/schedules", scheduleBody, csrfCookie, csrfToken)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), "SCHEDULE_CONFLICT") {
		t.Fatalf("conflict = %d, body = %s", conflict.Code, conflict.Body.String())
	}
	listed := getWorkspaceAPI(handler, "/v1/schedules", csrfCookie, csrfToken)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), scheduledLocal) {
		t.Fatalf("schedule list = %d, body = %s", listed.Code, listed.Body.String())
	}
}

func TestLocalLoginAuthorizesJobAndLogoutRevokesSession(t *testing.T) {
	server, queue := localAuthServer(t)
	handler := server.Handler()
	csrfCookie, csrfToken := session(t, handler)

	unauthorized := postJob(handler, validBody(), testIdempotency, testOrigin, csrfCookie, csrfToken)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated create status = %d, body = %s", unauthorized.Code, unauthorized.Body.String())
	}
	if queue.Depth() != 0 {
		t.Fatal("unauthenticated request was queued")
	}

	for _, credentials := range []struct {
		email    string
		password string
	}{
		{email: "unknown@prooflayer.local", password: "correct horse battery staple"},
		{email: "admin@prooflayer.local", password: "incorrect password value"},
	} {
		response := login(handler, credentials.email, credentials.password, csrfCookie, csrfToken)
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "AUTHENTICATION_FAILED") {
			t.Fatalf("invalid login = %d, body = %s", response.Code, response.Body.String())
		}
	}

	loggedIn := login(handler, " ADMIN@ProofLayer.Local ", "correct horse battery staple", csrfCookie, csrfToken)
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loggedIn.Code, loggedIn.Body.String())
	}
	sessionCookie := cookieNamed(t, loggedIn, sessionCookieName)
	rotatedCSRF := cookieNamed(t, loggedIn, csrfCookieName)
	if !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode ||
		!rotatedCSRF.HttpOnly || rotatedCSRF.SameSite != http.SameSiteStrictMode ||
		rotatedCSRF.Value == csrfToken {
		t.Fatalf("login cookies are not hardened or rotated: session=%+v csrf=%+v", sessionCookie, rotatedCSRF)
	}
	var loginDocument struct {
		Authenticated bool                `json:"authenticated"`
		CSRFToken     string              `json:"csrf_token"`
		User          localauth.Principal `json:"user"`
		Workspace     localauth.Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(loggedIn.Body.Bytes(), &loginDocument); err != nil {
		t.Fatal(err)
	}
	if !loginDocument.Authenticated || loginDocument.CSRFToken != rotatedCSRF.Value ||
		loginDocument.User.WorkspaceID != loginDocument.Workspace.ID {
		t.Fatalf("login document = %+v", loginDocument)
	}

	authorizedRequest := httptest.NewRequest(http.MethodPost, "/v1/test-jobs", strings.NewReader(validBody()))
	authorizedRequest.Header.Set("Content-Type", "application/json")
	authorizedRequest.Header.Set("Origin", testOrigin)
	authorizedRequest.Header.Set("Sec-Fetch-Site", "same-origin")
	authorizedRequest.Header.Set("Idempotency-Key", testIdempotency)
	authorizedRequest.Header.Set("X-ProofLayer-CSRF", rotatedCSRF.Value)
	authorizedRequest.AddCookie(rotatedCSRF)
	authorizedRequest.AddCookie(sessionCookie)
	authorized := httptest.NewRecorder()
	handler.ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusCreated || queue.Depth() != 1 {
		t.Fatalf("authenticated create = %d, body = %s, depth = %d", authorized.Code, authorized.Body.String(), queue.Depth())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	logoutRequest.Header.Set("Origin", testOrigin)
	logoutRequest.Header.Set("Sec-Fetch-Site", "same-origin")
	logoutRequest.Header.Set("X-ProofLayer-CSRF", rotatedCSRF.Value)
	logoutRequest.AddCookie(rotatedCSRF)
	logoutRequest.AddCookie(sessionCookie)
	logout := httptest.NewRecorder()
	handler.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", logout.Code, logout.Body.String())
	}
	if cookieNamed(t, logout, sessionCookieName).MaxAge != -1 || cookieNamed(t, logout, csrfCookieName).MaxAge != -1 {
		t.Fatal("logout did not clear browser cookies")
	}

	reuse := httptest.NewRequest(http.MethodGet, "/v1/test-jobs/00000000-0000-4000-8000-000000000000", nil)
	reuse.Header.Set("Sec-Fetch-Site", "same-origin")
	reuse.Header.Set("X-ProofLayer-CSRF", rotatedCSRF.Value)
	reuse.AddCookie(rotatedCSRF)
	reuse.AddCookie(sessionCookie)
	reused := httptest.NewRecorder()
	handler.ServeHTTP(reused, reuse)
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session reuse status = %d, body = %s", reused.Code, reused.Body.String())
	}
}

func TestAuthenticatedSessionDocumentIsWorkspaceBound(t *testing.T) {
	server, _ := localAuthServer(t)
	handler := server.Handler()
	csrfCookie, csrfToken := session(t, handler)
	loggedIn := login(handler, "admin@prooflayer.local", "correct horse battery staple", csrfCookie, csrfToken)
	sessionCookie := cookieNamed(t, loggedIn, sessionCookieName)

	request := httptest.NewRequest(http.MethodGet, "/v1/session", nil)
	request.AddCookie(sessionCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session status = %d", response.Code)
	}
	var document struct {
		Authenticated bool                `json:"authenticated"`
		User          localauth.Principal `json:"user"`
		Workspace     localauth.Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if !document.Authenticated || document.User.Role != localauth.RoleAdmin ||
		document.User.WorkspaceID != document.Workspace.ID {
		t.Fatalf("session document = %+v", document)
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
