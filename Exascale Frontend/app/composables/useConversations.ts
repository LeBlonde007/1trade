/**
 * useConversations — client for server-side chat history (platform-core /v1/conversations via the BFF).
 * Conversations persist the playground transcript (with media stored in object storage and referenced
 * by URL), so history is durable and cross-device. All calls are best-effort: a logged-out user or a
 * backend without the feature degrades to an in-memory session rather than throwing.
 */
export interface ConversationMeta {
  id: string
  title: string
  model: string
  created_at: string
  updated_at: string
}

export interface Conversation extends ConversationMeta {
  transcript: unknown[]
}

export function useConversations() {
  const list = useState<ConversationMeta[]>('conv-list', () => [])

  /** refresh reloads the conversation list (sidebar). Silent on failure (e.g. not signed in). */
  async function refresh(): Promise<void> {
    try {
      const r = await $fetch<{ conversations: ConversationMeta[] }>('/api/conversations')
      list.value = r.conversations ?? []
    } catch { /* not signed in / feature unavailable — keep whatever we have */ }
  }

  /** create persists a new conversation and returns it (with its server id), or null on failure. */
  async function create(title: string, model: string, transcript: unknown[]): Promise<Conversation | null> {
    try {
      return await $fetch<Conversation>('/api/conversations', { method: 'POST', body: { title, model, transcript } })
    } catch { return null }
  }

  /** load fetches one conversation's full transcript, or null if missing/unauthorized. */
  async function load(id: string): Promise<Conversation | null> {
    try { return await $fetch<Conversation>(`/api/conversations/${encodeURIComponent(id)}`) } catch { return null }
  }

  /** save replaces a conversation's transcript (and title/model). Best-effort. */
  async function save(id: string, title: string, model: string, transcript: unknown[]): Promise<void> {
    try { await $fetch(`/api/conversations/${encodeURIComponent(id)}`, { method: 'PUT', body: { title, model, transcript } }) } catch { /* keep session copy */ }
  }

  /** remove deletes a conversation and drops it from the local list. */
  async function remove(id: string): Promise<void> {
    try { await $fetch(`/api/conversations/${encodeURIComponent(id)}`, { method: 'DELETE' }) } catch { /* ignore */ }
    list.value = list.value.filter((c) => c.id !== id)
  }

  return { list, refresh, create, load, save, remove }
}
