/**
 * useCommandPalette — shared open-state for the global Cmd+K palette.
 * Backed by `useState` so SSR doesn't leak state across requests.
 */
export function useCommandPalette() {
  const isOpen = useState<boolean>('cmd-palette-open', () => false)
  return {
    isOpen,
    open:   () => { isOpen.value = true },
    close:  () => { isOpen.value = false },
    toggle: () => { isOpen.value = !isOpen.value },
  }
}
