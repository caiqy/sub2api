package service

import "testing"

func TestShouldRejectHiddenPurchasePageAllowsAdmin(t *testing.T) {
	user := &User{HiddenPurchasePage: true}
	if shouldRejectHiddenPurchasePage(user, RoleAdmin) {
		t.Fatal("admin should bypass hidden purchase page denial")
	}
	if !shouldRejectHiddenPurchasePage(user, RoleUser) {
		t.Fatal("regular user should be denied when purchase page is hidden")
	}
}
