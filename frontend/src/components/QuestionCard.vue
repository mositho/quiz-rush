<template>
  <div class="question-card">
    <div class="question-card__prompt">
      <h2 class="question-card__question">{{ question.text }}</h2>
    </div>

    <div class="question-card__answers">
      <button
        v-for="(option, index) in question.options"
        :key="index"
        class="question-card__answer"
        :class="answerClasses(index)"
        :disabled="disabled"
        type="button"
        @click="emit('answerSelected', index)"
      >
        <span class="question-card__index">{{ answerIndexLabel(index) }}</span>
        <span>{{ option }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Question } from "@/types/apiResponses";

const props = defineProps<{
  question: Question;
  disabled?: boolean;
  selectedIndex?: number | null;
  wasCorrect?: boolean | null;
}>();

const emit = defineEmits<{
  (e: "answerSelected", index: number): void;
}>();

function answerIndexLabel(index: number) {
  return String.fromCharCode(65 + index);
}

function answerClasses(index: number) {
  const classes = [];

  if (props.selectedIndex === index) {
    classes.push("question-card__answer--selected");
  }

  if (props.selectedIndex === index && props.wasCorrect === true) {
    classes.push("question-card__answer--correct");
  }

  if (props.selectedIndex === index && props.wasCorrect === false) {
    classes.push("question-card__answer--wrong");
  }

  return classes;
}
</script>

<style scoped>
.question-card {
  display: grid;
  gap: var(--space-3);
}

.question-card__prompt {
  display: grid;
}

.question-card__question {
  margin: 0;
  font-size: clamp(1.35rem, 2.8vw, 2rem);
  line-height: 1.15;
  color: var(--color-heading);
  width: 100%;
}

.question-card__answers {
  display: grid;
  gap: var(--space-3);
  grid-auto-rows: auto;
}

.question-card__answer {
  display: flex;
  align-items: center;
  align-self: start;
  gap: var(--space-3);
  height: auto;
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  color: var(--color-heading);
  text-align: left;
  cursor: pointer;
  transition:
    transform var(--transition-fast),
    border-color var(--transition-fast),
    background var(--transition-fast);
}

.question-card__answer:hover:not(:disabled) {
  transform: translateY(-1px);
}

.question-card__answer:disabled {
  cursor: default;
}

.question-card__answer--selected {
  border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border));
}

.question-card__answer--correct {
  background: var(--color-success-soft);
  border-color: color-mix(in srgb, var(--color-success) 55%, var(--color-border));
}

.question-card__answer--wrong {
  background: var(--color-danger-soft);
  border-color: color-mix(in srgb, var(--color-danger) 55%, var(--color-border));
}

.question-card__index {
  display: inline-flex;
  flex: 0 0 2.2rem;
  align-items: center;
  justify-content: center;
  width: 2.2rem;
  min-width: 2.2rem;
  height: 2.2rem;
  min-height: 2.2rem;
  border-radius: 0.8rem;
  background: var(--color-surface-strong);
  font-weight: 800;
}

@media (min-width: 768px) {
  .question-card__answers {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    grid-auto-rows: 1fr;
  }

  .question-card__answer {
    align-self: stretch;
    height: 100%;
    min-height: 6.5rem;
  }
}
</style>
