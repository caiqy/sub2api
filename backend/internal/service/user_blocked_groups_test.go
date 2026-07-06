package service

import "testing"

func TestUserCanBindGroupRejectsBlockedPublicGroup(t *testing.T) {
	user := &User{BlockedGroups: []int64{10}, AllowedGroups: []int64{20}}
	if user.CanBindGroup(10, false) {
		t.Fatalf("blocked public group should not be bindable")
	}
	if !user.CanBindGroup(11, false) {
		t.Fatalf("unblocked public group should be bindable")
	}
	if !user.CanBindGroup(20, true) {
		t.Fatalf("allowed exclusive group should be bindable")
	}
	if user.CanBindGroup(21, true) {
		t.Fatalf("unallowed exclusive group should not be bindable")
	}
}
