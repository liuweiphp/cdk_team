package service

import "testing"

func TestRedeemServiceRequiresDatabase(t *testing.T) {
	svc := NewRedeemService(nil)

	result, err := svc.RedeemByCode(" ABCD2345 ", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected an error when database is not configured")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
}
