package fieldvalidator

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func allPresent() map[string]bool {
	return map[string]bool{
		"host.name":            true,
		"process.name":         true,
		"process.command_line": true,
		"user.name":            true,
	}
}

func TestValidatePassesAllRequiredFields(t *testing.T) {
	result, err := Validate(allPresent())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusPassed || result.PresentFieldCount != 4 || len(result.MissingFields) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidateFailsMissingProcessCommandLine(t *testing.T) {
	presence := allPresent()
	presence["process.command_line"] = false
	result, err := Validate(presence)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusFailed || len(result.MissingFields) != 1 || result.MissingFields[0] != "process.command_line" {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidateTreatsAbsentEvidenceAsMissing(t *testing.T) {
	result, err := Validate(map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusFailed || len(result.MissingFields) != 4 {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidateRejectsUnknownEvidenceField(t *testing.T) {
	presence := allPresent()
	presence["process.command_line.value"] = true
	if _, err := Validate(presence); !errors.Is(err, ErrInvalidPresenceEvidence) {
		t.Fatalf("error = %v", err)
	}
}

func TestResultSerializationContainsNoRawEndpointValues(t *testing.T) {
	result, err := Validate(allPresent())
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(payload)
	for _, prohibited := range []string{"cmd.exe", "Administrator", "CommandLine", "_raw"} {
		if strings.Contains(serialized, prohibited) {
			t.Fatalf("serialized result contains prohibited value %q", prohibited)
		}
	}
}
