package service

import "testing"

func TestTeamAccessAllowsOwnerToModifyOnlyOwnData(t *testing.T) {
	access := TeamAccess{CurrentUserID: 2, SharedOwnerIDs: []uint{1}}

	if !access.CanModify(2) {
		t.Fatal("owner should be allowed to modify own data")
	}
	if access.CanModify(1) {
		t.Fatal("team member should not be allowed to modify shared owner's data")
	}
}

func TestTeamAccessOwnerIDsIncludesCurrentAndJoinedOwners(t *testing.T) {
	access := TeamAccess{CurrentUserID: 2, SharedOwnerIDs: []uint{1, 3, 1}}

	got := access.OwnerIDs()
	want := []uint{2, 1, 3}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
