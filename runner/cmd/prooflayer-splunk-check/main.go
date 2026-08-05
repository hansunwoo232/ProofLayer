package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hansunwoo232/ProofLayer/runner/internal/splunk"
)

func main() {
	baseURL := flag.String("url", "https://127.0.0.1:8089", "Splunk management URL")
	allowLabCertificate := flag.Bool(
		"allow-local-lab-self-signed",
		false,
		"allow the isolated loopback lab's self-signed certificate",
	)
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "positional arguments are not accepted")
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
	if err := connector.Check(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{
		"status":   "passed",
		"identity": splunk.ObserverUsername,
		"index":    splunk.AllowedIndex,
	})
}
