/**
 * useOrderDetail — global slide-in drawer for inspecting an order (E1).
 * Open from /trade, /history, /portfolio positions, or the command palette.
 *
 *   const od = useOrderDetail()
 *   od.open('ord_8c2a48f1')
 */
export function useOrderDetail() {
  const isOpen  = useState<boolean>('order-detail-open', () => false)
  const orderId = useState<string | null>('order-detail-id', () => null)

  return {
    isOpen,
    orderId,
    open(id: string) {
      orderId.value = id
      isOpen.value  = true
    },
    close() {
      isOpen.value  = false
    },
  }
}
