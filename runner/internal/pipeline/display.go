package pipeline

type DisplayState struct {
	Label  string `json:"label"`
	Tone   string `json:"tone"`
	Symbol string `json:"symbol"`
}

func Display(status StageStatus) (DisplayState, error) {
	switch status {
	case StatusPassed:
		return DisplayState{Label: "PASS", Tone: "positive", Symbol: "check"}, nil
	case StatusFailed:
		return DisplayState{Label: "FAIL", Tone: "critical", Symbol: "x"}, nil
	case StatusNotTested:
		return DisplayState{Label: "NOT TESTED", Tone: "neutral", Symbol: "minus"}, nil
	case StatusDegraded:
		return DisplayState{Label: "DEGRADED", Tone: "warning", Symbol: "warning"}, nil
	case StatusRunning:
		return DisplayState{Label: "RUNNING", Tone: "informative", Symbol: "progress"}, nil
	case StatusPending:
		return DisplayState{Label: "PENDING", Tone: "neutral", Symbol: "clock"}, nil
	default:
		return DisplayState{}, transition("", "unknown_display_status")
	}
}
