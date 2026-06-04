package domain

import "testing"

// TestCanPurchaseRealMoney verifies that only a verified tenant may buy with real money.
func TestCanPurchaseRealMoney(t *testing.T) {
	cases := map[KYCStatus]bool{
		KYCUnverified: false,
		KYCPending:    false,
		KYCRejected:   false,
		KYCVerified:   true,
	}
	for status, want := range cases {
		if got := CanPurchaseRealMoney(status); got != want {
			t.Errorf("CanPurchaseRealMoney(%q) = %v, want %v", status, got, want)
		}
	}
}

// TestCanSubmitKYC verifies submission is allowed from unverified/rejected only (not pending/verified).
func TestCanSubmitKYC(t *testing.T) {
	cases := map[KYCStatus]bool{
		KYCUnverified: true,
		KYCRejected:   true,
		KYCPending:    false,
		KYCVerified:   false,
	}
	for status, want := range cases {
		if got := CanSubmitKYC(status); got != want {
			t.Errorf("CanSubmitKYC(%q) = %v, want %v", status, got, want)
		}
	}
}

// TestValidKYCStatus checks the status whitelist (and that junk is rejected).
func TestValidKYCStatus(t *testing.T) {
	for _, s := range []KYCStatus{KYCUnverified, KYCPending, KYCVerified, KYCRejected} {
		if !ValidKYCStatus(s) {
			t.Errorf("ValidKYCStatus(%q) = false, want true", s)
		}
	}
	if ValidKYCStatus("approved") {
		t.Error("ValidKYCStatus(\"approved\") = true, want false")
	}
}

// TestValidKYCDecision checks that only verified/rejected are legal review outcomes.
func TestValidKYCDecision(t *testing.T) {
	if !ValidKYCDecision(KYCVerified) || !ValidKYCDecision(KYCRejected) {
		t.Error("verified/rejected must be valid decisions")
	}
	if ValidKYCDecision(KYCPending) || ValidKYCDecision(KYCUnverified) {
		t.Error("pending/unverified must not be valid decisions")
	}
}

// TestValidEntityType checks the entity-type whitelist.
func TestValidEntityType(t *testing.T) {
	if !ValidEntityType("individual") || !ValidEntityType("business") {
		t.Error("individual/business must be valid entity types")
	}
	if ValidEntityType("partnership") || ValidEntityType("") {
		t.Error("unknown/empty entity type must be invalid")
	}
}
