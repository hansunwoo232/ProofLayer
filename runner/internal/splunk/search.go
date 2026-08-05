package splunk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
)

var (
	ErrInvalidSearchWindow = errors.New("invalid Splunk search window")
	ErrEventNotFound       = errors.New("correlation event was not found in Splunk")
	ErrAmbiguousEvent      = errors.New("multiple correlation events were found in Splunk")
	ErrInvalidSearchResult = errors.New("invalid Splunk correlation result")
)

type SearchWindow struct {
	Earliest time.Time
	Latest   time.Time
}

func (window SearchWindow) Validate() error {
	if window.Earliest.IsZero() || window.Latest.IsZero() || !window.Earliest.Before(window.Latest) {
		return ErrInvalidSearchWindow
	}
	if window.Latest.Sub(window.Earliest) > 24*time.Hour {
		return ErrInvalidSearchWindow
	}
	return nil
}

type CorrelationEvidence struct {
	CorrelationID      string          `json:"correlation_id"`
	Provider           string          `json:"provider"`
	EventID            int             `json:"event_id"`
	RecordID           uint64          `json:"record_id"`
	EndpointEventTime  time.Time       `json:"endpoint_event_time"`
	IngestionLatencyMS int64           `json:"ingestion_latency_ms"`
	FieldPresence      map[string]bool `json:"field_presence"`
}

func (connector *Connector) SearchExact(
	ctx context.Context,
	correlationID string,
	window SearchWindow,
) (CorrelationEvidence, error) {
	if !correlation.Valid(correlationID) {
		return CorrelationEvidence{}, ErrInvalidSearchResult
	}
	if err := window.Validate(); err != nil {
		return CorrelationEvidence{}, err
	}

	search := buildExactSearch(correlationID, window)
	payload, err := connector.export(ctx, search)
	if err != nil {
		return CorrelationEvidence{}, err
	}
	rows, err := decodeCorrelationRows(payload)
	if err != nil {
		return CorrelationEvidence{}, err
	}
	if len(rows) == 0 {
		return CorrelationEvidence{}, ErrEventNotFound
	}
	if len(rows) > 1 {
		return CorrelationEvidence{}, ErrAmbiguousEvent
	}
	if rows[0].CorrelationID != correlationID {
		return CorrelationEvidence{}, ErrInvalidSearchResult
	}
	return rows[0], nil
}

func buildExactSearch(correlationID string, window SearchWindow) string {
	earliest := strconv.FormatInt(window.Earliest.UTC().Unix(), 10)
	latest := strconv.FormatInt(window.Latest.UTC().Unix(), 10)
	return `search index=prooflayer_test source="prooflayer:windows-lab" earliest=` + earliest +
		` latest=` + latest + ` "` + correlationID + `"` +
		` | spath` +
		` | eval correlation_id=mvindex(correlation_id,0), provider=mvindex(provider,0), event_id=mvindex(event_id,0), record_id=mvindex(record_id,0), endpoint_event_time=mvindex(endpoint_event_time,0), host_name=mvindex('host.name',0), process_name=mvindex('process.name',0), process_command_line=mvindex('process.command_line',0), user_name=mvindex('user.name',0)` +
		` | where correlation_id="` + correlationID + `" AND event_id=1` +
		` | eval ingestion_latency_ms=round((_indextime-_time)*1000,0), host_name_present=if(isnull(host_name),0,1), process_name_present=if(isnull(process_name),0,1), process_command_line_present=if(isnull(process_command_line),0,1), user_name_present=if(isnull(user_name),0,1)` +
		` | head 2` +
		` | table correlation_id,provider,event_id,record_id,endpoint_event_time,ingestion_latency_ms,host_name_present,process_name_present,process_command_line_present,user_name_present`
}

func decodeCorrelationRows(payload []byte) ([]CorrelationEvidence, error) {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	rows := make([]CorrelationEvidence, 0, 2)
	for {
		var envelope struct {
			Result struct {
				CorrelationID      string `json:"correlation_id"`
				Provider           string `json:"provider"`
				EventID            string `json:"event_id"`
				RecordID           string `json:"record_id"`
				EndpointEventTime  string `json:"endpoint_event_time"`
				IngestionLatencyMS string `json:"ingestion_latency_ms"`
				HostNamePresent    string `json:"host_name_present"`
				ProcessNamePresent string `json:"process_name_present"`
				CommandLinePresent string `json:"process_command_line_present"`
				UserNamePresent    string `json:"user_name_present"`
			} `json:"result"`
		}
		err := decoder.Decode(&envelope)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrInvalidSearchResult
		}
		if envelope.Result.CorrelationID == "" {
			continue
		}
		eventID, eventErr := strconv.Atoi(envelope.Result.EventID)
		recordID, recordErr := strconv.ParseUint(envelope.Result.RecordID, 10, 64)
		latency, latencyErr := strconv.ParseInt(envelope.Result.IngestionLatencyMS, 10, 64)
		eventTime, timeErr := time.Parse(time.RFC3339Nano, envelope.Result.EndpointEventTime)
		if eventErr != nil || recordErr != nil || latencyErr != nil || timeErr != nil {
			return nil, ErrInvalidSearchResult
		}
		if !correlation.Valid(envelope.Result.CorrelationID) ||
			envelope.Result.Provider != "Microsoft-Windows-Sysmon" || eventID != 1 {
			return nil, ErrInvalidSearchResult
		}
		fieldPresence, presenceErr := decodePresence(envelope.Result.HostNamePresent, envelope.Result.ProcessNamePresent, envelope.Result.CommandLinePresent, envelope.Result.UserNamePresent)
		if presenceErr != nil {
			return nil, ErrInvalidSearchResult
		}
		rows = append(rows, CorrelationEvidence{
			CorrelationID:      envelope.Result.CorrelationID,
			Provider:           envelope.Result.Provider,
			EventID:            eventID,
			RecordID:           recordID,
			EndpointEventTime:  eventTime.UTC(),
			IngestionLatencyMS: latency,
			FieldPresence:      fieldPresence,
		})
	}
	if len(rows) > 2 {
		return nil, fmt.Errorf("%w: result_limit", ErrInvalidSearchResult)
	}
	return rows, nil
}

func decodePresence(host, process, commandLine, user string) (map[string]bool, error) {
	values := map[string]string{
		"host.name":            host,
		"process.name":         process,
		"process.command_line": commandLine,
		"user.name":            user,
	}
	presence := make(map[string]bool, len(values))
	for field, value := range values {
		switch value {
		case "0":
			presence[field] = false
		case "1":
			presence[field] = true
		default:
			return nil, ErrInvalidSearchResult
		}
	}
	return presence, nil
}
