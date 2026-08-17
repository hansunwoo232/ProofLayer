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
	"strings"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/runqueue"
)

const (
	csrfCookieName = "prooflayer_csrf"
	maximumBody    = 4096
)

type Server struct {
	queue         *runqueue.Queue
	allowedOrigin string
	dashboard     http.Handler
	csrfToken     func() (string, error)
}

func New(queue *runqueue.Queue, allowedOrigin, dashboardDirectory string) (*Server, error) {
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
	return &Server{
		queue:         queue,
		allowedOrigin: allowedOrigin,
		dashboard:     http.FileServer(http.Dir(absDashboard)),
		csrfToken:     randomCSRFToken,
	}, nil
}

func (server *Server) Handler() http.Handler {
	return server.securityHeaders(http.HandlerFunc(server.route))
}

func (server *Server) route(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/":
		http.Redirect(writer, request, "/result-screen-wireframe.html", http.StatusSeeOther)
	case "/v1/session":
		server.handleSession(writer, request)
	case "/v1/test-jobs":
		server.handleCreateJob(writer, request)
	default:
		if strings.HasPrefix(request.URL.Path, "/v1/test-jobs/") {
			server.handleJobStatus(writer, request)
			return
		}
		server.dashboard.ServeHTTP(writer, request)
	}
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
	http.SetCookie(writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   1800,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, http.StatusOK, map[string]string{
		"schema_version": "1.0",
		"csrf_token":     token,
	})
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
