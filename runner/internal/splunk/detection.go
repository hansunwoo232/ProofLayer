package splunk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
)

const processMarkerDetectionID = "prooflayer.windows_process_marker"

var (
	ErrInvalidDetectionPlan   = errors.New("invalid detection plan")
	ErrInvalidDetectionResult = errors.New("invalid detection result")
	ErrAmbiguousDetection     = errors.New("multiple detection results were found")
)

type DetectionMode string

const (
	DetectionModeInline      DetectionMode = "inline"
	DetectionModeSavedSearch DetectionMode = "saved_search"
)

type DetectionStatus string

const (
	DetectionStatusPassed DetectionStatus = "passed"
	DetectionStatusFailed DetectionStatus = "failed"
)

type DetectionPlan struct {
	detectionID     string
	mode            DetectionMode
	savedSearchName string
}

func BuiltInInlineDetectionPlan() DetectionPlan {
	return DetectionPlan{
		detectionID: processMarkerDetectionID,
		mode:        DetectionModeInline,
	}
}

func BuiltInSavedSearchDetectionPlan() DetectionPlan {
	return DetectionPlan{
		detectionID:     processMarkerDetectionID,
		mode:            DetectionModeSavedSearch,
		savedSearchName: "ProofLayer Windows Process Marker",
	}
}

func (plan DetectionPlan) ID() string {
	return plan.detectionID
}

func (plan DetectionPlan) Mode() DetectionMode {
	return plan.mode
}

func (plan DetectionPlan) Reference() string {
	if plan.mode == DetectionModeSavedSearch {
		return plan.savedSearchName
	}
	return "builtin:" + plan.detectionID
}

func (plan DetectionPlan) Validate() error {
	if plan.detectionID != processMarkerDetectionID {
		return ErrInvalidDetectionPlan
	}
	switch plan.mode {
	case DetectionModeInline:
		if plan.savedSearchName != "" {
			return ErrInvalidDetectionPlan
		}
	case DetectionModeSavedSearch:
		if plan.savedSearchName != "ProofLayer Windows Process Marker" {
			return ErrInvalidDetectionPlan
		}
	default:
		return ErrInvalidDetectionPlan
	}
	return nil
}

type DetectionEvidence struct {
	Status        DetectionStatus `json:"status"`
	Detected      bool            `json:"detected"`
	CorrelationID string          `json:"correlation_id"`
	DetectionID   string          `json:"detection_id"`
	Mode          DetectionMode   `json:"mode"`
	RuleReference string          `json:"rule_reference"`
	MatchCount    int             `json:"match_count"`
}

func (connector *Connector) SearchDetection(
	ctx context.Context,
	correlationID string,
	window SearchWindow,
	plan DetectionPlan,
) (DetectionEvidence, error) {
	if err := plan.Validate(); err != nil {
		return DetectionEvidence{}, err
	}
	if !correlation.Valid(correlationID) {
		return DetectionEvidence{}, ErrInvalidDetectionResult
	}
	if err := window.Validate(); err != nil {
		return DetectionEvidence{}, err
	}

	payload, err := connector.export(ctx, buildDetectionSearch(correlationID, window, plan))
	if err != nil {
		return DetectionEvidence{}, err
	}
	rows, err := decodeDetectionRows(payload)
	if err != nil {
		return DetectionEvidence{}, err
	}
	evidence := DetectionEvidence{
		Status:        DetectionStatusFailed,
		Detected:      false,
		CorrelationID: correlationID,
		DetectionID:   plan.ID(),
		Mode:          plan.Mode(),
		RuleReference: plan.Reference(),
		MatchCount:    len(rows),
	}
	if len(rows) == 0 {
		return evidence, nil
	}
	if len(rows) > 1 {
		return DetectionEvidence{}, ErrAmbiguousDetection
	}
	if rows[0].CorrelationID != correlationID || rows[0].DetectionID != plan.ID() {
		return DetectionEvidence{}, ErrInvalidDetectionResult
	}
	evidence.Status = DetectionStatusPassed
	evidence.Detected = true
	return evidence, nil
}

func buildDetectionSearch(correlationID string, window SearchWindow, plan DetectionPlan) string {
	earliest := strconv.FormatInt(window.Earliest.UTC().Unix(), 10)
	latest := strconv.FormatInt(window.Latest.UTC().Unix(), 10)
	if plan.mode == DetectionModeSavedSearch {
		return `| savedsearch "ProofLayer Windows Process Marker" correlation_id="` + correlationID +
			`" earliest_epoch="` + earliest + `" latest_epoch="` + latest + `"` +
			` | head 2 | table correlation_id,detection_id`
	}
	return `search index=prooflayer_test source="prooflayer:windows-lab" earliest=` + earliest +
		` latest=` + latest + ` "` + correlationID + `"` +
		` | spath` +
		` | eval correlation_id=mvindex(correlation_id,0), event_id=mvindex(event_id,0)` +
		` | where correlation_id="` + correlationID + `" AND event_id=1` +
		` | eval detection_id="` + processMarkerDetectionID + `"` +
		` | head 2 | table correlation_id,detection_id`
}

type detectionRow struct {
	CorrelationID string
	DetectionID   string
}

func decodeDetectionRows(payload []byte) ([]detectionRow, error) {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	rows := make([]detectionRow, 0, 2)
	for {
		var envelope struct {
			Result struct {
				CorrelationID string `json:"correlation_id"`
				DetectionID   string `json:"detection_id"`
			} `json:"result"`
		}
		err := decoder.Decode(&envelope)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrInvalidDetectionResult
		}
		if envelope.Result.CorrelationID == "" && envelope.Result.DetectionID == "" {
			continue
		}
		if !correlation.Valid(envelope.Result.CorrelationID) || envelope.Result.DetectionID != processMarkerDetectionID {
			return nil, ErrInvalidDetectionResult
		}
		rows = append(rows, detectionRow{
			CorrelationID: envelope.Result.CorrelationID,
			DetectionID:   envelope.Result.DetectionID,
		})
	}
	return rows, nil
}
