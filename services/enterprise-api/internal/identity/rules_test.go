package identity

import "testing"

func TestNormalizePage(t *testing.T) {
	page, size, err := normalizePage(0, 0)
	if err != nil || page != 1 || size != 20 {
		t.Fatalf("page=%d size=%d err=%v", page, size, err)
	}
	if _, _, err := normalizePage(0, 101); err == nil {
		t.Fatal("expected invalid page size")
	}
}

func TestValidateAssignments(t *testing.T) {
	if err := validateAssignments(AssignmentInput{}); err == nil {
		t.Fatal("expected at least one organization")
	}
	err := validateAssignments(AssignmentInput{OrganizationIDs: []string{"a"}, PrimaryOrganizationID: "b"})
	if err == nil {
		t.Fatal("expected primary to be rejected")
	}
	if err := validateAssignments(AssignmentInput{OrganizationIDs: []string{"a", "b"}, PrimaryOrganizationID: "a"}); err != nil {
		t.Fatal(err)
	}
}

func TestParentCreatesCycle(t *testing.T) {
	orgs := []Organization{
		{ID: "root"},
		{ID: "child", ParentID: "root"},
		{ID: "leaf", ParentID: "child"},
	}
	if !parentCreatesCycle(orgs, "root", "leaf") {
		t.Fatal("expected cycle")
	}
	if parentCreatesCycle(orgs, "leaf", "root") {
		t.Fatal("moving a leaf under the root is valid")
	}
	if !parentCreatesCycle(orgs, "child", "child") {
		t.Fatal("expected self parent to be rejected")
	}
}
