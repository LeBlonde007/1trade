/**
 * useSidebar — shared state for the dashboard nav.
 *
 * `open`      mobile only (≤768px): the rail becomes an off-canvas drawer toggled from the
 *             topbar hamburger, closed by tapping a nav item or the backdrop.
 * `collapsed` desktop: the rail shows labels by default and can be collapsed to icons.
 *             Persisted, because it is a workspace preference — a trader on a laptop wants
 *             the chart width back, and should not have to reclaim it every session.
 *
 * Labels are the default deliberately. The rail used to be icon-only at 64px with hover
 * tooltips, for destinations like Markets / Index / Portfolio / History whose icons
 * (List / TrendingUp / Briefcase / Clock) are not self-evident — so finding a screen meant
 * hovering each one in turn.
 */
const STORAGE_KEY = '1t:sidebar-collapsed'

export function useSidebar() {
  const open = useState('app:sidebar-open', () => false)
  const collapsed = useState('app:sidebar-collapsed', () => false)

  // Hydrate the persisted preference once, on the client only.
  if (import.meta.client) {
    try {
      const saved = localStorage.getItem(STORAGE_KEY)
      if (saved !== null) {
        const want = saved === '1'
        if (want !== collapsed.value) collapsed.value = want
      }
    } catch { /* storage blocked — fall back to expanded */ }
  }

  /** Collapse or expand the desktop rail, remembering the choice. */
  function setCollapsed(v: boolean) {
    collapsed.value = v
    if (import.meta.client) {
      try { localStorage.setItem(STORAGE_KEY, v ? '1' : '0') } catch { /* quota */ }
    }
  }

  return {
    open,
    toggle: () => { open.value = !open.value },
    close: () => { open.value = false },
    collapsed,
    setCollapsed,
    toggleCollapsed: () => setCollapsed(!collapsed.value),
  }
}
