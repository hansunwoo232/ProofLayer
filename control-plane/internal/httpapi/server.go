package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/localauth"
	"github.com/hansunwoo232/ProofLayer/control-plane/internal/mvpstore"
	"github.com/hansunwoo232/ProofLayer/control-plane/internal/runqueue"
)

const (
	csrfCookieName    = "prooflayer_csrf"
	sessionCookieName = "prooflayer_session"
	maximumBody       = 4096
)

type Server struct {
	queue         *runqueue.Queue
	allowedOrigin string
	dashboard     http.Handler
	csrfToken     func() (string, error)
	runner        *RunnerBinding
	localAuth     *localauth.Service
	workspace     *mvpstore.Service
}

func New(queue *runqueue.Queue, allowedOrigin, dashboardDirectory string) (*Server, error) {
	return newServer(queue, allowedOrigin, dashboardDirectory, nil, nil)
}

func NewWithLocalAuth(
	queue *runqueue.Queue,
	allowedOrigin,
	dashboardDirectory string,
	localAuth *localauth.Service,
) (*Server, error) {
	if localAuth == nil {
		return nil, errors.New("local authentication service is required")
	}
	return newServer(queue, allowedOrigin, dashboardDirectory, nil, localAuth)
}

func NewWithRunner(
	queue *runqueue.Queue,
	allowedOrigin,
	dashboardDirectory string,
	runner RunnerBinding,
) (*Server, error) {
	if err := runner.validate(); err != nil {
		return nil, err
	}
	runner.EnvironmentID = strings.ToLower(runner.EnvironmentID)
	runner.HostID = strings.ToLower(runner.HostID)
	runner.RunnerID = strings.ToLower(runner.RunnerID)
	if runner.Version == "" {
		runner.Version = "0.1.0"
	}
	return newServer(queue, allowedOrigin, dashboardDirectory, &runner, nil)
}

func NewWithRunnerAndLocalAuth(
	queue *runqueue.Queue,
	allowedOrigin,
	dashboardDirectory string,
	runner RunnerBinding,
	localAuth *localauth.Service,
) (*Server, error) {
	if err := runner.validate(); err != nil {
		return nil, err
	}
	if localAuth == nil {
		return nil, errors.New("local authentication service is required")
	}
	runner.EnvironmentID = strings.ToLower(runner.EnvironmentID)
	runner.HostID = strings.ToLower(runner.HostID)
	runner.RunnerID = strings.ToLower(runner.RunnerID)
	if runner.Version == "" {
		runner.Version = "0.1.0"
	}
	return newServer(queue, allowedOrigin, dashboardDirectory, &runner, localAuth)
}

func newServer(
	queue *runqueue.Queue,
	allowedOrigin,
	dashboardDirectory string,
	runner *RunnerBinding,
	localAuth *localauth.Service,
) (*Server, error) {
	if queue == nil {
		return nil, errors.New("job queue is required")
	}
	origin, err := url.Parse(allowedOrigin)
	if err != nil || origin.Scheme != "http" || origin.Hostname() != "127.0.0.1" || origin.Port() == "" ||
		origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return nil, errors.New("allowed origin must be an explicit loopback origin")
	}
	absDashboard, err := filepath.Abs(dashboardDirectory)
	if err != nil {
		return nil, err
	}
	environmentID, hostID := queue.Identity()
	workspaceID := "8ba7b810-9dad-41d1-80b4-00c04fd430c8"
	if localAuth != nil {
		workspaceID = localAuth.Workspace().ID
	}
	runnerID := ""
	runnerVersion := "0.1.0"
	if runner != nil {
		runnerID = runner.RunnerID
		runnerVersion = runner.Version
	}
	workspace, err := mvpstore.New(mvpstore.Config{
		WorkspaceID: workspaceID, EnvironmentID: environmentID, HostID: hostID,
		RunnerID: runnerID, HostName: "WIN-LAB-01", RunnerVersion: runnerVersion,
	})
	if err != nil {
		return nil, err
	}
	return &Server{
		queue:         queue,
		allowedOrigin: allowedOrigin,
		dashboard:     http.FileServer(http.Dir(absDashboard)),
		csrfToken:     randomCSRFToken,
		runner:        runner,
		localAuth:     localAuth,
		workspace:     workspace,
	}, nil
}

func (server *Server) Handler() http.Handler {
	return server.securityHeaders(http.HandlerFunc(server.route))
}

