/**
 * useTxDetail — transaction-detail drawer (F4).
 * Opened from any ledger row in /wallet, recurring /wallet, or audit deep-links.
 */
export function useTxDetail() {
  const isOpen = useState<boolean>('tx-detail-open', () => false)
  const txId   = useState<string | null>('tx-detail-id', () => null)
  return {
    isOpen,
    txId,
    open(id: string) { txId.value = id; isOpen.value = true },
    close() { isOpen.value = false },
  }
}
