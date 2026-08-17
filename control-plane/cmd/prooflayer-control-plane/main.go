package main

import (
	"context"
	"crypto/ed25519"
	"errors"
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
	listenAddress = "127.0.0.1:8787"
	localOrigin   = "http://127.0.0.1:8787"

	localEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	localHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
	localOperatorID    = "7ba7b811-9dad-41d1-80b4-00c04fd430c8"
)

func main() {
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatal("generate local signing key: ", err)
	}

	queue, err := runqueue.New(runqueue.Config{
		Capacity:          16,
		EnvironmentID:     localEnvironmentID,
		HostID:            localHostID,
		RequestedBy:       localOperatorID,
		SigningKeyID:      "local-poc-ephemeral",
		SigningPrivateKey: privateKey,
	})
	if err != nil {
		log.Fatal("create local job queue: ", err)
	}

	dashboard, err := findDashboardDirectory()
	if err != nil {
		log.Fatal(err)
	}
	api, err := httpapi.New(queue, localOrigin, dashboard)
	if err != nil {
		log.Fatal("create local HTTP API: ", err)
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

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		context, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(context); err != nil {
			log.Printf("control plane shutdown: %v", err)
		}
	}()

	log.Printf("ProofLayer local Control Plane listening at %s", localOrigin)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("serve local Control Plane: ", err)
	}
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
