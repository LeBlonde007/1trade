/**
 * useBilling — credit purchases through the BFF, in US dollars: card or ACH bank debit via Stripe
 * checkout, or a USD wire invoice.
 */
export type PayMethod = 'card' | 'ach' | 'wire'

export interface WireInvoice {
  purchase_id: string
  amount_usd: string
  credits: string
  credit_type: string
  reference: string
  instructions: { bank_name: string; bank_address?: string; account_name: string; account_number: string; routing_number: string; swift?: string }
}

export function useBilling() {
  const loading = useState('billing:loading', () => false)

  /**
   * checkout creates a purchase and returns its Stripe checkout URL. In dev/sandbox a card checkout
   * carries `settled: true` — MockStripe books the credits inline. An ACH debit is never settled at
   * checkout: its credits arrive when the bank clears it.
   */
  async function checkout(amount: string, creditType: string, method: 'card' | 'ach' = 'card') {
    loading.value = true
    try {
      return await $fetch<{ purchase_id: string; checkout_url: string; settled?: boolean }>('/api/billing/checkout', {
        method: 'POST',
        body: { amount, credit_type: creditType, method },
      })
    } finally {
      loading.value = false
    }
  }

  /** wire returns an invoice to pay by USD wire: bank details, exact amount and the reference. */
  async function wire(amount: string, creditType: string) {
    loading.value = true
    try {
      return await $fetch<WireInvoice>('/api/billing/wires', { method: 'POST', body: { amount, credit_type: creditType } })
    } finally {
      loading.value = false
    }
  }

  return { loading, checkout, wire }
}
