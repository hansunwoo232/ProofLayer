package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/splunk"
)

func main() {
	baseURL := flag.String("url", "https://127.0.0.1:8089", "Splunk management URL")
	correlationID := flag.String("correlation-id", "", "canonical ProofLayer correlation ID")
	windowMinutes := flag.Int("window-minutes", 60, "bounded search window in minutes")
	pollSeconds := flag.Int("poll-seconds", 2, "polling interval in seconds")
	timeoutSeconds := flag.Int("timeout-seconds", 60, "overall polling timeout in seconds")
	allowLabCertificate := flag.Bool(
		"allow-local-lab-self-signed",
		false,
		"allow the isolated loopback lab's self-signed certificate",
	)
	flag.Parse()
	if flag.NArg() != 0 || *correlationID == "" ||
		*windowMinutes < 1 || *windowMinutes > 1440 ||
		*pollSeconds < 1 || *pollSeconds > 5 ||
		*timeoutSeconds < *pollSeconds || *timeoutSeconds > 60 {
		fmt.Fprintln(os.Stderr, "invalid correlation, window, polling, or timeout option")
		os.Exit(2)
	}

	password, err := splunk.ObserverPasswordFromEnvironment()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if *allowLabCertificate {
		if *baseURL != "https://127.0.0.1:8089" {
			fmt.Fprintln(os.Stderr, "the self-signed exception is restricted to the loopback lab")
			os.Exit(2)
		}
		tlsConfig.InsecureSkipVerify = true // Isolated loopback lab only; never a customer default.
	}
	connector, err := splunk.New(splunk.Config{
		BaseURL:  *baseURL,
		Username: splunk.ObserverUsername,
		Password: password,
		TLS:      tlsConfig,
	}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	policy := splunk.PollingPolicy{
		Timeout:         time.Duration(*timeoutSeconds) * time.Second,
		Interval:        time.Duration(*pollSeconds) * time.Second,
		MaximumAttempts: *timeoutSeconds / *pollSeconds,
	}
	latest := time.Now().UTC().Add(time.Minute)
	evidence, attempts, err := splunk.PollExact(
		context.Background(),
		connector,
		*correlationID,
		splunk.SearchWindow{Earliest: latest.Add(-time.Duration(*windowMinutes) * time.Minute), Latest: latest},
		policy,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		Status   string                     `json:"status"`
		Attempts int                        `json:"attempts"`
		Evidence splunk.CorrelationEvidence `json:"evidence"`
	}{Status: "passed", Attempts: attempts, Evidence: evidence})
}
