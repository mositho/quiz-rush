//TODO Add a play button if logged in and check change logic based on login status.
<template>
  <main class="home-view">
    <section class="home-view__panel">
      <h1>Quiz Rush</h1>

      <p class="home-view__status">
        <span v-if="authState.initialized && authState.authenticated">
          Signed in as {{ authState.username }}.
        </span>
        <span v-else-if="authState.initialized">Not signed in.</span>
        <span v-else>Checking session...</span>
      </p>

      <div class="home-view__actions">
        <button v-if="!authState.authenticated" type="button" @click="handleLogin">
          Sign in with Keycloak
        </button>
        <button v-else type="button" @click="handleLogout">Sign out</button>
      </div>
    </section>

    <section class="home-view__panel home-view__panel--wide">
      <h2>Backend Auth Check</h2>
      <p class="home-view__hint">
        Public request should return 200. Protected request should return 201 when signed in and 401
        when not signed in.
      </p>

      <div class="home-view__actions home-view__actions--stacked">
        <button type="button" :disabled="loading" @click="loadLeaderboard">
          Test public GET /api/leaderboard/demo
        </button>
        <button type="button" :disabled="loading" @click="createResult">
          Test protected POST /api/results
        </button>
      </div>

      <p v-if="loading" class="home-view__hint">Sending request...</p>

      <div v-if="lastResult" class="home-view__result">
        <p>
          <strong>{{ lastResult.label }}</strong>
        </p>
        <p>Status: {{ lastResult.status }}</p>
        <pre>{{ lastResult.body }}</pre>
      </div>
    </section>
    <section>
      <button @click="startGame">Start Game</button>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";

import { ApiError, apiFetch } from "../services/api";
import { authState, loginWithKeycloak, logoutFromKeycloak } from "../services/keycloak";
import { useGameSession } from "@/composables/useGameSession";

interface LeaderboardResponse {
  packageSlug: string;
  entries: Array<{ player: string; score: number }>;
}

interface CreateResultResponse {
  status: string;
}

interface RequestResult {
  label: string;
  status: number | string;
  body: string;
}

const router = useRouter();
const { session, loading: sessionLoading, startNewSession } = useGameSession();

const loading = ref(false);
const lastResult = ref<RequestResult | null>(null);
function handleLogin() {
  void loginWithKeycloak();
}

function handleLogout() {
  void logoutFromKeycloak();
}

async function loadLeaderboard() {
  loading.value = true;

  try {
    const response = await apiFetch<LeaderboardResponse>("/leaderboard/demo");

    lastResult.value = {
      label: "Public leaderboard request",
      status: 200,
      body: JSON.stringify(response, null, 2),
    };
  } catch (error) {
    lastResult.value = mapError("Public leaderboard request", error);
  } finally {
    loading.value = false;
  }
}

async function createResult() {
  loading.value = true;

  try {
    const response = await apiFetch<CreateResultResponse>("/results", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({}),
    });

    lastResult.value = {
      label: "Protected result submission",
      status: 201,
      body: JSON.stringify(response, null, 2),
    };
  } catch (error) {
    lastResult.value = mapError("Protected result submission", error);
  } finally {
    loading.value = false;
  }
}

function mapError(label: string, error: unknown): RequestResult {
  if (error instanceof ApiError) {
    return {
      label,
      status: error.status,
      body: error.body || error.message,
    };
  }

  return {
    label,
    status: "error",
    body: error instanceof Error ? error.message : "Unknown error",
  };
}

async function startGame() {
  // Placeholder for starting a game session
  await startNewSession({
    durationSeconds: 180,
    selectedQuestionSetIds: ["lf1", "lf2"],
  }).then(() => {
    if (session.value?.sessionId) {
      router.push(`/game/${session.value.sessionId}`);
    }
  });
}
</script>

<style scoped>
.home-view {
  min-height: 100vh;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 420px));
  place-content: center;
  gap: 1.5rem;
  padding: 2rem;
}

.home-view__panel {
  padding: 1.5rem;
  border: 1px solid var(--border);
  border-radius: 1.25rem;
  background: var(--bg);
  color: var(--text);
  box-shadow: var(--shadow);
  text-align: center;
}

.home-view__panel--wide {
  text-align: left;
}

.home-view__panel h1,
.home-view__panel h2,
.home-view__panel strong {
  color: var(--text-h);
}

.home-view__actions button {
  padding: 0.8rem 1.2rem;
  border: 0;
  border-radius: 999px;
  background: var(--accent);
  color: #fff;
  cursor: pointer;
}

.home-view__status,
.home-view__hint {
  margin: 0;
}

.home-view__actions {
  display: flex;
  justify-content: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.home-view__actions--stacked {
  justify-content: flex-start;
}

.home-view__actions button:disabled {
  cursor: progress;
  opacity: 0.7;
}

.home-view__result {
  display: grid;
  gap: 0.5rem;
}

.home-view__result p {
  margin: 0;
}

.home-view__result pre {
  margin: 0;
  padding: 1rem;
  overflow: auto;
  border-radius: 0.75rem;
  background: var(--code-bg);
  color: var(--text-h);
  border: 1px solid var(--border);
}
</style>
