<template>
  <div class="gameplay-view">
    <div v-if="loading" class="gameplay-view__loading">Loading session...</div>
    <div v-else-if="error" class="gameplay-view__error">{{ error }}</div>

    <div v-else-if="session" class="game-container">
      <header class="gameplay-view__header">
        <div class="gameplay-view__score">Score: {{ session.currentScore }}</div>
        <div class="gameplay-view__progress">
          {{ session.answeredQuestions }} / {{ session.totalQuestions }}
        </div>
      </header>
      <TimerBar :ends-at="session.endsAt" :duration-seconds="session.durationSeconds" />

      <QuestionCard
        v-if="session.currentQuestion"
        :question="session.currentQuestion"
        @answer-selected="sendAnswer"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useGameSession } from "@/composables/useGameSession";
import QuestionCard from "@/components/QuestionCard.vue";
import TimerBar from "@/components/TimerBar.vue";

const router = useRouter();
const route = useRoute();
const { session, loading, error, loadSession, confirmAnswer, answerResult } = useGameSession();

onMounted(async () => {
  const sessionId = route.params.sessionId as string;
  if (sessionId) {
    await loadSession(sessionId);
  } else {
    router.push("/");
  }
});

async function sendAnswer(index: number) {
  await confirmAnswer(index);
  showAnswerFeedback(index, answerResult.value?.correct ?? false);
  setTimeout(() => {
    restGameHub();
  }, 1000);
}

function showAnswerFeedback(index: number, correct: boolean): void {
  const color = correct ? "green" : "red";
  const button = document.querySelectorAll(".answer-btn")[index] as HTMLButtonElement;
  if (correct) {
    const score = document.querySelector(".gameplay-view__score") as HTMLDivElement;
    score.textContent = "Score: " + session.value?.currentScore;
  } else {
    const timerBar = document.querySelector(".timer-bar") as HTMLDivElement;
    timerBar.style.backgroundColor = "red";
  }
  button.style.backgroundColor = color;
}

function restGameHub() {
  if (!session.value) return;
  const answerButtons = document.querySelectorAll(".answer-btn");
  answerButtons.forEach((button) => {
    if (button instanceof HTMLElement) {
      button.style.backgroundColor = "";
    }
  });
  const timerBar = document.querySelector(".timer-bar") as HTMLDivElement;
  if (timerBar) {
    timerBar.style.backgroundColor = "";
  }
  loadSession(session.value.sessionId);
}
</script>

<style scoped>
.gameplay-view {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.gameplay-view__loading,
.gameplay-view__error {
  text-align: center;
  padding: 2rem;
}

.gameplay-view__error {
  color: #ef4444;
}

.game-container {
  width: 100%;
  max-width: 600px;
}

.gameplay-view__header {
  display: flex;
  justify-content: space-between;
  padding: 1rem;
  margin-bottom: 1rem;
  background: var(--bg);
  border-radius: 0.5rem;
}

.gameplay-view__score {
  font-weight: bold;
  font-size: 1.25rem;
}

.gameplay-view__progress {
  color: var(--text);
  opacity: 0.7;
}

.gameplay-view__finished {
  text-align: center;
  padding: 2rem;
  background: var(--bg);
  border-radius: 1rem;
}

.gameplay-view__finished h2 {
  margin-top: 0;
}

.gameplay-view__finished button {
  margin-top: 1rem;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 999px;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
}
</style>
