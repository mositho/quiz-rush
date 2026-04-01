<template>
  <div class="gameplay">
    <h1>Gameplay</h1>

    <div v-if="loading">Loading questions…</div>
    <div v-else-if="error">Error: {{ error }}</div>
    <div v-else>
      <div v-if="currentQuestion">
        <QuestionCard
          :question="currentQuestion.Question"
          :answers="currentAnswers"
          @select="onSelect"
        />

        <div class="controls">
          <button v-if="!answered" class="next-btn" disabled>Choose an answer</button>
        </div>

        <div class="feedback" v-if="answered">
          <p v-if="lastCorrect">Correct!</p>
          <p v-else>Wrong — correct answer: {{ correctText }}</p>
        </div>
      </div>

      <div v-if="finished" class="finished">
        <h2>Finished</h2>
        <p>Score: {{ score }} / {{ questions.length }}</p>
        <button @click="restart">Play again</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import { apiFetch } from "../services/api";
import QuestionCard from "../components/QuestionCard.vue";

interface Question {
question: string;
options: string[];
correctAnswer: number;
}
const route = useRoute();
const setId = String(route.params.setId || "1");
const loading = ref(true);
const error = ref<string | null>(null);
const questions = ref<Question[]>([]);
const currentIndex = ref(0);
const score = ref(0);
const answered = ref(false);
const correct = ref(false);
onMounted(async () => {
  try {
    questions.value = await apiFetch(`/sets/${setId}`);
  } catch (e) {
    error.value = "Failed to load questions";
  } finally {
    loading.value = false;
  }
});
const currentQuestion = computed(() => questions.value[currentIndex.value]);
const currentAnswers = computed(() =>
  currentQuestion.value?.Options.map((text, i) => ({ id: i, text })) || []
);
const correctText = computed(() => currentQuestion.value?.Options[currentQuestion.value.CorrectAnswer]);
function onSelect(ans: { id: number }) {
  if (answered.value) return;
  answered.value = true;
  correct.value = ans.id === currentQuestion.value.CorrectAnswer;
  if (correct.value) score.value++;
}
function nextQuestion() {
  answered.value = false;
  correct.value = false;
  currentIndex.value++;
}
const finished = computed(() => currentIndex.value >= questions.value.length);
function restart() {
  currentIndex.value = 0;
  score.value = 0;
  answered.value = false;
  correct.value = false;
}

</script>

<style scoped>
.gameplay {
  padding: 1rem;
}
.next-btn {
  margin-top: 0.6rem;
}
.feedback {
  margin-top: 0.6rem;
}
.finished {
  margin-top: 1rem;
}
</style>
