<template>
  <div class="card">
    <h2 class="question">{{ question.text }}</h2>
    <div class="answers">
      <button
        v-for="(option, index) in question.options"
        :key="index"
        class="answer-btn"
        @click="selectAnswer(index)"
      >
        {{ option }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Question } from "@/types/apiResponses";

defineProps<{
  question: Question;
}>();

const emit = defineEmits<{
  (e: "answerSelected", index: number): void;
}>();

function selectAnswer(index: number) {
  emit("answerSelected", index);
}
</script>

<style scoped>
.card {
  padding: 1.5rem;
  border-radius: 16px;
  background: rgb(54, 1, 78);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}
.question {
  margin: 0 0 1.5rem;
  font-size: 1.3rem;
  line-height: 1.4;
  color: #ffffff;
}
.answers {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.answer-btn {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  border-radius: 12px;
  background: #00d9ff;
  cursor: pointer;
  border: 2px solid transparent;
  transition: all 0.2s;
  width: 100%;
  text-align: left;
  font-size: 1rem;
}
.answer-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  border-color: var(--accent);
}
.letter {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #e5e7eb;
  font-weight: 700;
  flex-shrink: 0;
}
.text {
  flex: 1;
  color: #1f2937;
}
</style>
