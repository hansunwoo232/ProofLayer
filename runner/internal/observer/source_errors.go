package observer

import "errors"

var (
	ErrEvidenceLimit           = errors.New("Windows event evidence exceeded the local limit")
	ErrWindowsEventQuery       = errors.New("Windows event query failed")
	ErrWindowsEventXML         = errors.New("Windows event XML invalid")
	ErrWindowsEventEncoding    = errors.New("Windows event encoding invalid")
	ErrWindowsEventDeclaration = errors.New("Windows event declaration invalid")
	ErrWindowsEventDocument    = errors.New("Windows event document missing")
	ErrWindowsEventRecord      = errors.New("Windows event records invalid")
	ErrWindowsEventTimestamp   = errors.New("Windows event timestamps invalid")
)
