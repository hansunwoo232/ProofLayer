package observer

import (
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

var errInvalidWindowsEventXML = errors.New("invalid Windows event XML")

// parseWindowsEventXML accepts the concatenated XML documents emitted by
// wevtutil. Windows versions can add a UTF-8 BOM and can vary the XML
// declaration, so declarations are removed structurally instead of matching
// one exact byte sequence.
func parseWindowsEventXML(raw []byte) ([]Event, error) {
	payload, err := decodeWindowsEventXML(raw)
	if err != nil {
		return nil, err
	}
	payload = strings.ReplaceAll(payload, "\ufeff", "")
	for {
		start := strings.Index(payload, "<?")
		if start < 0 {
			break
		}
		endOffset := strings.Index(payload[start+2:], "?>")
		if endOffset < 0 {
			return nil, fmt.Errorf("%w: %w", errInvalidWindowsEventXML, ErrWindowsEventDeclaration)
		}
		end := start + 2 + endOffset + 2
		payload = payload[:start] + payload[end:]
	}

	documents := eventDocuments(payload)
	if len(documents) == 0 {
		if strings.TrimSpace(payload) == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %w", errInvalidWindowsEventXML, ErrWindowsEventDocument)
	}
	events := make([]Event, 0, len(documents))
	parsedRecords := 0
	for _, document := range documents {
		var parsed xmlEvent
		if err := xml.Unmarshal([]byte(document), &parsed); err != nil {
			// A malformed unrelated record must not prevent a valid, bounded
			// correlation event in the same wevtutil response from being used.
			continue
		}
		parsedRecords++
		createdAt, err := time.Parse(time.RFC3339Nano, parsed.System.TimeCreated.SystemTime)
		if err != nil {
			continue
		}
		values := make([]string, 0, len(parsed.EventData))
		for _, data := range parsed.EventData {
			values = append(values, data.Value)
		}
		events = append(events, Event{
			Provider:       parsed.System.Provider.Name,
			EventID:        parsed.System.EventID,
			RecordID:       parsed.System.RecordID,
			TimeCreatedUTC: createdAt.UTC(),
			DataValues:     values,
		})
	}
	if len(events) == 0 {
		if parsedRecords == 0 {
			return nil, fmt.Errorf("%w: %w", errInvalidWindowsEventXML, ErrWindowsEventRecord)
		}
		return nil, fmt.Errorf("%w: %w", errInvalidWindowsEventXML, ErrWindowsEventTimestamp)
	}
	return events, nil
}

func eventDocuments(payload string) []string {
	result := make([]string, 0)
	for cursor := 0; cursor < len(payload); {
		relativeStart := strings.Index(payload[cursor:], "<Event")
		if relativeStart < 0 {
			break
		}
		start := cursor + relativeStart
		afterName := start + len("<Event")
		if afterName >= len(payload) || (payload[afterName] != '>' && payload[afterName] != ' ' && payload[afterName] != '\t' && payload[afterName] != '\r' && payload[afterName] != '\n') {
			cursor = afterName
			continue
		}
		relativeEnd := strings.Index(payload[afterName:], "</Event>")
		if relativeEnd < 0 {
			break
		}
		end := afterName + relativeEnd + len("</Event>")
		result = append(result, payload[start:end])
		cursor = end
	}
	return result
}

func decodeWindowsEventXML(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	if len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf {
		raw = raw[3:]
		if !utf8.Valid(raw) {
			return "", fmt.Errorf("%w: %w", errInvalidWindowsEventXML, ErrWindowsEventEncoding)
		}
		return string(raw), nil
	}
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe {
		return decodeUTF16(raw[2:], binary.LittleEndian)
	}
	if len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff {
		return decodeUTF16(raw[2:], binary.BigEndian)
	}
	// wevtutil /uni:true normally emits a UTF-16 BOM. Preserve support for
	// redirected or wrapped invocations that remove it before the Runner sees
	// the bytes by recognizing the characteristic alternating NUL pattern.
	if order, ok := bomlessUTF16Order(raw); ok {
		return decodeUTF16(raw, order)
	}
	if utf8.Valid(raw) {
		return string(raw), nil
	}
	return "", fmt.Errorf("%w: %w", errInvalidWindowsEventXML, ErrWindowsEventEncoding)
}

func bomlessUTF16Order(raw []byte) (binary.ByteOrder, bool) {
	if len(raw) < 8 || len(raw)%2 != 0 {
		return nil, false
	}
	pairs := len(raw) / 2
	if pairs > 64 {
		pairs = 64
	}
	evenNUL, oddNUL := 0, 0
	for index := 0; index < pairs; index++ {
		if raw[index*2] == 0 {
			evenNUL++
		}
		if raw[index*2+1] == 0 {
			oddNUL++
		}
	}
	threshold := pairs * 2 / 3
	maximumOpposite := pairs / 8
	switch {
	case oddNUL >= threshold && evenNUL <= maximumOpposite:
		return binary.LittleEndian, true
	case evenNUL >= threshold && oddNUL <= maximumOpposite:
		return binary.BigEndian, true
	default:
		return nil, false
	}
}

func decodeUTF16(raw []byte, order binary.ByteOrder) (string, error) {
	if len(raw)%2 != 0 {
		return "", fmt.Errorf("%w: %w", errInvalidWindowsEventXML, ErrWindowsEventEncoding)
	}
	words := make([]uint16, len(raw)/2)
	for index := range words {
		words[index] = order.Uint16(raw[index*2 : index*2+2])
	}
	return string(utf16.Decode(words)), nil
}

type xmlEvent struct {
	System struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
		EventID     int    `xml:"EventID"`
		RecordID    uint64 `xml:"EventRecordID"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
	} `xml:"System"`
	EventData []struct {
		Value string `xml:",chardata"`
	} `xml:"EventData>Data"`
}
