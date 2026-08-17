package runqueue

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	SchemaVersion             = "1.0"
	jobLifetime               = 2 * time.Minute
	idempotencyRetention      = 24 * time.Hour
	maximumIdempotencyRecords = 1024
)

var (
	ErrInvalidRequest      = errors.New("invalid run request")
	ErrInvalidIdempotency  = errors.New("invalid idempotency key")
	ErrIdempotencyConflict = errors.New("idempotency key conflicts with an existing request")
	ErrQueueFull           = errors.New("job queue is full")

	uuidPattern        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22,64}$`)
	approvedScenarios  = map[string]string{
		"windows-process-marker":          "0.1.0",
		"windows-registry-run-key-canary": "0.1.0",
		"windows-scheduled-task-canary":   "0.1.0",
	}
)

type CreateRequest struct {
	SchemaVersion   string `json:"schema_version"`
	EnvironmentID   string `json:"environment_id"`
	HostID          string `json:"host_id"`
	ScenarioID      string `json:"scenario_id"`
	ScenarioVersion string `json:"scenario_version"`
}

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

type Receipt struct {
	SchemaVersion string `json:"schema_version"`
	JobID         string `json:"job_id"`
	CorrelationID string `json:"correlation_id"`
	Status        string `json:"status"`
	Replayed      bool   `json:"replayed"`
}

type Config struct {
	Capacity          int
	EnvironmentID     string
	HostID            string
	RequestedBy       string
	SigningKeyID      string
	SigningPrivateKey ed25519.PrivateKey
	Now               func() time.Time
}

type idempotencyRecord struct {
	fingerprint [32]byte
	receipt     Receipt
	expiresAt   time.Time
}

type Queue struct {
	mu          sync.Mutex
	capacity    int
	environment string
	host        string
	requestedBy string
	keyID       string
	privateKey  ed25519.PrivateKey
	now         func() time.Time
	jobs        []Job
	records     map[string]idempotencyRecord
}

func New(config Config) (*Queue, error) {
	if config.Capacity < 1 || config.Capacity > 128 {
		return nil, fmt.Errorf("queue capacity must be between 1 and 128")
	}
	if !validUUID(config.EnvironmentID) || !validUUID(config.HostID) || !validUUID(config.RequestedBy) {
		return nil, fmt.Errorf("queue identity binding is invalid")
	}
	if config.SigningKeyID == "" || len(config.SigningKeyID) > 80 {
		return nil, fmt.Errorf("signing key ID is invalid")
	}
	if len(config.SigningPrivateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("Ed25519 private key is invalid")
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	return &Queue{
		capacity:    config.Capacity,
		environment: strings.ToLower(config.EnvironmentID),
		host:        strings.ToLower(config.HostID),
		requestedBy: strings.ToLower(config.RequestedBy),
		keyID:       config.SigningKeyID,
		privateKey:  append(ed25519.PrivateKey(nil), config.SigningPrivateKey...),
		now:         now,
		jobs:        make([]Job, 0, config.Capacity),
		records:     make(map[string]idempotencyRecord),
	}, nil
}

func (queue *Queue) Enqueue(idempotencyKey string, request CreateRequest) (Receipt, error) {
	if !idempotencyPattern.MatchString(idempotencyKey) {
		return Receipt{}, ErrInvalidIdempotency
	}
	request = normalize(request)
	if err := queue.validate(request); err != nil {
		return Receipt{}, err
	}
	encodedRequest, err := json.Marshal(request)
	if err != nil {
		return Receipt{}, fmt.Errorf("encode request fingerprint: %w", err)
	}
	fingerprint := sha256.Sum256(encodedRequest)

	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneExpiredLocked(queue.now().UTC())

	if record, ok := queue.records[idempotencyKey]; ok {
		if record.fingerprint != fingerprint {
			return Receipt{}, ErrIdempotencyConflict
		}
		receipt := record.receipt
		receipt.Replayed = true
		return receipt, nil
	}
	if len(queue.jobs) >= queue.capacity {
		return Receipt{}, ErrQueueFull
	}
	if len(queue.records) >= maximumIdempotencyRecords {
		return Receipt{}, ErrQueueFull
	}

	job, err := queue.createJob(request)
	if err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{
		SchemaVersion: SchemaVersion,
		JobID:         job.JobID,
		CorrelationID: job.CorrelationID,
		Status:        "queued",
		Replayed:      false,
	}
	queue.jobs = append(queue.jobs, job)
	queue.records[idempotencyKey] = idempotencyRecord{
		fingerprint: fingerprint,
		receipt:     receipt,
		expiresAt:   job.RequestedAt.Add(idempotencyRetention),
	}
	return receipt, nil
}

func (queue *Queue) Lease(environmentID, hostID string) (Job, bool) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneExpiredLocked(queue.now().UTC())

	environmentID = strings.ToLower(environmentID)
	hostID = strings.ToLower(hostID)
	for index, job := range queue.jobs {
		if job.EnvironmentID != environmentID || job.HostID != hostID {
			continue
		}
		queue.jobs = append(queue.jobs[:index], queue.jobs[index+1:]...)
		return cloneJob(job), true
	}
	return Job{}, false
}

func (queue *Queue) Depth() int {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	return len(queue.jobs)
}

func (queue *Queue) pruneExpiredLocked(now time.Time) {
	activeJobs := queue.jobs[:0]
	for _, job := range queue.jobs {
		if now.Before(job.ExpiresAt) {
			activeJobs = append(activeJobs, job)
		}
	}
	queue.jobs = activeJobs
	for key, record := range queue.records {
		if !now.Before(record.expiresAt) {
			delete(queue.records, key)
		}
	}
}

func Verify(job Job, publicKey ed25519.PublicKey) bool {
	payload, err := json.Marshal(unsigned(job))
	if err != nil {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(job.Signature.Value)
	if err != nil {
		return false
	}
	return job.Signature.Algorithm == "Ed25519" && ed25519.Verify(publicKey, payload, signature)
}

func (queue *Queue) validate(request CreateRequest) error {
	if request.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: schema_version", ErrInvalidRequest)
	}
	if request.EnvironmentID != queue.environment || request.HostID != queue.host {
		return fmt.Errorf("%w: identity_binding", ErrInvalidRequest)
	}
	version, ok := approvedScenarios[request.ScenarioID]
	if !ok || request.ScenarioVersion != version {
		return fmt.Errorf("%w: scenario_allowlist", ErrInvalidRequest)
	}
	return nil
}

func (queue *Queue) createJob(request CreateRequest) (Job, error) {
	jobID, err := randomUUID()
	if err != nil {
		return Job{}, err
	}
	correlationID, err := randomCorrelationID()
	if err != nil {
		return Job{}, err
	}
	nonce, err := randomToken(24)
	if err != nil {
		return Job{}, err
	}
	now := queue.now().UTC()
	job := Job{
		SchemaVersion:   SchemaVersion,
		JobID:           jobID,
		CorrelationID:   correlationID,
		EnvironmentID:   request.EnvironmentID,
		HostID:          request.HostID,
		ScenarioID:      request.ScenarioID,
		ScenarioVersion: request.ScenarioVersion,
		RequestedBy:     queue.requestedBy,
		RequestedAt:     now,
		ExpiresAt:       now.Add(jobLifetime),
		Nonce:           nonce,
		Parameters:      map[string]any{},
	}
	payload, err := json.Marshal(unsigned(job))
	if err != nil {
		return Job{}, fmt.Errorf("encode signed job: %w", err)
	}
	job.Signature = Signature{
		Algorithm: "Ed25519",
		KeyID:     queue.keyID,
		Value:     base64.RawURLEncoding.EncodeToString(ed25519.Sign(queue.privateKey, payload)),
	}
	return job, nil
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

func normalize(request CreateRequest) CreateRequest {
	request.EnvironmentID = strings.ToLower(request.EnvironmentID)
	request.HostID = strings.ToLower(request.HostID)
	return request
}

func cloneJob(job Job) Job {
	job.Parameters = make(map[string]any, len(job.Parameters))
	for key, value := range job.Parameters {
		job.Parameters[key] = value
	}
	return job
}

func validUUID(value string) bool {
	return uuidPattern.MatchString(strings.ToLower(value))
}

func randomUUID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate job ID: %w", err)
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(buffer)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}

func randomCorrelationID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate correlation ID: %w", err)
	}
	return "PL-" + strings.ToUpper(hex.EncodeToString(buffer)), nil
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
