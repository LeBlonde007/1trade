<script setup lang="ts">
/**
 * BaseInput — the only text input.
 * Tabular variant right-aligns and uses mono font (for price/quantity inputs).
 *
 * v-model:
 *  <BaseInput v-model="email" label="Email" type="email" />
 *  <BaseInput v-model="qty" label="Quantity" mono />
 *
 * Inline help-dot:
 *  <BaseInput
 *    v-model="price"
 *    label="Limit price"
 *    mono
 *    help-what="The price you're willing to pay (buy) or accept (sell)."
 *    help-why="Use a limit when the spread is wide or you don't want to chase price."
 *    help-field="trade.order.limit-px"
 *  />
 */
const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    label?: string
    placeholder?: string
    type?: string
    /** Numeric (mono + tabular + right-aligned). */
    mono?: boolean
    /** Right-side adornment (unit, symbol). */
    suffix?: string
    /** Disabled state. */
    disabled?: boolean
    /** Show as inline (no label stack). */
    inline?: boolean
    /** Help dot — pass the WHAT to enable. */
    helpWhat?: string
    /** Help dot — optional WHY rationale. */
    helpWhy?: string
    /** Help dot — optional stable id for feedback persistence. */
    helpField?: string
    /** Help dot — override the popover title (defaults to `label`). */
    helpTitle?: string
  }>(),
  {
    type: 'text',
    mono: false,
    disabled: false,
    inline: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const onInput = (e: Event) => {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
}

const popoverTitle = computed(() => props.helpTitle ?? props.label ?? 'Field')
</script>

<template>
  <label class="input" :class="{ mono, inline }">
    <span v-if="label && !inline" class="label-row">
      <span class="label">{{ label }}</span>
      <BaseHelpDot
        v-if="helpWhat"
        :title="popoverTitle"
        :what="helpWhat"
        :why="helpWhy"
        :field="helpField"
        placement="bottom"
      />
    </span>
    <div class="control">
      <input
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        @input="onInput"
      />
      <span v-if="suffix" class="suffix">{{ suffix }}</span>
    </div>
  </label>
</template>

<style scoped>
.input {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
}

.input.inline {
  flex-direction: row;
  align-items: center;
  gap: var(--sp-3);
}

.label-row {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.label {
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  letter-spacing: var(--ls-wide);
  text-transform: uppercase;
  color: var(--text-3);
}

.control {
  position: relative;
  display: flex;
  align-items: stretch;
  background: var(--elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  transition: border-color var(--dur) var(--ease), box-shadow var(--dur) var(--ease);
}

.control:focus-within {
  border-color: var(--brand);
  box-shadow: 0 0 0 2px var(--pos-soft, rgba(200, 242, 92, 0.18));
}

input {
  flex: 1;
  height: 40px;
  padding: 0 var(--sp-4);
  background: transparent;
  border: 0;
  outline: 0;
  color: var(--text);
  font-size: var(--fs-base);
  font-family: var(--font-sans);
}

.mono input {
  font-family: var(--font-mono);
  text-align: right;
  font-variant-numeric: tabular-nums;
}

input::placeholder {
  color: var(--text-3);
}

input:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.suffix {
  display: flex;
  align-items: center;
  padding: 0 var(--sp-4);
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
  color: var(--text-3);
  border-left: 1px solid var(--border);
  background: var(--sunken, transparent);
}
</style>
