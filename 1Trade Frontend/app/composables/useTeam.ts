/**
 * useTeam — team management through the BFF (platform-core v1.6): members, invitations and
 * sub-accounts. Live only; there is no mock data.
 */
export interface Member {
  id: string
  email: string
  roles: string[]
  sub_account_id: string | null
  email_verified: boolean
  created_at: string
}

export interface Invite {
  id: string
  email: string
  roles: string[]
  sub_account_id: string | null
  created_at: string
  expires_at: string
  dev_token?: string
}

export interface SubAccount { id: string; name: string; members: number; created_at: string }

export function useTeam() {
  /** members lists the tenant's members. */
  const members = () => $fetch<{ members: Member[] }>('/api/team/members').then(r => r.members)
  /** invites lists pending invitations (admin). */
  const invites = () => $fetch<{ invites: Invite[] }>('/api/team/invites').then(r => r.invites)
  /** subAccounts lists the tenant's sub-accounts. */
  const subAccounts = () => $fetch<{ sub_accounts: SubAccount[] }>('/api/team/sub-accounts').then(r => r.sub_accounts)
  /** invite invites an email with roles, optionally into a sub-account. */
  const invite = (email: string, roles: string[], subAccountId: string | null) =>
    $fetch<Invite>('/api/team/invites', { method: 'POST', body: { email, roles, sub_account_id: subAccountId } })
  /** revoke cancels a pending invitation. */
  const revoke = (id: string) => $fetch(`/api/team/invites/${id}`, { method: 'DELETE' })
  /** remove removes a member. */
  const remove = (id: string) => $fetch(`/api/team/members/${id}`, { method: 'DELETE' })
  /** place moves a member into a sub-account (null = the main balance). */
  const place = (id: string, subAccountId: string | null) =>
    $fetch(`/api/team/members/${id}/sub-account`, { method: 'PUT', body: { sub_account_id: subAccountId } })
  /** createSubAccount adds a sub-account. */
  const createSubAccount = (name: string) => $fetch<SubAccount>('/api/team/sub-accounts', { method: 'POST', body: { name } })
  /** transfer funds a sub-account or returns credits from it, under a caller-held key. */
  const transfer = (id: string, creditType: string, amount: string, direction: 'fund' | 'return', key: string) =>
    $fetch(`/api/team/sub-accounts/${id}/transfer`, {
      method: 'POST', body: { credit_type: creditType, amount, direction }, headers: { 'Idempotency-Key': key },
    })
  /** lookupInvite reads what an invitation is for (public: the token is the credential). */
  const lookupInvite = (token: string) =>
    $fetch<{ email: string; tenant_name: string; roles: string[]; expires_at: string }>('/api/team/invites/lookup', { query: { token } })
  /** acceptInvite creates the member and signs them in. */
  const acceptInvite = (token: string, password: string) =>
    $fetch('/api/team/invites/accept', { method: 'POST', body: { token, password } })

  return { members, invites, subAccounts, invite, revoke, remove, place, createSubAccount, transfer, lookupInvite, acceptInvite }
}
