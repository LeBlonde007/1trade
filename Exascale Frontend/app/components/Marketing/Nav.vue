<script setup lang="ts">
/**
 * MarketingNav — sticky top nav for public pages.
 * Ported from Exascale Homepage design.
 *
 * Scroll past 24px → semi-transparent canvas backdrop + blur + bottom border.
 */
const scrolled = ref(false)

onMounted(() => {
  const onScroll = () => { scrolled.value = window.scrollY > 24 }
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })
  onUnmounted(() => window.removeEventListener('scroll', onScroll))
})

const navLinks = [
  { label: 'Markets',     to: '/#markets' },
  { label: 'Methodology', to: '/benchmark' },
  { label: 'Index',       to: '/#index' },
  { label: 'Compute',     to: '/#products' },
  { label: 'Docs',        to: '#' },
  { label: 'About',       to: '#' },
]

const route = useRoute()
const isActive = (to: string) => {
  if (to.startsWith('/#') || to === '#') return false
  return route.path === to
}
</script>

<template>
  <nav class="top" :class="{ scrolled }" aria-label="Primary">
    <div class="container-x">
      <NuxtLink to="/" class="brand">
        <span class="mark" />Exascale
      </NuxtLink>

      <div class="links">
        <NuxtLink
          v-for="l in navLinks"
          :key="l.label"
          :to="l.to"
          class="nav-link"
          :class="{ active: isActive(l.to) }"
        >
          {{ l.label }}
        </NuxtLink>
      </div>

      <div class="cta">
        <NuxtLink to="/login" class="nav-link">Sign in</NuxtLink>
        <BaseButton as="a" :href="'/signup'" variant="primary" size="md">Open account</BaseButton>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.top {
  position: sticky;
  top: 0;
  z-index: 50;
  background-color: transparent;
  border-bottom: 1px solid transparent;
  transition:
    background-color 200ms,
    backdrop-filter  200ms,
    border-color     200ms;
}

.top.scrolled {
  background-color: color-mix(in srgb, var(--canvas) 82%, transparent);
  -webkit-backdrop-filter: saturate(140%) blur(14px);
  backdrop-filter:         saturate(140%) blur(14px);
  border-bottom-color: var(--border);
}

.container-x {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 72px;
  max-width: 1280px;
  margin: 0 auto;
  padding: 0 var(--sp-6);
  gap: var(--sp-6);
}

.brand {
  display: inline-flex;
  align-items: center;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 20px;
  letter-spacing: -0.02em;
  color: var(--text);
  text-decoration: none;
}

.mark {
  display: inline-block;
  width: 14px;
  height: 14px;
  background: var(--brand);
  margin-right: 10px;
  vertical-align: -1px;
}

.links {
  display: flex;
  gap: var(--sp-6);
  align-items: center;
}

.cta {
  display: flex;
  gap: var(--sp-4);
  align-items: center;
}

.nav-link {
  color: var(--text-2);
  font-size: var(--fs-base);
  letter-spacing: -0.005em;
  text-decoration: none;
  transition: color var(--dur) var(--ease);
}

.nav-link:hover { color: var(--text); }

.nav-link.active {
  color: var(--text);
  position: relative;
}
.nav-link.active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -26px;
  height: 2px;
  background: var(--text);
}

@media (max-width: 768px) {
  .links { display: none; }
  .container-x { gap: var(--sp-4); padding: 0 var(--sp-4); }
}
</style>
