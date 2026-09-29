package domain

// AccountType is what an account signed up as. It chooses the product surface the web app shows; it
// grants nothing — roles and is_paper decide what an account may do.
type AccountType string

// Account types (platform-core.yaml AccountType).
const (
	AccountTrader     AccountType = "trader"
	AccountAICompany  AccountType = "ai_company"
	AccountEnterprise AccountType = "enterprise"
	AccountDatacenter AccountType = "datacenter"
)

// ParseAccountType validates a signup's account type; empty means the default, ai_company.
func ParseAccountType(s string) (AccountType, bool) {
	switch t := AccountType(s); t {
	case "":
		return AccountAICompany, true
	case AccountTrader, AccountAICompany, AccountEnterprise, AccountDatacenter:
		return t, true
	default:
		return "", false
	}
}
