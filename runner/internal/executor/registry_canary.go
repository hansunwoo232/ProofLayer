package executor

import (
	"context"
	"errors"
	"strings"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
)

const (
	registryCanaryKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	registryCanaryPrefix  = "ProofLayer_"
	registryCanaryData    = `C:\Windows\System32\cmd.exe /d /c exit 0`
)

var ErrArtifactPresent = errors.New("scenario artifact remains")

func RunRegistryCanary(ctx context.Context, correlationID string) error {
	valueName, err := registryCanaryValueName(correlationID)
	if err != nil {
		return err
	}
	return runRegistryCanary(ctx, valueName)
}

func CleanupRegistryCanary(ctx context.Context, correlationID string) error {
	valueName, err := registryCanaryValueName(correlationID)
	if err != nil {
		return err
	}
	return cleanupRegistryCanary(ctx, valueName)
}

func VerifyRegistryCanaryAbsent(ctx context.Context, correlationID string) error {
	valueName, err := registryCanaryValueName(correlationID)
	if err != nil {
		return err
	}
	return verifyRegistryCanaryAbsent(ctx, valueName)
}

func registryCanaryValueName(correlationID string) (string, error) {
	if !correlation.Valid(correlationID) {
		return "", errors.New("invalid correlation ID")
	}
	return registryCanaryPrefix + strings.TrimPrefix(correlationID, "PL-"), nil
}

func registryCanaryHandler() ScenarioHandler {
	return handlerFunctions{
		execute:      RunRegistryCanary,
		cleanup:      CleanupRegistryCanary,
		verifyAbsent: VerifyRegistryCanaryAbsent,
	}
}
