package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

const version = "0.1.0-dev"

type selfCheck struct {
	Version                   string `json:"version"`
	Platform                  string `json:"platform"`
	AllowlistedScenarioCount  int    `json:"allowlisted_scenario_count"`
	ArbitraryCommandExecution bool   `json:"arbitrary_command_execution"`
	RegistrationState         string `json:"registration_state"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: prooflayer-runner <version|catalog|self-check|new-correlation>")
	}

	catalog := scenario.BuiltInCatalog()
	switch args[0] {
	case "version":
		fmt.Println(version)
		return nil
	case "catalog":
		return writeJSON(catalog.List())
	case "self-check":
		return writeJSON(selfCheck{
			Version:                   version,
			Platform:                  runtime.GOOS + "/" + runtime.GOARCH,
			AllowlistedScenarioCount:  catalog.Len(),
			ArbitraryCommandExecution: false,
			RegistrationState:         "unregistered",
		})
	case "new-correlation":
		value, err := correlation.Generate()
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(true)
	return encoder.Encode(value)
}
