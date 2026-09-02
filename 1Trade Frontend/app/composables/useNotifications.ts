/**
 * useNotifications — shared state for the global Notifications drawer.
 * The drawer itself lives in app/components/App/Notifications.vue and
 * is mounted once inside the `app` layout. Backed by `useState` so SSR
 * doesn't leak state across requests.
 */
export function useNotifications() {
  const isOpen = useState<boolean>('notif-drawer-open', () => false)
  return {
    isOpen,
    open:   () => { isOpen.value = true },
    close:  () => { isOpen.value = false },
    toggle: () => { isOpen.value = !isOpen.value },
  }
}
