/**
 * usePositionDetail — slide-in drawer for inspecting + closing a position (E5).
 * Trigger from /portfolio positions table or the command palette.
 */
export function usePositionDetail() {
  const isOpen   = useState<boolean>('position-detail-open', () => false)
  const positionId = useState<string | null>('position-detail-id', () => null)

  return {
    isOpen,
    positionId,
    open(id: string) {
      positionId.value = id
      isOpen.value = true
    },
    close() { isOpen.value = false },
  }
}
