<script setup lang="ts">
/**
 * MarketingNav — sticky top nav for public pages.
 * Ported from 1Trade Homepage design.
 *
 * Scroll past 24px → semi-transparent canvas backdrop + blur + bottom border.
 */
const scrolled = ref(false)
const mobileOpen = ref(false) // hamburger menu (≤768px)
// The on-page section currently in view (e.g. "#platform") — drives the nav underline (scrollspy).
const activeHash = ref('')
let spy: IntersectionObserver | null = null

// Every entry goes where its label says. The previous set pointed "Docs" at /inference
// (an authenticated app screen, not documentation), "Compute" at a section covering all
// three layers, and "About" at #problem — four of six were same-page anchors, so the nav
// read as a table of contents rather than a site nav. Anchors that remain are labelled as
// the section they actually scroll to.
const navLinks = [
  { label: 'Platform', to: '/#platform' },
  { label: 'Credits',  to: '/#credits' },
  { label: 'API',      to: '/#api' },
  { label: 'Index',    to: '/benchmark' },
  { label: 'Status',   to: '/status' },
]

const route = useRoute()
// Close the mobile menu on any navigation.
watch(() => route.fullPath, () => { mobileOpen.value = false })
// A hash link is active when its section is the one in view on the landing page; a route link is
// active on an exact path match (works on every marketing page).
const isActive = (to: string) => {
  if (to.startsWith('/#')) return route.path === '/' && activeHash.value === to.slice(1)
  return route.path === to
}

onMounted(() => {
  const onScroll = () => { scrolled.value = window.scrollY > 24 }
  onScroll()
  window.addEventListener('scroll', onScroll, { passive: true })

  // Scrollspy — only the landing page has the hash sections. Highlight whichever crosses the
  // viewport's middle band.
  if (route.path === '/' && 'IntersectionObserver' in window) {
    const sections = ['platform', 'credits', 'api', 'exchange']
      .map((id) => document.getElementById(id))
      .filter((el): el is HTMLElement => el !== null)
    if (sections.length) {
      spy = new IntersectionObserver((entries) => {
        const top = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0]
        if (top) activeHash.value = '#' + top.target.id
      }, { rootMargin: '-45% 0px -50% 0px', threshold: [0, 0.25, 0.5, 1] })
      sections.forEach((s) => spy!.observe(s))
    }
  }

  onUnmounted(() => {
    window.removeEventListener('scroll', onScroll)
    spy?.disconnect()
  })
})
</script>

<template>
  <nav class="top" :class="{ scrolled }" aria-label="Primary">
    <div class="container-x">
      <NuxtLink to="/" class="brand">
        <BrandLogo variant="full" size="lg" />
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
        <NuxtLink to="/login" class="nav-link signin">Sign in</NuxtLink>
        <BaseButton as="a" :href="'/signup'" variant="primary" size="md">Open account</BaseButton>
        <button
          class="burger" :class="{ open: mobileOpen }" aria-label="Menu"
          :aria-expanded="mobileOpen" @click="mobileOpen = !mobileOpen"
        >
          <span /><span /><span />
        </button>
      </div>
    </div>

    <!-- Mobile menu — the nav links + sign-in, revealed by the hamburger (≤768px). -->
    <Transition name="sheet">
      <div v-if="mobileOpen" class="mobile-menu">
        <NuxtLink
          v-for="l in navLinks" :key="l.label" :to="l.to"
          class="m-link" :class="{ active: isActive(l.to) }" @click="mobileOpen = false"
        >{{ l.label }}</NuxtLink>
        <NuxtLink to="/login" class="m-link" @click="mobileOpen = false">Sign in</NuxtLink>
      </div>
    </Transition>
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

/* the mark + wordmark come from <BrandLogo>; nothing to style here */

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

/* Hamburger — three bars, hidden on desktop. */
.burger {
  display: none;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  width: 36px;
  height: 36px;
  padding: 0 7px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.burger span {
  display: block;
  height: 2px;
  background: var(--text);
  border-radius: 1px;
  transition: transform 200ms var(--ease), opacity 200ms var(--ease);
}
.burger.open span:nth-child(1) { transform: translateY(6px) rotate(45deg); }
.burger.open span:nth-child(2) { opacity: 0; }
.burger.open span:nth-child(3) { transform: translateY(-6px) rotate(-45deg); }

/* Mobile dropdown sheet */
.mobile-menu {
  display: none;
  flex-direction: column;
  padding: var(--sp-3) var(--sp-6) var(--sp-5);
  background: color-mix(in srgb, var(--canvas) 94%, transparent);
  -webkit-backdrop-filter: saturate(140%) blur(14px);
  backdrop-filter: saturate(140%) blur(14px);
  border-bottom: 1px solid var(--border);
}
.m-link {
  padding: var(--sp-3) 0;
  color: var(--text-2);
  text-decoration: none;
  font-size: var(--fs-md);
  border-bottom: 1px solid var(--border);
}
.m-link:last-child { border-bottom: 0; }
.m-link.active, .m-link:hover { color: var(--text); }

.sheet-enter-active, .sheet-leave-active { transition: opacity 180ms var(--ease), transform 180ms var(--ease); }
.sheet-enter-from, .sheet-leave-to { opacity: 0; transform: translateY(-6px); }

@media (max-width: 768px) {
  .links { display: none; }
  .signin { display: none; } /* moves into the hamburger sheet */
  .burger { display: inline-flex; }
  .mobile-menu { display: flex; }
  .container-x { gap: var(--sp-3); padding: 0 var(--sp-4); }
}
</style>
