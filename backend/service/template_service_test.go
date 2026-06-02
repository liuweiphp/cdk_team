package service

import "testing"

func TestRenderTemplateReplacesContentPlaceholder(t *testing.T) {
	template := "配置链接\n{{content}}\n其他说明"

	got := renderTemplate(template, "https://example.com/sub")
	want := "配置链接\nhttps://example.com/sub\n其他说明"
	if got != want {
		t.Fatalf("unexpected rendered template:\nwant: %q\n got: %q", want, got)
	}
}

func TestGenerateRedeemCodeUsesValidFormat(t *testing.T) {
	code, err := generateRedeemCode()
	if err != nil {
		t.Fatalf("generate code failed: %v", err)
	}
	if len(code) != 16 {
		t.Fatalf("expected code length 16, got %d", len(code))
	}
	if !validCodeChars(code) {
		t.Fatalf("generated invalid code: %s", code)
	}
}
