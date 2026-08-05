package splunk

import (
	"errors"
	"os"
)

const ObserverPasswordEnvironment = "PROOFLAYER_OBSERVER_PASSWORD"

var ErrCredentialUnavailable = errors.New("Splunk observer credential is unavailable")

func ObserverPasswordFromEnvironment() (string, error) {
	password := os.Getenv(ObserverPasswordEnvironment)
	if len(password) < 16 {
		return "", ErrCredentialUnavailable
	}
	return password, nil
}