func (server *Server) route(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/v1/runners/") {
		server.handleRunnerRoute(writer, request)
		return
	}
	switch request.URL.Path {
	case "/":
		if server.localAuth != nil {
			if _, ok := server.authenticatedPrincipal(request); !ok {
				http.Redirect(writer, request, "/login.html", http.StatusSeeOther)
				return
			}
		}
		http.Redirect(writer, request, "/result-screen-wireframe.html", http.StatusSeeOther)
	case "/v1/session":
		server.handleSession(writer, request)
	case "/v1/auth/login":
		server.handleLogin(writer, request)
	case "/v1/auth/logout":
		server.handleLogout(writer, request)
	case "/v1/test-jobs":
		server.handleCreateJob(writer, request)
	case "/v1/scenarios":
		server.handleScenarios(writer, request)
	case "/v1/hosts":
		server.handleHosts(writer, request)
	case "/v1/schedules":
		server.handleSchedules(writer, request)
	case "/v1/test-runs":
		server.handleHistory(writer, request)
	default:
		if strings.HasPrefix(request.URL.Path, "/v1/test-jobs/") {
			server.handleJobStatus(writer, request)
			return
		}
		server.dashboard.ServeHTTP(writer, request)
	}
}

func (server *Server) DispatchDueSchedules() error {
	return server.workspace.DispatchDue(server.queue)
}

func (server *Server) handleSession(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	token, err := server.csrfToken()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "SESSION_TOKEN_FAILED")
		return
	}
	setCSRFCookie(writer, token, request.TLS != nil)
	writer.Header().Set("Cache-Control", "no-store")
	document := map[string]any{
		"schema_version": "1.0",
		"csrf_token":     token,
		"authenticated":  server.localAuth == nil,
	}
	if principal, ok := server.authenticatedPrincipal(request); ok {
		document["authenticated"] = true
		document["user"] = principal
		document["workspace"] = server.localAuth.Workspace()
	}
	writeJSON(writer, http.StatusOK, document)
}

type loginRequest struct {
	SchemaVersion string `json:"schema_version"`
	Email         string `json:"email"`
	Password      string `json:"password"`
}

