<template>
  <main class="home">
    <header class="home__header">
      <h1 class="home__title">Quiz Rush</h1>
      <div class="home__auth">
        <template v-if="!authState.initialized">
          <span class="home__auth-hint">Checking session…</span>
        </template>
        <template v-else-if="authState.authenticated">
          <span class="home__auth-name">{{ authState.username }}</span>
          <button class="btn btn--ghost" type="button" @click="handleLogout">Sign out</button>
        </template>
        <template v-else>
          <button class="btn btn--ghost" type="button" @click="handleLogin">Sign in</button>
        </template>
      </div>
    </header>

    <div class="home__body">
      <!-- Config panel -->
      <section class="card">
        <h2 class="card__title">New Game</h2>

        <label class="field">
          <span class="field__label">Duration</span>
          <select v-model="durationSeconds" class="field__input">
            <option :value="60">1 minute</option>
            <option :value="120">2 minutes</option>
            <option :value="180">3 minutes</option>
            <option :value="300">5 minutes</option>
          </select>
        </label>

        <fieldset class="field" :disabled="setsLoading">
          <legend class="field__label">Question sets</legend>
          <p v-if="setsLoading" class="hint">Loading sets…</p>
          <p v-else-if="setsError" class="hint hint--error">{{ setsError }}</p>
          <label v-for="set in questionSets" :key="set.id" class="checkbox">
            <input type="checkbox" :value="set.id" v-model="selectedSetIds" />
            <span class="checkbox__label"
              >{{ set.name }} <em class="hint">({{ set.length }} questions)</em></span
            >
          </label>
        </fieldset>

        <p v-if="startError" class="hint hint--error">{{ startError }}</p>

        <button
          class="btn btn--primary btn--full"
          type="button"
          :disabled="selectedSetIds.length === 0 || starting"
          @click="handleStart"
        >
          {{ starting ? "Starting…" : "Play" }}
        </button>
      </section>

      <!-- Profile panel (only when signed in) -->
      <section v-if="authState.authenticated" class="card">
        <h2 class="card__title">Profile</h2>
        <p class="home__auth-name">{{ authState.username }}</p>
        <!-- Placeholder for scores/stats — wire up when history page exists -->
        <p class="hint">Score history coming soon.</p>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { useGameSession } from "@/composables/useGameSession";
import { getQuestionSets } from "@/services/api";
import { authState, loginWithKeycloak, logoutFromKeycloak } from "@/services/keycloak";
import type { QuestionSet } from "@/types/apiResponses";
import { onMounted, ref } from "vue";

const { startNewSession } = useGameSession();

const durationSeconds = ref(180);
const selectedSetIds = ref<string[]>([]);
const questionSets = ref<QuestionSet[]>([]);
const setsLoading = ref(false);
const setsError = ref<string | null>(null);
const starting = ref(false);
const startError = ref<string | null>(null);

onMounted(async () => {
  setsLoading.value = true;
  setsError.value = null;
  try {
    questionSets.value = await getQuestionSets();
    // pre-select all sets
    selectedSetIds.value = questionSets.value.map((s) => s.id);
  } catch {
    setsError.value = "Could not load question sets.";
  } finally {
    setsLoading.value = false;
  }
});

function handleLogin() {
  void loginWithKeycloak();
}

function handleLogout() {
  void logoutFromKeycloak();
}

async function handleStart() {
  if (selectedSetIds.value.length === 0) return;
  starting.value = true;
  startError.value = null;
  try {
    await startNewSession({
      durationSeconds: durationSeconds.value,
      selectedQuestionSetIds: selectedSetIds.value,
    });
  } catch {
    startError.value = "Failed to start session. Please try again.";
  } finally {
    starting.value = false;
  }
}
</script>

<style scoped>
.home {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 1.5rem;
  gap: 1.5rem;
  max-width: 860px;
  margin: 0 auto;
}

.home__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.home__title {
  margin: 0;
  font-size: 1.8rem;
  color: var(--text-h);
}

.home__auth {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.home__auth-name {
  font-weight: 600;
  color: var(--text-h);
}

.home__body {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.5rem;
  align-items: start;
}

.card {
  padding: 1.5rem;
  border: 1px solid var(--border);
  border-radius: 1.25rem;
  background: var(--bg);
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.card__title {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text-h);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  border: none;
  padding: 0;
  margin: 0;
}

.field__label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text);
  opacity: 0.7;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.field__input {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  background: var(--bg);
  color: var(--text);
  font-size: 1rem;
}

.checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.95rem;
}

.hint {
  margin: 0;
  font-size: 0.8rem;
  opacity: 0.6;
}

.hint--error {
  color: #ef4444;
  opacity: 1;
}

.btn {
  padding: 0.7rem 1.2rem;
  border: 0;
  border-radius: 999px;
  font-size: 1rem;
  cursor: pointer;
  transition: opacity 0.15s;
}

.btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.btn--primary {
  background: var(--accent);
  color: #fff;
  font-weight: 600;
}

.btn--ghost {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text);
}

.btn--full {
  width: 100%;
}
</style>
