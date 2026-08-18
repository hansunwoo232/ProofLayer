package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/httpapi"
	"github.com/hansunwoo232/ProofLayer/control-plane/internal/runqueue"
)

const (
	listenAddress    = "127.0.0.1:8787"
	localOrigin      = "http://127.0.0.1:8787"
	runnerTLSAddress = "127.0.0.1:8788"

	localEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	localHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
	localRunnerID      = "6ba7b812-9dad-41d1-80b4-00c04fd430c8"
	localOperatorID    = "7ba7b811-9dad-41d1-80b4-00c04fd430c8"
	localSigningKeyID  = "local-poc-ed25519-v1"
)

func main() {
	privateKey, err := loadSigningPrivateKey(os.Getenv("PROOFLAYER_SIGNING_SEED"))
	if err != nil {
		log.Fatal("load local signing key: ", err)
	}

	queue, err := runqueue.New(runqueue.Config{
		Capacity:          16,
		EnvironmentID:     localEnvironmentID,
		HostID:            localHostID,
		RequestedBy:       localOperatorID,
		SigningKeyID:      localSigningKeyID,
		SigningPrivateKey: privateKey,
	})
	if err != nil {
		log.Fatal("create local job queue: ", err)
	}

	dashboard, err := findDashboardDirectory()
	if err != nil {
		log.Fatal(err)
	}
	runnerToken := os.Getenv("PROOFLAYER_RUNNER_TOKEN")
	var api *httpapi.Server
	if runnerToken == "" {
		api, err = httpapi.New(queue, localOrigin, dashboard)
	} else {
		api, err = httpapi.NewWithRunner(queue, localOrigin, dashboard, httpapi.RunnerBinding{
			RunnerID:      localRunnerID,
			EnvironmentID: localEnvironmentID,
			HostID:        localHostID,
			BearerToken:   runnerToken,
		})
	}
	if err != nil {
		log.Fatal("create local HTTP API: ", err)
	}
	if runnerToken == "" {
		log.Print("Runner transport disabled: PROOFLAYER_RUNNER_TOKEN is not set")
	} else {
		publicKey := privateKey.Public().(ed25519.PublicKey)
		log.Printf("Runner transport enabled for %s", localRunnerID)
		log.Printf("Runner signing key %s public key: %s", localSigningKeyID, base64.RawURLEncoding.EncodeToString(publicKey))
	}

	server := &http.Server{
		Addr:              listenAddress,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	servers := []*http.Server{server}
	tlsCertificate := os.Getenv("PROOFLAYER_RUNNER_TLS_CERT")
	tlsPrivateKey := os.Getenv("PROOFLAYER_RUNNER_TLS_KEY")
	var runnerTLSServer *http.Server
	if runnerToken != "" && tlsCertificate != "" && tlsPrivateKey != "" {
		runnerTLSServer = &http.Server{
			Addr: runnerTLSAddress, Handler: api.Handler(),
			ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
			WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second,
			MaxHeaderBytes: 16 << 10,
		}
		servers = append(servers, runnerTLSServer)
		log.Printf("Runner TLS transport listening at https://10.0.2.100:8788 through the isolated QEMU forward")
	} else if runnerToken != "" && (tlsCertificate != "" || tlsPrivateKey != "") {
		log.Fatal("PROOFLAYER_RUNNER_TLS_CERT and PROOFLAYER_RUNNER_TLS_KEY must be set together")
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErrors := make(chan error, len(servers))
	go func() { serveErrors <- server.ListenAndServe() }()
	if runnerTLSServer != nil {
		go func() { serveErrors <- runnerTLSServer.ListenAndServeTLS(tlsCertificate, tlsPrivateKey) }()
	}
	log.Printf("ProofLayer local Control Plane listening at %s", localOrigin)
	var serveErr error
	select {
	case <-shutdownContext.Done():
	case serveErr = <-serveErrors:
		stop()
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, activeServer := range servers {
		if err := activeServer.Shutdown(shutdown); err != nil {
			log.Printf("control plane shutdown: %v", err)
		}
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		log.Fatal("serve local Control Plane: ", serveErr)
	}
}

func loadSigningPrivateKey(encodedSeed string) (ed25519.PrivateKey, error) {
	if encodedSeed == "" {
		_, privateKey, err := ed25519.GenerateKey(nil)
		return privateKey, err
	}
	seed, err := base64.RawURLEncoding.DecodeString(encodedSeed)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("PROOFLAYER_SIGNING_SEED must be a base64url-encoded %d-byte seed", ed25519.SeedSize)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func findDashboardDirectory() (string, error) {
	for _, candidate := range []string{"../dashboard", "dashboard"} {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(filepath.Join(absolute, "result-screen-wireframe.html")); err == nil && !info.IsDir() {
			return absolute, nil
		}
	}
	return "", errors.New("dashboard/result-screen-wireframe.html was not found; run from the repository root or control-plane directory")
}
