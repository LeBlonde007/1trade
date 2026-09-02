/**
 * useKyc — read + submit the tenant's identity verification (F22). KYC gates real-money (is_paper=
 * false) credit purchases; sandbox flows never need it. The server is authoritative — this is the
 * convenience surface the buy gate uses to show the right state and submit minimal PII.
 */
export interface KycState {
  status: 'unverified' | 'pending' | 'verified' | 'rejected'
  can_purchase: boolean
  can_submit: boolean
  legal_name?: string
  country?: string
  entity_type?: 'individual' | 'business'
  submitted_at?: string | null
  reviewed_at?: string | null
}

export interface KycSubmission {
  legal_name: string
  country: string
  entity_type: 'individual' | 'business'
}

export function useKyc() {
  const kyc = useState<KycState | null>('account:kyc', () => null)
  const loading = useState('account:kyc:loading', () => false)
  const submitting = useState('account:kyc:submitting', () => false)

  /** load fetches the tenant's current KYC status. */
  async function load() {
    loading.value = true
    try {
      kyc.value = await $fetch<KycState>('/api/account/kyc')
    } finally {
      loading.value = false
    }
    return kyc.value
  }

  /** submit posts an identity-verification submission and stores the resulting state. */
  async function submit(body: KycSubmission) {
    submitting.value = true
    try {
      kyc.value = await $fetch<KycState>('/api/account/kyc', { method: 'POST', body })
      return kyc.value
    } finally {
      submitting.value = false
    }
  }

  return { kyc, loading, submitting, load, submit }
}
