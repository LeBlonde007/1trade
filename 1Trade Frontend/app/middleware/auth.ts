/**
 * auth — route guard for app pages. Loads the identity from the session cookie (via the BFF) and
 * redirects to /login when there's no session. Apply with `definePageMeta({ middleware: 'auth' })`.
 */
export default defineNuxtRouteMiddleware(async () => {
  const { user, refresh } = useAuth()
  if (!user.value) await refresh()
  if (!user.value) return navigateTo('/login')
})
