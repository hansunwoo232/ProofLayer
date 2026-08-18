package controlplane

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	runnerjob "github.com/hansunwoo232/ProofLayer/runner/internal/job"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

const SchemaVersion = "1.0"

var (
	ErrJobInvalid   = errors.New("leased job is invalid")
	ErrJobSignature = errors.New("leased job signature is invalid")
	ErrJobIdentity  = errors.New("leased job identity does not match this Runner")
	ErrJobExpired   = errors.New("leased job has expired")
	ErrJobReplayed  = errors.New("leased job nonce was already consumed")

	jobUUIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	noncePattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{32,64}$`)
)

type Signature struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	Value     string `json:"value"`
}

type Job struct {
	SchemaVersion   string         `json:"schema_version"`
	JobID           string         `json:"job_id"`
	CorrelationID   string         `json:"correlation_id"`
	EnvironmentID   string         `json:"environment_id"`
	HostID          string         `json:"host_id"`
	ScenarioID      string         `json:"scenario_id"`
	ScenarioVersion string         `json:"scenario_version"`
	RequestedBy     string         `json:"requested_by"`
	RequestedAt     time.Time      `json:"requested_at"`
	ExpiresAt       time.Time      `json:"expires_at"`
	Nonce           string         `json:"nonce"`
	Parameters      map[string]any `json:"parameters"`
	Signature       Signature      `json:"signature"`
}

type unsignedJob struct {
	SchemaVersion   string         `json:"schema_version"`
	JobID           string         `json:"job_id"`
	CorrelationID   string         `json:"correlation_id"`
	EnvironmentID   string         `json:"environment_id"`
	HostID          string         `json:"host_id"`
	ScenarioID      string         `json:"scenario_id"`
	ScenarioVersion string         `json:"scenario_version"`
	RequestedBy     string         `json:"requested_by"`
	RequestedAt     time.Time      `json:"requested_at"`
	ExpiresAt       time.Time      `json:"expires_at"`
	Nonce           string         `json:"nonce"`
	Parameters      map[string]any `json:"parameters"`
}

type Verifier struct {
	mu         sync.Mutex
	identity   identity.RunnerIdentity
	catalog    scenario.Catalog
	keyID      string
	publicKey  ed25519.PublicKey
	consumed   map[string]time.Time
	maximumAge time.Duration
}

func NewVerifier(
	runnerIdentity identity.RunnerIdentity,
	catalog scenario.Catalog,
	keyID string,
	publicKey ed25519.PublicKey,
) (*Verifier, error) {
	if keyID == "" || len(keyID) > 80 || len(publicKey) != ed25519.PublicKeySize {
		return nil, ErrJobInvalid
	}
	return &Verifier{
		identity:   runnerIdentity,
		catalog:    catalog,
		keyID:      keyID,
		publicKey:  append(ed25519.PublicKey(nil), publicKey...),
		consumed:   make(map[string]time.Time),
		maximumAge: 2 * time.Minute,
	}, nil
}

func (verifier *Verifier) VerifyAndConsume(job Job, now time.Time) error {
	if now.IsZero() || verifier.identity.Validate(now) != nil {
		return ErrJobInvalid
	}
	if job.SchemaVersion != SchemaVersion || !jobUUIDPattern.MatchString(strings.ToLower(job.JobID)) ||
		!jobUUIDPattern.MatchString(strings.ToLower(job.RequestedBy)) || !correlation.Valid(job.CorrelationID) ||
		!noncePattern.MatchString(job.Nonce) || len(job.Parameters) != 0 {
		return ErrJobInvalid
	}
	if !verifier.identity.Authorizes(job.EnvironmentID, job.HostID) {
		return ErrJobIdentity
	}
	if job.RequestedAt.IsZero() || job.ExpiresAt.IsZero() || !job.ExpiresAt.After(job.RequestedAt) ||
		job.ExpiresAt.Sub(job.RequestedAt) > verifier.maximumAge || job.RequestedAt.After(now.Add(time.Minute)) {
		return ErrJobInvalid
	}
	if !now.Before(job.ExpiresAt) {
		return ErrJobExpired
	}
	if _, err := verifier.catalog.Resolve(job.ScenarioID, job.ScenarioVersion); err != nil {
		return ErrJobInvalid
	}
	if !verifier.validSignature(job) {
		return ErrJobSignature
	}

	verifier.mu.Lock()
	defer verifier.mu.Unlock()
	for nonce, expiresAt := range verifier.consumed {
		if !now.Before(expiresAt) {
			delete(verifier.consumed, nonce)
		}
	}
	if _, ok := verifier.consumed[job.Nonce]; ok {
		return ErrJobReplayed
	}
	if len(verifier.consumed) >= 1024 {
		return ErrJobInvalid
	}
	verifier.consumed[job.Nonce] = job.ExpiresAt
	return nil
}

func (verifier *Verifier) validSignature(job Job) bool {
	if job.Signature.Algorithm != "Ed25519" || job.Signature.KeyID != verifier.keyID {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(job.Signature.Value)
	if err != nil {
		return false
	}
	payload, err := json.Marshal(unsigned(job))
	if err != nil {
		return false
	}
	return ed25519.Verify(verifier.publicKey, payload, signature)
}

func (job Job) ExecutionRequest() runnerjob.ExecutionRequest {
	parameters := make(map[string]any, len(job.Parameters))
	for key, value := range job.Parameters {
		parameters[key] = value
	}
	return runnerjob.ExecutionRequest{
		CorrelationID:   job.CorrelationID,
		EnvironmentID:   job.EnvironmentID,
		HostID:          job.HostID,
		ScenarioID:      job.ScenarioID,
		ScenarioVersion: job.ScenarioVersion,
		Parameters:      parameters,
	}
}

func unsigned(job Job) unsignedJob {
	return unsignedJob{
		SchemaVersion:   job.SchemaVersion,
		JobID:           job.JobID,
		CorrelationID:   job.CorrelationID,
		EnvironmentID:   job.EnvironmentID,
		HostID:          job.HostID,
		ScenarioID:      job.ScenarioID,
		ScenarioVersion: job.ScenarioVersion,
		RequestedBy:     job.RequestedBy,
		RequestedAt:     job.RequestedAt,
		ExpiresAt:       job.ExpiresAt,
		Nonce:           job.Nonce,
		Parameters:      job.Parameters,
	}
}
