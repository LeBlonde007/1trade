<script setup lang="ts">
/**
 * BaseButton — the ONLY button in the app.
 * Variants follow the brand book exactly: primary / secondary / tertiary / buy / sell / destructive / dark.
 * Sizes: sm (32px), md (40px default), lg (48px).
 *
 * Use cases:
 *  <BaseButton variant="primary">Open Account</BaseButton>
 *  <BaseButton variant="buy" size="lg" full>Buy 1,000 AI</BaseButton>
 *  <BaseButton variant="tertiary" as="a" href="/docs">Docs →</BaseButton>
 *
 * If you find yourself making a second button, add a variant here instead.
 */
const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'tertiary' | 'buy' | 'sell' | 'destructive' | 'dark'
    size?: 'sm' | 'md' | 'lg'
    /** Render as anchor instead of button (for NuxtLink interop, pass `as="a"` + `href`/`to`). */
    as?: 'button' | 'a'
    full?: boolean
    disabled?: boolean
    type?: 'button' | 'submit' | 'reset'
  }>(),
  {
    variant: 'primary',
    size: 'md',
    as: 'button',
    type: 'button',
    full: false,
    disabled: false,
  },
)
</script>

<template>
  <component
    :is="as"
    class="btn"
    :class="[`btn-${variant}`, `btn-${size}`, { 'btn-full': full, 'btn-disabled': disabled }]"
    :type="as === 'button' ? type : undefined"
    :disabled="as === 'button' ? disabled : undefined"
    :aria-disabled="disabled || undefined"
  >
    <slot />
  </component>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  font-family: var(--font-sans);
  font-weight: 500;
  letter-spacing: -0.005em;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  cursor: pointer;
  white-space: nowrap;
  text-decoration: none;
  transition:
    background-color var(--dur) var(--ease),
    border-color var(--dur) var(--ease),
    color var(--dur) var(--ease);
}

/* Sizes */
.btn-sm { height: 32px; padding: 0 var(--sp-3); font-size: var(--fs-sm); }
.btn-md { height: 40px; padding: 0 var(--sp-5); font-size: var(--fs-base); }
.btn-lg { height: 48px; padding: 0 var(--sp-6); font-size: var(--fs-md); }

.btn-full { width: 100%; }

.btn-disabled { opacity: 0.55; cursor: not-allowed; pointer-events: none; }

/* Variants */
.btn-primary {
  background: var(--brand);
  color: var(--text-on-accent);
}
.btn-primary:hover { background: var(--brand-hov); }

.btn-secondary {
  background: transparent;
  color: var(--text);
  border-color: var(--border-strong);
}
.btn-secondary:hover { background: var(--sunken, var(--hover, var(--elevated))); }

.btn-tertiary {
  background: transparent;
  color: var(--text);
}
.btn-tertiary:hover { background: var(--sunken, var(--hover, transparent)); }

.btn-dark {
  background: var(--text);
  color: var(--canvas);
}
.btn-dark:hover { background: var(--text-2); }

.btn-buy {
  background: var(--pos);
  color: var(--text-on-color);
}
.btn-buy:hover { filter: brightness(1.08); }

.btn-sell {
  background: var(--neg);
  color: var(--text-on-color);
}
.btn-sell:hover { filter: brightness(1.08); }

.btn-destructive {
  background: var(--neg);
  color: var(--text-on-color);
}
.btn-destructive:hover { filter: brightness(1.08); }
</style>