func (server *Server) handleLogin(writer http.ResponseWriter, request *http.Request) {
	if server.localAuth == nil {
		writeError(writer, http.StatusNotFound, "LOCAL_AUTH_DISABLED")
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	if !server.validOrigin(request) || !validCSRF(request) {
		writeError(writer, http.StatusForbidden, "REQUEST_INTEGRITY_FAILED")
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(writer, http.StatusUnsupportedMediaType, "CONTENT_TYPE_INVALID")
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maximumBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var login loginRequest
	if err := decoder.Decode(&login); err != nil || ensureEOF(decoder) != nil ||
		login.SchemaVersion != "1.0" || len(login.Email) > 254 || len(login.Password) > 128 {
		writeError(writer, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	principal, err := server.localAuth.Authenticate(login.Email, login.Password)
	if err != nil {
		writer.Header().Set("WWW-Authenticate", `Session realm="ProofLayer"`)
		writeError(writer, http.StatusUnauthorized, "AUTHENTICATION_FAILED")
		return
	}
	sessionToken, err := server.localAuth.CreateSession(principal)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "SESSION_CREATION_FAILED")
		return
	}
	csrfToken, err := server.csrfToken()
	if err != nil {
		server.localAuth.RevokeSession(sessionToken)
		writeError(writer, http.StatusInternalServerError, "SESSION_TOKEN_FAILED")
		return
	}
	secure := request.TLS != nil
	setCSRFCookie(writer, csrfToken, secure)
	http.SetCookie(writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, http.StatusOK, map[string]any{
		"schema_version": "1.0",
		"authenticated":  true,
		"csrf_token":     csrfToken,
		"user":           principal,
		"workspace":      server.localAuth.Workspace(),
	})
}

func (server *Server) handleLogout(writer http.ResponseWriter, request *http.Request) {
	if server.localAuth == nil {
		writeError(writer, http.StatusNotFound, "LOCAL_AUTH_DISABLED")
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	if !server.validOrigin(request) || !validCSRF(request) {
		writeError(writer, http.StatusForbidden, "REQUEST_INTEGRITY_FAILED")
		return
	}
	sessionCookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED")
		return
	}
	if _, err := server.localAuth.VerifySession(sessionCookie.Value); err != nil {
		clearBrowserCookies(writer, request.TLS != nil)
		writeError(writer, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED")
		return
	}
	server.localAuth.RevokeSession(sessionCookie.Value)
	clearBrowserCookies(writer, request.TLS != nil)
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusNoContent)
}

func (server *Server) handleCreateJob(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	if !server.validOrigin(request) || !validCSRF(request) {
		writeError(writer, http.StatusForbidden, "REQUEST_INTEGRITY_FAILED")
		return
	}
	if !server.requireAuthentication(writer, request) {
		return
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(writer, http.StatusUnsupportedMediaType, "CONTENT_TYPE_INVALID")
		return
	}

	idempotencyKey := request.Header.Get("Idempotency-Key")
	request.Body = http.MaxBytesReader(writer, request.Body, maximumBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var createRequest runqueue.CreateRequest
	if err := decoder.Decode(&createRequest); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(writer, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE")
			return
		}
		writeError(writer, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	if err := ensureEOF(decoder); err != nil {
		writeError(writer, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	if err := server.workspace.Authorize(createRequest.HostID, createRequest.ScenarioID, createRequest.ScenarioVersion); err != nil {
		switch {
		case errors.Is(err, mvpstore.ErrHostAccessDenied):
			writeError(writer, http.StatusForbidden, "HOST_ACCESS_DENIED")
		default:
			writeError(writer, http.StatusBadRequest, "SCENARIO_INVALID")
		}
		return
	}

	receipt, err := server.queue.Enqueue(idempotencyKey, createRequest)
	if err != nil {
		switch {
		case errors.Is(err, runqueue.ErrInvalidRequest), errors.Is(err, runqueue.ErrInvalidIdempotency):
			writeError(writer, http.StatusBadRequest, "REQUEST_INVALID")
		case errors.Is(err, runqueue.ErrIdempotencyConflict):
			writeError(writer, http.StatusConflict, "IDEMPOTENCY_CONFLICT")
		case errors.Is(err, runqueue.ErrQueueFull):
			writeError(writer, http.StatusServiceUnavailable, "QUEUE_CAPACITY_REACHED")
		default:
			writeError(writer, http.StatusInternalServerError, "JOB_CREATION_FAILED")
		}
		return
	}
	status := http.StatusCreated
	if receipt.Replayed {
		status = http.StatusOK
	}
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, status, receipt)
}

func (server *Server) handleScenarios(writer http.ResponseWriter, request *http.Request) {
	if !server.readRequestAllowed(writer, request) {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"schema_version": mvpstore.SchemaVersion,
		"items":          server.workspace.Scenarios(),
	})
}

func (server *Server) handleHosts(writer http.ResponseWriter, request *http.Request) {
	if !server.readRequestAllowed(writer, request) {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"schema_version": mvpstore.SchemaVersion,
		"items":          server.workspace.Hosts(),
	})
}

func (server *Server) handleSchedules(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		if !server.readRequestAllowed(writer, request) {
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"schema_version": mvpstore.SchemaVersion,
			"items":          server.workspace.Schedules(),
		})
	case http.MethodPost:
		if !server.mutationRequestAllowed(writer, request) {
			return
		}
		var scheduleRequest mvpstore.ScheduleRequest
		if !decodeBrowserJSON(writer, request, &scheduleRequest) {
			return
		}
		schedule, err := server.workspace.CreateSchedule(scheduleRequest)
		if err != nil {
			switch {
			case errors.Is(err, mvpstore.ErrScheduleConflict):
				writeError(writer, http.StatusConflict, "SCHEDULE_CONFLICT")
			case errors.Is(err, mvpstore.ErrScheduleMissed):
				writeError(writer, http.StatusUnprocessableEntity, "SCHEDULE_TIME_PASSED")
			default:
				writeError(writer, http.StatusBadRequest, "SCHEDULE_INVALID")
			}
			return
		}
		writeJSON(writer, http.StatusCreated, schedule)
	default:
		writer.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
	}
}

func (server *Server) handleHistory(writer http.ResponseWriter, request *http.Request) {
	if !server.readRequestAllowed(writer, request) {
		return
	}
	query := runqueue.HistoryQuery{
		HostID: request.URL.Query().Get("host_id"), ScenarioID: request.URL.Query().Get("scenario_id"),
	}
	var err error
	if value := request.URL.Query().Get("page"); value != "" {
		query.Page, err = strconv.Atoi(value)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "HISTORY_QUERY_INVALID")
			return
		}
	}
	if value := request.URL.Query().Get("page_size"); value != "" {
		query.PageSize, err = strconv.Atoi(value)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "HISTORY_QUERY_INVALID")
			return
		}
	}
	if query.From, err = parseOptionalTime(request.URL.Query().Get("from")); err != nil {
		writeError(writer, http.StatusBadRequest, "HISTORY_QUERY_INVALID")
		return
	}
	if query.To, err = parseOptionalTime(request.URL.Query().Get("to")); err != nil {
		writeError(writer, http.StatusBadRequest, "HISTORY_QUERY_INVALID")
		return
	}
	page, err := server.queue.History(query)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "HISTORY_QUERY_INVALID")
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (server *Server) readRequestAllowed(writer http.ResponseWriter, request *http.Request) bool {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return false
	}
	if !validCSRF(request) ||
		(request.Header.Get("Sec-Fetch-Site") != "" && request.Header.Get("Sec-Fetch-Site") != "same-origin") {
		writeError(writer, http.StatusForbidden, "REQUEST_INTEGRITY_FAILED")
		return false
	}
	if !server.requireAuthentication(writer, request) {
		return false
	}
	writer.Header().Set("Cache-Control", "no-store")
	return true
}

