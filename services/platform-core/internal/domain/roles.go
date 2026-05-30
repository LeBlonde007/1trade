// Package domain is the pure, IO-free core of platform-core: password hashing, JWT issuance and
// verification, and API-key generation. No database, no HTTP — fully unit-testable. The JWT claims
// here are the contract every other service reads (docs/contracts/openapi/platform-core.yaml
// JwtClaims) — they MUST match what consumers like credit-ledger verify (tenant_id, is_paper, roles).
package domain

// Role is an RBAC role (matches the Role enum in openapi/platform-core.yaml). `trader` is reserved
// for Phase 2 and unused in Phase 1.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleBilling  Role = "billing"
	RoleEngineer Role = "engineer"
	RoleViewer   Role = "viewer"
	RoleTrader   Role = "trader" // Phase 2
)

// validRoles is the allowed set; ValidRole guards against persisting/issuing unknown roles.
var validRoles = map[Role]bool{
	RoleAdmin: true, RoleBilling: true, RoleEngineer: true, RoleViewer: true, RoleTrader: true,
}

// ValidRole reports whether r is a known role.
func ValidRole(r Role) bool { return validRoles[r] }

// HasRole reports whether the held roles satisfy `required`. Admin implicitly satisfies any role
// (full access within its tenant).
func HasRole(held []Role, required Role) bool {
	for _, r := range held {
		if r == required || r == RoleAdmin {
			return true
		}
	}
	return false
}
