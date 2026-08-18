package observer

import (
	"encoding/binary"
	"errors"
	"testing"
	"unicode/utf16"
)

func TestParseWindowsEventXMLHandlesBOMAndVaryingDeclarations(t *testing.T) {
	payload := "\ufeff<?xml version='1.0' encoding='utf-8'?>" +
		`<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System>` +
		`<Provider Name="Microsoft-Windows-Sysmon"/><EventID>1</EventID>` +
		`<TimeCreated SystemTime="2026-08-18T19:08:00.7800000Z"/>` +
		`<EventRecordID>4325</EventRecordID></System><EventData>` +
		`<Data Name="Image">C:\Windows\System32\cmd.exe</Data>` +
		`<Data Name="CommandLine">cmd.exe /c echo ` + testCorrelationID + ` &gt;NUL</Data>` +
		`</EventData></Event>` +
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System>` +
		`<Provider Name="Microsoft-Windows-Sysmon"/><EventID>1</EventID>` +
		`<TimeCreated SystemTime="2026-08-18T19:09:00Z"/>` +
		`<EventRecordID>4326</EventRecordID></System><EventData>` +
		`<Data Name="CommandLine">unrelated</Data></EventData></Event>`

	events, err := parseWindowsEventXML([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Provider != "Microsoft-Windows-Sysmon" || events[0].EventID != 1 || events[0].RecordID != 4325 {
		t.Fatalf("first event = %+v", events[0])
	}
	if !contains(events[0].DataValues, testCorrelationID) {
		t.Fatal("correlation ID was not decoded from EventData")
	}
}

func TestParseWindowsEventXMLRejectsUnterminatedDeclaration(t *testing.T) {
	_, err := parseWindowsEventXML([]byte("<?xml version='1.0'"))
	if !errors.Is(err, errInvalidWindowsEventXML) {
		t.Fatalf("error = %v", err)
	}
	if !errors.Is(err, ErrWindowsEventDeclaration) {
		t.Fatalf("error = %v, want declaration classification", err)
	}
}

func TestParseWindowsEventXMLClassifiesInvalidEncoding(t *testing.T) {
	_, err := parseWindowsEventXML([]byte{0xff, 0x00, 0xfe})
	if !errors.Is(err, ErrWindowsEventEncoding) {
		t.Fatalf("error = %v, want encoding classification", err)
	}
}

func TestParseWindowsEventXMLClassifiesMissingDocument(t *testing.T) {
	_, err := parseWindowsEventXML([]byte("not event XML"))
	if !errors.Is(err, ErrWindowsEventDocument) {
		t.Fatalf("error = %v, want document classification", err)
	}
}

func TestParseWindowsEventXMLClassifiesInvalidRecords(t *testing.T) {
	_, err := parseWindowsEventXML([]byte(`<Event><System><EventID>1</EventID></System><EventData><Data>&</Data></EventData></Event>`))
	if !errors.Is(err, ErrWindowsEventRecord) {
		t.Fatalf("error = %v, want record classification", err)
	}
}

func TestParseWindowsEventXMLClassifiesInvalidTimestamps(t *testing.T) {
	payload := `<Event><System><Provider Name="Microsoft-Windows-Sysmon"/><EventID>1</EventID>` +
		`<TimeCreated SystemTime="not-a-time"/><EventRecordID>4325</EventRecordID></System>` +
		`<EventData><Data>` + testCorrelationID + `</Data></EventData></Event>`
	_, err := parseWindowsEventXML([]byte(payload))
	if !errors.Is(err, ErrWindowsEventTimestamp) {
		t.Fatalf("error = %v, want timestamp classification", err)
	}
}

func TestParseWindowsEventXMLHandlesUTF16LEFromRedirectedWevtutil(t *testing.T) {
	payload := `<?xml version="1.0" encoding="utf-16"?><Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System><Provider Name="Microsoft-Windows-Sysmon"/><EventID>1</EventID><TimeCreated SystemTime="2026-08-18T19:08:00.7800000Z"/><EventRecordID>4325</EventRecordID></System><EventData><Data Name="CommandLine">` + testCorrelationID + `</Data></EventData></Event>`
	words := utf16.Encode([]rune(payload))
	raw := make([]byte, 2+len(words)*2)
	raw[0], raw[1] = 0xff, 0xfe
	for index, word := range words {
		binary.LittleEndian.PutUint16(raw[2+index*2:], word)
	}

	events, err := parseWindowsEventXML(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].RecordID != 4325 || !contains(events[0].DataValues, testCorrelationID) {
		t.Fatalf("events = %+v", events)
	}
}

func TestParseWindowsEventXMLHandlesUTF16WithoutBOM(t *testing.T) {
	payload := `<Event><System><Provider Name="Microsoft-Windows-Sysmon"/><EventID>1</EventID>` +
		`<TimeCreated SystemTime="2026-08-18T19:08:00Z"/><EventRecordID>4325</EventRecordID></System>` +
		`<EventData><Data>` + testCorrelationID + `</Data></EventData></Event>`
	words := utf16.Encode([]rune(payload))
	for _, test := range []struct {
		name  string
		order binary.ByteOrder
	}{
		{name: "little endian", order: binary.LittleEndian},
		{name: "big endian", order: binary.BigEndian},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := make([]byte, len(words)*2)
			for index, word := range words {
				test.order.PutUint16(raw[index*2:], word)
			}
			events, err := parseWindowsEventXML(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].RecordID != 4325 || !contains(events[0].DataValues, testCorrelationID) {
				t.Fatalf("events = %+v", events)
			}
		})
	}
}

func TestParseWindowsEventXMLKeepsValidRecordsBesideMalformedRecords(t *testing.T) {
	payload := `<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System>` +
		`<Provider Name="Microsoft-Windows-Sysmon"/><EventID>1</EventID>` +
		`<TimeCreated SystemTime="not-a-time"/><EventRecordID>4324</EventRecordID>` +
		`</System><EventData><Data>unrelated</Data></EventData></Event>` +
		`<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event"><System>` +
		`<Provider Name="Microsoft-Windows-Sysmon"/><EventID>1</EventID>` +
		`<TimeCreated SystemTime="2026-08-18T19:08:00.7800000Z"/><EventRecordID>4325</EventRecordID>` +
		`</System><EventData><Data>` + testCorrelationID + `</Data></EventData></Event>`

	events, err := parseWindowsEventXML([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].RecordID != 4325 || !contains(events[0].DataValues, testCorrelationID) {
		t.Fatalf("events = %+v", events)
	}
}