func (server *Server) mutationRequestAllowed(writer http.ResponseWriter, request *http.Request) bool {
	if !server.validOrigin(request) || !validCSRF(request) {
		writeError(writer, http.StatusForbidden, "REQUEST_INTEGRITY_FAILED")
		return false
	}
	return server.requireAuthentication(writer, request)
}

func decodeBrowserJSON(writer http.ResponseWriter, request *http.Request, target any) bool {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(writer, http.StatusUnsupportedMediaType, "CONTENT_TYPE_INVALID")
		return false
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maximumBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil || ensureEOF(decoder) != nil {
		writeError(writer, http.StatusBadRequest, "REQUEST_INVALID")
		return false
	}
	return true
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func (server *Server) handleJobStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	if !validCSRF(request) ||
		(request.Header.Get("Sec-Fetch-Site") != "" && request.Header.Get("Sec-Fetch-Site") != "same-origin") {
		writeError(writer, http.StatusForbidden, "REQUEST_INTEGRITY_FAILED")
		return
	}
	if !server.requireAuthentication(writer, request) {
		return
	}
	jobID := strings.TrimPrefix(request.URL.Path, "/v1/test-jobs/")
	if jobID == "" || strings.Contains(jobID, "/") {
		writeError(writer, http.StatusNotFound, "JOB_NOT_FOUND")
		return
	}
	status, ok := server.queue.Status(jobID)
	if !ok {
		writeError(writer, http.StatusNotFound, "JOB_NOT_FOUND")
		return
	}
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, http.StatusOK, status)
}

func (server *Server) authenticatedPrincipal(request *http.Request) (localauth.Principal, bool) {
	if server.localAuth == nil {
		return localauth.Principal{}, false
	}
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		return localauth.Principal{}, false
	}
	principal, err := server.localAuth.VerifySession(cookie.Value)
	return principal, err == nil
}

func (server *Server) requireAuthentication(writer http.ResponseWriter, request *http.Request) bool {
	if server.localAuth == nil {
		return true
	}
	if _, ok := server.authenticatedPrincipal(request); !ok {
		writer.Header().Set("WWW-Authenticate", `Session realm="ProofLayer"`)
		writeError(writer, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED")
		return false
	}
	return true
}

func (server *Server) validOrigin(request *http.Request) bool {
	return request.Header.Get("Origin") == server.allowedOrigin &&
		(request.Header.Get("Sec-Fetch-Site") == "" || request.Header.Get("Sec-Fetch-Site") == "same-origin")
}

func validCSRF(request *http.Request) bool {
	cookie, err := request.Cookie(csrfCookieName)
	if err != nil {
		return false
	}
	header := request.Header.Get("X-ProofLayer-CSRF")
	if len(cookie.Value) < 32 || len(cookie.Value) != len(header) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) == 1
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON object")
	}
	return nil
}

func randomCSRFToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func setCSRFCookie(writer http.ResponseWriter, token string, secure bool) {
	http.SetCookie(writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   1800,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearBrowserCookies(writer http.ResponseWriter, secure bool) {
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		http.SetCookie(writer, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteStrictMode,
		})
	}
}

func (server *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(writer, request)
	})
}

func writeError(writer http.ResponseWriter, status int, code string) {
	writeJSON(writer, status, map[string]string{
		"schema_version": "1.0",
		"code":           code,
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
