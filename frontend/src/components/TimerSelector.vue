<template>
  <div class="timer-selector">
    <button class="timer-selector__trigger" type="button" @click="open = !open">
      <span class="stacked-label">
        <span class="stacked-label__title">Timer</span>
        <span class="stacked-label__value">{{ activeLabel }}</span>
      </span>
      <span class="timer-selector__hint">Tap to change</span>
    </button>

    <div v-if="open" class="timer-selector__options">
      <button
        v-for="preset in presets"
        :key="preset"
        class="timer-selector__option"
        :class="{ 'timer-selector__option--active': modelValue === preset }"
        type="button"
        @click="selectPreset(preset)"
      >
        {{ formatDurationLabel(preset) }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { TIMER_PRESETS, formatDurationLabel } from "@/utils/gameConfig";

const props = defineProps<{
  modelValue: number;
}>();

const emit = defineEmits<{
  (e: "update:modelValue", value: number): void;
}>();

const open = ref(false);
const presets = [...TIMER_PRESETS];
const activeLabel = computed(() => formatDurationLabel(props.modelValue));

function selectPreset(value: number) {
  emit("update:modelValue", value);
}
</script>

<style scoped>
.timer-selector {
  display: grid;
  gap: var(--space-3);
}

.timer-selector__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  width: 100%;
  padding: 0;
  text-align: left;
  background: transparent;
  color: var(--color-heading);
  cursor: pointer;
}

.timer-selector__hint {
  font-size: 0.9rem;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.timer-selector__options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
}

.timer-selector__option {
  padding: 0.85rem 1rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-heading);
  cursor: pointer;
  transition:
    transform var(--transition-fast),
    background var(--transition-fast);
}

.timer-selector__option--active {
  background: var(--color-primary-soft);
  border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border));
}
</style>
