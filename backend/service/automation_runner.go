package service

import (
	"encoding/json"
)

type AutomationResult struct {
	Status          string `json:"status"`
	ExternalOrderNo string `json:"external_order_no"`
	SubscribeURL    string `json:"subscribe_url"`
	Error           string `json:"error"`
}

type AutomationRunner struct {
	PythonBin      string
	ScriptPath     string
	TimeoutSeconds int
	MaxRetries     int
}

func NewAutomationRunner(pythonBin, scriptPath string, timeoutSeconds, maxRetries int) *AutomationRunner {
	return &AutomationRunner{
		PythonBin:      pythonBin,
		ScriptPath:     scriptPath,
		TimeoutSeconds: timeoutSeconds,
		MaxRetries:     maxRetries,
	}
}

func (r *AutomationRunner) parseResult(raw []byte) (*AutomationResult, error) {
	var out AutomationResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
