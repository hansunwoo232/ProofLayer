package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/runqueue"
)

var (
	runnerUUIDPattern  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	runnerTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,128}$`)
)

type RunnerBinding struct {
	RunnerID      string
	EnvironmentID string
	HostID        string
	BearerToken   string
}

type versionRequest struct {
	SchemaVersion string `json:"schema_version"`
}

type acknowledgeRequest struct {
	SchemaVersion string `json:"schema_version"`
	Accepted      bool   `json:"accepted"`
}

type stageRequest struct {
	SchemaVersion string               `json:"schema_version"`
	Status        runqueue.StageStatus `json:"status"`
	LatencyMS     int64                `json:"latency_ms"`
	DetailCode    string               `json:"detail_code,omitempty"`
}

func (binding RunnerBinding) validate() error {
	for _, value := range []string{binding.RunnerID, binding.EnvironmentID, binding.HostID} {
		if !runnerUUIDPattern.MatchString(strings.ToLower(value)) {
			return errors.New("runner identity binding is invalid")
		}
	}
	if !runnerTokenPattern.MatchString(binding.BearerToken) {
		return errors.New("runner bearer token is invalid")
	}
	return nil
}

func (server *Server) handleRunnerRoute(writer http.ResponseWriter, request *http.Request) {
	if server.runner == nil {
		writeError(writer, http.StatusNotFound, "RUNNER_ROUTE_NOT_FOUND")
		return
	}
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[0] != "v1" || parts[1] != "runners" ||
		!server.validRunnerAuthorization(request, parts[2]) {
		writer.Header().Set("WWW-Authenticate", `Bearer realm="prooflayer-runner"`)
		writeError(writer, http.StatusUnauthorized, "RUNNER_AUTHENTICATION_FAILED")
		return
	}
	writer.Header().Set("Cache-Control", "no-store")

	switch {
	case len(parts) == 4 && parts[3] == "jobs:lease":
		server.handleRunnerLease(writer, request)
	case len(parts) == 5 && parts[3] == "jobs" && strings.HasSuffix(parts[4], ":ack"):
		server.handleRunnerAcknowledge(writer, request, strings.TrimSuffix(parts[4], ":ack"))
	case len(parts) == 7 && parts[3] == "jobs" && parts[5] == "stages":
		server.handleRunnerStage(writer, request, parts[4], parts[6])
	case len(parts) == 5 && parts[3] == "jobs" && strings.HasSuffix(parts[4], ":complete"):
		server.handleRunnerComplete(writer, request, strings.TrimSuffix(parts[4], ":complete"))
	default:
		writeError(writer, http.StatusNotFound, "RUNNER_ROUTE_NOT_FOUND")
	}
}

func (server *Server) validRunnerAuthorization(request *http.Request, runnerID string) bool {
	if strings.ToLower(runnerID) != server.runner.RunnerID {
		return false
	}
	prefix := "Bearer "
	authorization := request.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, prefix) {
		return false
	}
	token := strings.TrimPrefix(authorization, prefix)
	if len(token) != len(server.runner.BearerToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(server.runner.BearerToken)) == 1
}

func (server *Server) handleRunnerLease(writer http.ResponseWriter, request *http.Request) {
	if !runnerMethod(writer, request, http.MethodPost) {
		return
	}
	var document versionRequest
	if !decodeRunnerRequest(writer, request, &document) {
		return
	}
	if document.SchemaVersion != runqueue.SchemaVersion {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return
	}
	job, ok := server.queue.Lease(server.runner.EnvironmentID, server.runner.HostID)
	if !ok {
		writer.Header().Set("Cache-Control", "no-store")
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, http.StatusOK, job)
}

func (server *Server) handleRunnerAcknowledge(writer http.ResponseWriter, request *http.Request, jobID string) {
	if !runnerMethod(writer, request, http.MethodPost) {
		return
	}
	var document acknowledgeRequest
	if jobID == "" {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return
	}
	if !decodeRunnerRequest(writer, request, &document) {
		return
	}
	if document.SchemaVersion != runqueue.SchemaVersion {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return
	}
	if err := server.queue.Acknowledge(server.runner.EnvironmentID, server.runner.HostID, jobID, document.Accepted); err != nil {
		writeRunnerLifecycleError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, versionRequest{SchemaVersion: runqueue.SchemaVersion})
}

func (server *Server) handleRunnerStage(writer http.ResponseWriter, request *http.Request, jobID, stage string) {
	if !runnerMethod(writer, request, http.MethodPut) {
		return
	}
	var document stageRequest
	if jobID == "" || stage == "" {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return
	}
	if !decodeRunnerRequest(writer, request, &document) {
		return
	}
	if document.SchemaVersion != runqueue.SchemaVersion {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return
	}
	err := server.queue.UpdateStage(server.runner.EnvironmentID, server.runner.HostID, jobID, runqueue.StageUpdate{
		Stage: stage, Status: document.Status, LatencyMS: document.LatencyMS, DetailCode: document.DetailCode,
	})
	if err != nil {
		writeRunnerLifecycleError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, versionRequest{SchemaVersion: runqueue.SchemaVersion})
}

func (server *Server) handleRunnerComplete(writer http.ResponseWriter, request *http.Request, jobID string) {
	if !runnerMethod(writer, request, http.MethodPost) {
		return
	}
	var document versionRequest
	if jobID == "" {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return
	}
	if !decodeRunnerRequest(writer, request, &document) {
		return
	}
	if document.SchemaVersion != runqueue.SchemaVersion {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return
	}
	if err := server.queue.Complete(server.runner.EnvironmentID, server.runner.HostID, jobID); err != nil {
		writeRunnerLifecycleError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, versionRequest{SchemaVersion: runqueue.SchemaVersion})
}

func runnerMethod(writer http.ResponseWriter, request *http.Request, method string) bool {
	if request.Method == method {
		return true
	}
	writer.Header().Set("Allow", method)
	writeError(writer, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
	return false
}

func decodeRunnerRequest(writer http.ResponseWriter, request *http.Request, target any) bool {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(writer, http.StatusUnsupportedMediaType, "CONTENT_TYPE_INVALID")
		return false
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maximumBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(writer, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE")
			return false
		}
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return false
	}
	if ensureEOF(decoder) != nil {
		writeError(writer, http.StatusBadRequest, "RUNNER_REQUEST_INVALID")
		return false
	}
	return true
}

func writeRunnerLifecycleError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runqueue.ErrJobNotFound):
		writeError(writer, http.StatusNotFound, "JOB_NOT_FOUND")
	case errors.Is(err, runqueue.ErrInvalidRequest):
		writeError(writer, http.StatusForbidden, "RUNNER_IDENTITY_MISMATCH")
	case errors.Is(err, runqueue.ErrInvalidTransition):
		writeError(writer, http.StatusConflict, "JOB_TRANSITION_REJECTED")
	default:
		writeError(writer, http.StatusInternalServerError, "RUNNER_UPDATE_FAILED")
	}
}
