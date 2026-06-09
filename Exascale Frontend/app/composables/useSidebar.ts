/**
 * useSidebar — shared open/close state for the dashboard nav on mobile. On desktop the icon rail is
 * always visible (CSS); at ≤768px it becomes an off-canvas drawer toggled from the topbar hamburger
 * and closed by tapping a nav item or the backdrop.
 */
export function useSidebar() {
  const open = useState('app:sidebar-open', () => false)
  return {
    open,
    toggle: () => { open.value = !open.value },
    close: () => { open.value = false },
  }
}
