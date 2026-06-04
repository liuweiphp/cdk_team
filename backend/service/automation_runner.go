package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type AutomationExecutor interface {
	Run(in AutomationRunInput) (*AutomationResult, error)
}

type AutomationRunInput struct {
	Action           string `json:"action"`
	TaskID           uint   `json:"task_id"`
	AccountName      string `json:"account_name"`
	AccountPrefix    string `json:"account_prefix"`
	TemplateCode     string `json:"template_code"`
	TargetCode       string `json:"target_code"`
	TargetName       string `json:"target_name"`
	Provider         string `json:"provider"`
	ExternalUsername string `json:"external_username"`
	ExternalPassword string `json:"external_password"`
	ExternalOrderNo  string `json:"external_order_no"`
	PayloadJSON      string `json:"payload_json"`
}

type AutomationResult struct {
	Status             string `json:"status"`
	ExternalOrderNo    string `json:"external_order_no"`
	ExternalUsername   string `json:"external_username"`
	ExternalPassword   string `json:"external_password"`
	SubscribeURL       string `json:"subscribe_url"`
	PaymentStatus      string `json:"payment_status"`
	ManualReviewReason string `json:"manual_review_reason"`
	BrowserTracePath   string `json:"browser_trace_path"`
	ScreenshotPath     string `json:"screenshot_path"`
	HTMLDumpPath       string `json:"html_dump_path"`
	PayloadJSON        string `json:"payload_json"`
	Error              string `json:"error"`
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

func (r *AutomationRunner) Run(in AutomationRunInput) (*AutomationResult, error) {
	if r == nil {
		return nil, errors.New("自动化执行器未配置")
	}
	if strings.TrimSpace(r.PythonBin) == "" {
		return nil, errors.New("自动化 Python 未配置")
	}
	if strings.TrimSpace(r.ScriptPath) == "" {
		return nil, errors.New("自动化脚本路径未配置")
	}

	payload, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	attempts := r.MaxRetries + 1
	if attempts <= 0 {
		attempts = 1
	}
	timeoutSeconds := r.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		cmd := exec.CommandContext(ctx, r.PythonBin, r.ScriptPath)
		cmd.Stdin = bytes.NewReader(payload)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		cancel()
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				lastErr = fmt.Errorf("自动化执行超时(%ds)", timeoutSeconds)
			} else if strings.TrimSpace(stderr.String()) != "" {
				lastErr = fmt.Errorf("自动化执行失败: %s", strings.TrimSpace(stderr.String()))
			} else {
				lastErr = fmt.Errorf("自动化执行失败: %w", err)
			}
			continue
		}

		out, err := r.parseResult(stdout.Bytes())
		if err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(out.Status) == "" {
			lastErr = errors.New("自动化返回缺少 status")
			continue
		}
		return out, nil
	}

	return nil, lastErr
}

func (r *AutomationRunner) parseResult(raw []byte) (*AutomationResult, error) {
	var out AutomationResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
