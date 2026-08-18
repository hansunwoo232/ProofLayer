// prooflayer-day30-lab runs one verified process-marker job through the local
// Control Plane, Sysmon, HEC, Splunk field validation, and detection pipeline.
// It is intentionally restricted to the isolated QEMU lab endpoints.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/audit"
	"github.com/hansunwoo232/ProofLayer/runner/internal/controlplane"
	"github.com/hansunwoo232/ProofLayer/runner/internal/executor"
	"github.com/hansunwoo232/ProofLayer/runner/internal/hec"
	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	"github.com/hansunwoo232/ProofLayer/runner/internal/observer"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
	"github.com/hansunwoo232/ProofLayer/runner/internal/splunk"
	"github.com/hansunwoo232/ProofLayer/runner/internal/worker"
)

const (
	labRunnerID      = "6ba7b812-9dad-41d1-80b4-00c04fd430c8"
	labEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	labHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
	labSigningKeyID  = "local-poc-ed25519-v1"
	maximumConfig    = 16 << 10
)

type labConfig struct {
	SchemaVersion          string `json:"schema_version"`
	ControlPlaneBaseURL    string `json:"control_plane_base_url"`
	RunnerToken            string `json:"runner_token"`
	SigningPublicKey       string `json:"signing_public_key"`
	HECEndpoint            string `json:"hec_endpoint"`
	HECToken               string `json:"hec_token"`
	SplunkBaseURL          string `json:"splunk_base_url"`
	SplunkObserverPassword string `json:"splunk_observer_password"`
	AllowLabSelfSignedTLS  bool   `json:"allow_lab_self_signed_tls"`
}

func main() {
	if len(os.Args) != 1 {
		fatal("this fixed Day 30 lab harness accepts no arguments")
	}
	config, err := loadConfig()
	if err != nil {
		fatal(err.Error())
	}
	publicKeyBytes, err := base64.RawURLEncoding.DecodeString(config.SigningPublicKey)
	if err != nil || len(publicKeyBytes) != ed25519.PublicKeySize {
		fatal("signing_public_key must be a base64url-encoded Ed25519 public key")
	}

	now := time.Now().UTC()
	runnerIdentity := identity.RunnerIdentity{
		SchemaVersion: identity.SchemaVersion,
		RunnerID:      labRunnerID, EnvironmentID: labEnvironmentID, HostID: labHostID,
		IdentityKeyID: "isolated-day30-lab", RegisteredAt: now.Add(-time.Minute),
		State: identity.StateActive,
	}
	tlsClient := isolatedLabHTTPClient()
	controlPlane, err := controlplane.New(controlplane.Config{
		BaseURL: config.ControlPlaneBaseURL, BearerToken: config.RunnerToken,
		Identity: runnerIdentity, SigningKeyID: labSigningKeyID,
		SigningPublicKey: ed25519.PublicKey(publicKeyBytes), HTTPClient: tlsClient,
	})
	if err != nil {
		fatal("create Control Plane client: " + err.Error())
	}
	exporter, err := hec.New(hec.Config{
		Endpoint: config.HECEndpoint, Token: config.HECToken,
		HTTPClient: isolatedLabHTTPClient(),
	})
	if err != nil {
		fatal("create HEC exporter: " + err.Error())
	}
	siem, err := splunk.New(splunk.Config{
		BaseURL: config.SplunkBaseURL, Username: splunk.ObserverUsername,
		Password: config.SplunkObserverPassword,
	}, isolatedLabHTTPClient())
	if err != nil {
		fatal("create Splunk observer: " + err.Error())
	}
	endpoint, err := observer.NewSysmonObserver(observer.NewWindowsEventSource(), observer.ApprovedSysmonPolicy())
	if err != nil {
		fatal("create endpoint observer: " + err.Error())
	}

	dataRoot := os.Getenv("ProgramData")
	if dataRoot == "" {
		dataRoot = os.TempDir()
	}
	auditDirectory := filepath.Join(dataRoot, "ProofLayer")
	if err := os.MkdirAll(auditDirectory, 0o700); err != nil {
		fatal("create audit directory: " + err.Error())
	}
	recorder, err := audit.NewFileRecorder(filepath.Join(auditDirectory, "runner-audit.jsonl"))
	if err != nil {
		fatal("create audit recorder: " + err.Error())
	}
	scenarioExecutor := executor.New(runnerIdentity, scenario.BuiltInCatalog(), recorder)
	jobWorker, err := worker.New(controlPlane, scenarioExecutor, endpoint, exporter, siem)
	if err != nil {
		fatal("create worker: " + err.Error())
	}

	waitContext, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	fmt.Fprintln(os.Stderr, "Waiting for one approved Day 30 job...")
	var result worker.Result
	for {
		result, err = jobWorker.RunOnce(waitContext)
		if err != nil {
			fatal("run one pipeline job: " + err.Error())
		}
		if result.Status != "idle" {
			break
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-waitContext.Done():
			timer.Stop()
			fatal("no approved Day 30 job arrived before the three-minute deadline")
		case <-timer.C:
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fatal("encode worker result: " + err.Error())
	}
	if result.Status == "failed" || result.Status == "rejected" {
		os.Exit(1)
	}
}

func loadConfig() (labConfig, error) {
	executable, err := os.Executable()
	if err != nil {
		return labConfig{}, err
	}
	path := filepath.Join(filepath.Dir(executable), "prooflayer-day30-config.json")
	file, err := os.Open(path)
	if err != nil {
		return labConfig{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maximumConfig+1))
	decoder.DisallowUnknownFields()
	var config labConfig
	if err := decoder.Decode(&config); err != nil {
		return labConfig{}, fmt.Errorf("decode Day 30 config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return labConfig{}, errors.New("Day 30 config contains more than one JSON value")
	}
	if config.SchemaVersion != "1.0" || !config.AllowLabSelfSignedTLS ||
		config.ControlPlaneBaseURL != "https://10.0.2.100:8788" ||
		config.HECEndpoint != "https://10.0.2.100:8088/services/collector/event" ||
		config.SplunkBaseURL != "https://10.0.2.100:8089" {
		return labConfig{}, errors.New("Day 30 config is not bound to the approved isolated lab endpoints")
	}
	return config, nil
}

func isolatedLabHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			// The exception is accepted only because loadConfig binds every target
			// to the three non-routable QEMU guest-forward endpoints above.
			InsecureSkipVerify: true, // #nosec G402 -- isolated lab only
		}},
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
