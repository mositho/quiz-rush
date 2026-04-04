<template>
  <div class="set-selector">
    <button class="set-selector__trigger" type="button" @click="open = !open">
      <span class="stacked-label">
        <span class="stacked-label__title">Question sets</span>
        <span class="stacked-label__value">{{ activeLabel }}</span>
      </span>
      <span class="set-selector__hint">{{ open ? "Tap to collapse" : "Tap to change" }}</span>
    </button>

    <div v-if="loading" class="state-message">Loading available question sets...</div>
    <div v-else-if="error" class="state-message state-message--error">{{ error }}</div>
    <div v-else-if="open" class="set-selector__grid">
      <button
        v-for="questionSet in questionSets"
        :key="questionSet.id"
        class="set-selector__option"
        :class="{ 'set-selector__option--selected': modelValue.includes(questionSet.id) }"
        type="button"
        @click="toggle(questionSet.id)"
      >
        <span class="set-selector__meta">{{ questionSet.length }}</span>
        <span class="set-selector__name">{{ questionSet.name }}</span>
        <span class="set-selector__description">{{
          questionSet.description || "Ready for play"
        }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { QuestionSet } from "@/types/apiResponses";

const props = defineProps<{
  questionSets: QuestionSet[];
  modelValue: string[];
  loading?: boolean;
  error?: string | null;
}>();

const emit = defineEmits<{
  (e: "update:modelValue", value: string[]): void;
}>();

const open = ref(false);
const activeLabel = computed(() => {
  if (props.loading) {
    return "Loading...";
  }

  if (props.error) {
    return "Unavailable";
  }

  if (props.questionSets.length === 0) {
    return "No sets";
  }

  if (props.modelValue.length === 0) {
    return "None selected";
  }

  if (props.modelValue.length === props.questionSets.length) {
    return "All sets";
  }

  if (props.modelValue.length === 1) {
    const selectedSet = props.questionSets.find(
      (questionSet) => questionSet.id === props.modelValue[0]
    );
    return selectedSet?.name ?? "1 set selected";
  }

  return `${props.modelValue.length} sets selected`;
});

function toggle(questionSetId: string) {
  const nextValue = props.modelValue.includes(questionSetId)
    ? props.modelValue.filter((entry) => entry !== questionSetId)
    : [...props.modelValue, questionSetId];

  emit("update:modelValue", nextValue);
}
</script>

<style scoped>
.set-selector {
  display: grid;
  gap: var(--space-4);
}

.set-selector__trigger {
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

.set-selector__hint {
  font-size: 0.9rem;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.set-selector__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
  max-height: 22rem;
  overflow-y: auto;
  padding-right: var(--space-1);
}

.set-selector__option {
  display: grid;
  align-content: start;
  gap: var(--space-2);
  min-height: 8.75rem;
  padding: var(--space-4);
  text-align: left;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  cursor: pointer;
  transition:
    border-color var(--transition-fast),
    background var(--transition-fast),
    color var(--transition-fast);
}

.set-selector__option--selected {
  background: var(--color-primary-soft);
  border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border));
}

.set-selector__name {
  font-weight: 800;
  color: var(--color-heading);
}

.set-selector__meta {
  justify-self: end;
  padding: 0.18rem 0.5rem;
  border-radius: var(--radius-pill);
  background: var(--color-surface-alt);
  color: var(--color-heading);
  font-size: 0.82rem;
  font-weight: 800;
}

.set-selector__description {
  color: var(--color-text-muted);
  font-size: 0.94rem;
}
</style>
