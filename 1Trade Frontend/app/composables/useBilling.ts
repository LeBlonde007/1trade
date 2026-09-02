/**
 * useBilling — start a credit purchase via the BFF (`/api/billing/checkout`) and get back a Stripe
 * checkout URL to redirect the customer to.
 */
export function useBilling() {
  const loading = useState('billing:loading', () => false)

  /**
   * checkout creates a purchase and returns its Stripe checkout URL. In dev/sandbox the response
   * carries `settled: true` — MockStripe books the credits inline (no hosted page / webhook), so the
   * caller just refreshes the balance. Real Stripe returns `settled: false` + a hosted checkout_url.
   */
  async function checkout(amount: string, creditType: string, currency = 'usd') {
    loading.value = true
    try {
      return await $fetch<{ purchase_id: string; checkout_url: string; settled?: boolean }>('/api/billing/checkout', {
        method: 'POST',
        body: { amount, credit_type: creditType, currency },
      })
    } finally {
      loading.value = false
    }
  }

  return { loading, checkout }
}
