package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutomationRunnerParsesPendingPaymentResult(t *testing.T) {
	runner := NewAutomationRunner("python3", "automation/yfjc_runner.py", 120, 2)
	out, err := runner.parseResult([]byte(`{"status":"pending_payment","external_order_no":"A100","subscribe_url":"","error":""}`))
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if out.Status != "pending_payment" || out.ExternalOrderNo != "A100" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestAutomationRunnerRunExecutesScript(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "runner.py")
	script := "import json,sys; payload=json.load(sys.stdin); json.dump({'status':'ready','external_order_no':'ORD-1','subscribe_url':'https://example.com/'+payload['action'],'error':''}, sys.stdout)"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	runner := NewAutomationRunner("python3", scriptPath, 5, 1)
	out, err := runner.Run(AutomationRunInput{
		Action:      "fetch_subscribe",
		TaskID:      7,
		AccountName: "vip-gptplus-0001",
	})
	if err != nil {
		t.Fatalf("run runner: %v", err)
	}
	if out.Status != "ready" {
		t.Fatalf("expected ready status, got %+v", out)
	}
	if out.ExternalOrderNo != "ORD-1" {
		t.Fatalf("unexpected external order no: %+v", out)
	}
	if out.SubscribeURL != "https://example.com/fetch_subscribe" {
		t.Fatalf("unexpected subscribe url: %+v", out)
	}
}
