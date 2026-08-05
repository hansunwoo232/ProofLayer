package fieldvalidator

import (
	"errors"
	"sort"
)

var ErrInvalidPresenceEvidence = errors.New("invalid field-presence evidence")

var requiredFields = []string{
	"host.name",
	"process.name",
	"process.command_line",
	"user.name",
}

type Status string

const (
	StatusPassed Status = "passed"
	StatusFailed Status = "failed"
)

type Result struct {
	Status             Status   `json:"status"`
	RequiredFieldCount int      `json:"required_field_count"`
	PresentFieldCount  int      `json:"present_field_count"`
	PresentFields      []string `json:"present_fields"`
	MissingFields      []string `json:"missing_fields"`
}

func Validate(presence map[string]bool) (Result, error) {
	allowed := make(map[string]struct{}, len(requiredFields))
	for _, field := range requiredFields {
		allowed[field] = struct{}{}
	}
	for field := range presence {
		if _, ok := allowed[field]; !ok {
			return Result{}, ErrInvalidPresenceEvidence
		}
	}

	result := Result{
		Status:             StatusPassed,
		RequiredFieldCount: len(requiredFields),
		PresentFields:      make([]string, 0, len(requiredFields)),
		MissingFields:      make([]string, 0, len(requiredFields)),
	}
	for _, field := range requiredFields {
		if presence[field] {
			result.PresentFields = append(result.PresentFields, field)
		} else {
			result.MissingFields = append(result.MissingFields, field)
		}
	}
	result.PresentFieldCount = len(result.PresentFields)
	if len(result.MissingFields) != 0 {
		result.Status = StatusFailed
	}
	sort.Strings(result.PresentFields)
	sort.Strings(result.MissingFields)
	return result, nil
}
