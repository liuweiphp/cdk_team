package service

import "testing"

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
