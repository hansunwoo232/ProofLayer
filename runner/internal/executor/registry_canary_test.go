package executor

import "testing"

func TestRegistryCanaryValueNameIsFixedAndCorrelationBound(t *testing.T) {
	valueName, err := registryCanaryValueName("PL-0123456789ABCDEF0123456789ABCDEF")
	if err != nil {
		t.Fatal(err)
	}
	if valueName != "ProofLayer_0123456789ABCDEF0123456789ABCDEF" {
		t.Fatalf("value name = %q", valueName)
	}
}

func TestRegistryCanaryRejectsInvalidCorrelationID(t *testing.T) {
	if _, err := registryCanaryValueName("ProofLayer-user-controlled"); err == nil {
		t.Fatal("invalid correlation ID accepted")
	}
}
