/**
 * useKeys — API-key management via the BFF (`/api/keys`). The created `secret` is exposed once (the
 * caller must show it immediately; it is never retrievable again).
 */
export interface ApiKey {
  id: string
  name: string
  prefix: string
  scopes: string[]
  created_at?: string
  revoked?: boolean
}

export function useKeys() {
  const keys = useState<ApiKey[]>('keys:list', () => [])
  const newSecret = useState('keys:newSecret', () => '')

  /** load fetches the tenant's keys (metadata only). */
  async function load() {
    keys.value = (await $fetch<{ keys: ApiKey[] }>('/api/keys')).keys
    return keys.value
  }

  /** create mints a key and captures the one-time secret. */
  async function create(name: string, scopes: string[]) {
    const r = await $fetch<ApiKey & { secret: string }>('/api/keys', { method: 'POST', body: { name, scopes } })
    newSecret.value = r.secret
    await load()
    return r
  }

  /** revoke deletes a key. */
  async function revoke(id: string) {
    await $fetch(`/api/keys/${id}`, { method: 'DELETE' })
    await load()
  }

  /** dismissSecret clears the one-time secret from memory after the user has copied it. */
  function dismissSecret() {
    newSecret.value = ''
  }

  return { keys, newSecret, load, create, revoke, dismissSecret }
}
