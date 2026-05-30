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

export function useWallet() {
  const balances = useState<Balance[]>('wallet:balances', () => [])
  const transactions = useState<WalletTx[]>('wallet:tx', () => [])

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

  return { balances, transactions, loadBalances, loadTransactions }
}
