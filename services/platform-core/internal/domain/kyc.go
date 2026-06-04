package domain

// KYCStatus is a tenant's identity-verification state (F22). It gates real-money credit purchases:
// only a `verified` tenant may buy with real money. Sandbox (paper) flows never consult KYC.
type KYCStatus string

const (
	// KYCUnverified is the default — the tenant has not submitted identity verification.
	KYCUnverified KYCStatus = "unverified"
	// KYCPending means a submission is awaiting review (manual, or instant in dev auto-approve).
	KYCPending KYCStatus = "pending"
	// KYCVerified means identity was confirmed — real-money purchases are permitted.
	KYCVerified KYCStatus = "verified"
	// KYCRejected means a submission was declined; the tenant may resubmit.
	KYCRejected KYCStatus = "rejected"
)

// ValidKYCStatus reports whether s is a known KYC status.
func ValidKYCStatus(s KYCStatus) bool {
	switch s {
	case KYCUnverified, KYCPending, KYCVerified, KYCRejected:
		return true
	default:
		return false
	}
}

// CanPurchaseRealMoney reports whether a tenant in status s may make a real-money (is_paper=false)
// purchase. Only `verified` qualifies — this is the single source of truth the checkout enforces.
func CanPurchaseRealMoney(s KYCStatus) bool { return s == KYCVerified }

// CanSubmitKYC reports whether a tenant in status s may submit (or resubmit) identity verification.
// A tenant can submit from unverified or after a rejection; submitting while pending or already
// verified is rejected (idempotency / no churn).
func CanSubmitKYC(s KYCStatus) bool { return s == KYCUnverified || s == KYCRejected }

// ValidKYCDecision reports whether `to` is a legal terminal review outcome for a pending submission.
func ValidKYCDecision(to KYCStatus) bool { return to == KYCVerified || to == KYCRejected }

// ValidEntityType reports whether t is a supported KYC entity type.
func ValidEntityType(t string) bool { return t == "individual" || t == "business" }
