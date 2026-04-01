<template>
  <div class="card">
    <h2 class="question">{{ question }}</h2>
    <div class="answers">
      <div
        v-for="(ans, idx) in answers"
        :key="ans.id"
        class="answer"
        :class="getClass(idx)"
        @click="$emit('select', { id: idx })"
      >
        <span class="letter">{{ letters[idx] }}</span>
        <span class="text">{{ ans.text }}</span>
        <span v-if="correctIndex === idx" class="icon">✓</span>
        <span v-else-if="selectedIndex === idx && correctIndex !== null" class="icon">✗</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

const props = defineProps<{
  question: string
  answers: { id: number; text: string }[]
  selectedIndex: number | null
  correctIndex: number | null
}>()

defineEmits<{ select: [ans: { id: number }] }>()

const letters = ['A', 'B', 'C', 'D']

const getClass = (idx: number) => ({
  selected: props.selectedIndex === idx && props.correctIndex === null,
  correct: props.correctIndex === idx,
  wrong: props.selectedIndex === idx && props.correctIndex !== null && props.correctIndex !== idx,
})
</script>

<style scoped>
.card {
  padding: 1.5rem;
  border-radius: 16px;
  background: white;
  box-shadow: 0 4px 20px rgba(0,0,0,0.08);
}
.question {
  margin: 0 0 1.5rem;
  font-size: 1.3rem;
  line-height: 1.4;
  color: #111827;
}
.answers { display: flex; flex-direction: column; gap: 0.75rem; }
.answer {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  border-radius: 12px;
  background: #f3f4f6;
  cursor: pointer;
  border: 2px solid transparent;
  transition: all 0.2s;
}
.answer:hover:not(.correct):not(.wrong) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}
.answer.selected { border-color: #3b82f6; background: #dbeafe; }
.answer.correct { border-color: #10b981; background: #d1fae5; }
.answer.wrong { border-color: #ef4444; background: #fee2e2; }
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
.answer.selected .letter { background: #3b82f6; color: white; }
.answer.correct .letter { background: #10b981; color: white; }
.answer.wrong .letter { background: #ef4444; color: white; }
.text { flex: 1; }
.icon { font-weight: bold; font-size: 1.2rem; }
.answer.correct .icon { color: #10b981; }
.answer.wrong .icon { color: #ef4444; }
</style>
