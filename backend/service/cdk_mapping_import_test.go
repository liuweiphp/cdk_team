package service

import (
	"strings"
	"testing"
)

func TestParseCodeItemMappings(t *testing.T) {
	input := strings.NewReader("ABCD2345,alpha.txt\nEFGH6789\tbeta.txt\nONLYCODE\n")

	mappings, invalids, err := parseCodeItemMappings(input, "mapping.csv")
	if err != nil {
		t.Fatalf("parse mapping failed: %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}
	if mappings[0].Code != "ABCD2345" || mappings[0].Filename != "alpha.txt" {
		t.Fatalf("unexpected first mapping: %#v", mappings[0])
	}
	if mappings[1].Code != "EFGH6789" || mappings[1].Filename != "beta.txt" {
		t.Fatalf("unexpected second mapping: %#v", mappings[1])
	}
	if len(invalids) != 1 || invalids[0].Line != 3 {
		t.Fatalf("expected line 3 invalid, got %#v", invalids)
	}
}
