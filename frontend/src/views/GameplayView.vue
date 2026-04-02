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
      <QuestionCard
        v-if="session.currentQuestion"
        :question="session.currentQuestion"
        @select="sendAnswer"
      />    
      <div v-else class="gameplay-view__finished">
        <h2>Game Finished!</h2>
        <p>Final Score: {{ session.currentScore }}</p>
        <p>Correct: {{ session.correctQuestions }} / {{ session.answeredQuestions }}</p>
        <button @click="goHome">Back to Home</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useGameSession } from "@/composables/useGameSession";

const router = useRouter();
const route = useRoute();
const { session, loading, error, loadSession, confirmAnswer } = useGameSession();

onMounted(async () => {
  const sessionId = route.params.sessionId as string;
  if (sessionId) {
    await loadSession(sessionId);
  } else {
    router.push("/");
  }
});

function sendAnswer(index: number) {
  confirmAnswer(index);
}
function goHome() {
  router.push("/");
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
