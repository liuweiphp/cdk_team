package service

import (
	"testing"

	"exchange_cdk/model"
)

func TestRedeemCategoryServiceListIncludesJoinedOwnerData(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db, "owner")
	member := createTestUser(t, db, "member")

	team := model.Team{OwnerID: owner.ID, Name: "owner-team"}
	if err := db.Create(&team).Error; err != nil {
		t.Fatalf("create team failed: %v", err)
	}
	if err := db.Create(&model.TeamMember{TeamID: team.ID, MemberID: member.ID}).Error; err != nil {
		t.Fatalf("create team member failed: %v", err)
	}

	svc := NewRedeemCategoryService(db)
	if _, err := svc.Create("账号分类", owner.ID); err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	list, total, err := svc.List(1, 20, "", "active", member.ID)
	if err != nil {
		t.Fatalf("list categories failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(list) != 1 || list[0].CreatedBy != owner.ID {
		t.Fatalf("expected joined owner category, got %#v", list)
	}
}
