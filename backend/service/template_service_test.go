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

func TestCreateTemplateWithExternalRequiresTargetCode(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "vip")

	svc := NewTemplateService(db)
	_, err := svc.CreateWithExternal("默认模板", "{{content}}", "", "GPT Plus", user.ID)
	if err == nil || err.Error() != "固定购买目标编码不能为空" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestUpdateExternalAccountPrefix(t *testing.T) {
	db := openTestDB(t)
	user := seedTestUser(t, db, "")

	svc := NewUserService(db, 0)
	if err := svc.UpdateExternalAccountPrefix(user.ID, "vip-team"); err != nil {
		t.Fatalf("update external prefix: %v", err)
	}

	fresh, err := svc.GetByID(user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if fresh.ExternalAccountPrefix != "vip-team" {
		t.Fatalf("unexpected prefix: %s", fresh.ExternalAccountPrefix)
	}
}
