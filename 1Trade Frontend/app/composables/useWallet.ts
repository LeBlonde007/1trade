/**
 * useWallet — credit balances + transactions from the BFF (`/api/wallet/*`). All amounts are
 * fixed-point decimal strings; render them mono + tabular-nums per the design system.
 */
export interface Balance {
  credit_type: string
  balance: string
  locked_amount: string
  is_paper: boolean
}

export interface WalletTx {
  tx_id: string
  credit_type: string
  operation: string
  amount: string
  balance_after: string
  is_paper: boolean
  created_at: string
}

/** ConversionRate is one published pair: `rate` is 'to' units per 1 'from' unit, pre-spread. */
export interface ConversionRate {
  from: string
  to: string
  rate: string
}

/** ConversionResult is the ledger's two-leg response: the burned `from` leg + the minted `to` leg. */
export interface ConversionResult {
  debit: WalletTx
  credit: WalletTx
}

export function useWallet() {
  const balances = useState<Balance[]>('wallet:balances', () => [])
  const transactions = useState<WalletTx[]>('wallet:tx', () => [])
  const rates = useState<ConversionRate[]>('wallet:rates', () => [])
  const spread = useState<string>('wallet:spread', () => '0.010000')
  const converting = useState<boolean>('wallet:converting', () => false)

  /** loadBalances fetches current balances. */
  async function loadBalances() {
    balances.value = (await $fetch<{ balances: Balance[] }>('/api/wallet/balances')).balances
    return balances.value
  }

  /** loadTransactions fetches recent ledger transactions. */
  async function loadTransactions() {
    transactions.value = (await $fetch<{ transactions: WalletTx[] }>('/api/wallet/transactions')).transactions
    return transactions.value
  }

  /** loadConversionRates fetches the current per-pair rates + house spread for the convert drawer. */
  async function loadConversionRates() {
    const res = await $fetch<{ spread: string; rates: ConversionRate[] }>('/api/wallet/conversion-rates')
    rates.value = res.rates || []
    spread.value = res.spread || '0.010000'
    return rates.value
  }

  /** rateFor returns the pre-spread decimal rate string for a (from,to) credit-type pair, or null. */
  function rateFor(from: string, to: string): string | null {
    return rates.value.find((r) => r.from === from && r.to === to)?.rate ?? null
  }

  /**
   * convert burns `amount` of `from` and mints the floored target of `to` at the published rate. The
   * idempotency key is minted here (not in the BFF) so a retry of the SAME submit is de-duped by the
   * ledger rather than double-converting. is_paper is derived server-side from the session JWT.
   */
  async function convert(from: string, to: string, amount: string): Promise<ConversionResult> {
    converting.value = true
    try {
      return await $fetch<ConversionResult>('/api/wallet/convert', {
        method: 'POST',
        body: { from, to, amount, rate: rateFor(from, to) ?? undefined, idempotency_key: crypto.randomUUID() },
      })
    } finally {
      converting.value = false
    }
  }

  return {
    balances, transactions, rates, spread, converting,
    loadBalances, loadTransactions, loadConversionRates, rateFor, convert,
  }
}
