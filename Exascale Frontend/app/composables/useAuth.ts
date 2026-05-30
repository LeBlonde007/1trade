/**
 * useAuth — session identity + auth actions, backed by the BFF (`/api/auth/*`). The JWT lives in an
 * httpOnly cookie the BFF manages; the client only ever sees the identity. Shared via useState so the
 * nav, guards, and pages read one source of truth.
 */
export interface Identity {
  user_id: string
  email: string
  tenant_id: string
  org_id?: string
  roles: string[]
  is_paper: boolean
}

export function useAuth() {
  const user = useState<Identity | null>('auth:user', () => null)

  /** login verifies credentials and loads the identity. */
  async function login(email: string, password: string) {
    user.value = await $fetch<Identity>('/api/auth/login', { method: 'POST', body: { email, password } })
    return user.value
  }

  /** signup creates a tenant + admin user and loads the identity. */
  async function signup(email: string, password: string, tenant_name?: string) {
    user.value = await $fetch<Identity>('/api/auth/signup', { method: 'POST', body: { email, password, tenant_name } })
    return user.value
  }

  /** refresh loads the current identity from the session cookie (null when logged out). */
  async function refresh() {
    try {
      user.value = await $fetch<Identity>('/api/auth/me')
    } catch {
      user.value = null
    }
    return user.value
  }

  /** logout clears the session. */
  async function logout() {
    await $fetch('/api/auth/logout', { method: 'POST' })
    user.value = null
  }

  return { user, login, signup, refresh, logout }
}
