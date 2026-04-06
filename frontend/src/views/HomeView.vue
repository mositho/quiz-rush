<template>
  <section class="page home">
    <div class="home__layout">
      <div class="section-stack home__controls">
        <SurfaceCard>
          <TimerSelector v-model="durationSeconds" />
        </SurfaceCard>

        <SurfaceCard>
          <QuestionSetSelector
            v-model="selectedSetIds"
            :question-sets="questionSets"
            :loading="setsLoading"
            :error="setsError"
          />
        </SurfaceCard>
      </div>
    </div>

    <div v-if="startError" class="home__status-banner">
      <p class="state-message state-message--error">{{ startError }}</p>
    </div>

    <div v-if="!isSignedIn" class="home__login-banner">
      <div class="home__login-banner-inner">
        <span class="home__login-hint-text">Login to save your score.</span>
      </div>
    </div>

    <StickyActionBar>
      <button
        class="button button--primary"
        type="button"
        :disabled="selectedSetIds.length === 0 || starting || setsLoading"
        @click="handleStart"
      >
        {{ starting ? "Starting game..." : "Start Game" }}
      </button>
    </StickyActionBar>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import StickyActionBar from "@/components/StickyActionBar.vue";
import QuestionSetSelector from "@/components/QuestionSetSelector.vue";
import SurfaceCard from "@/components/SurfaceCard.vue";
import TimerSelector from "@/components/TimerSelector.vue";
import { useCurrentUser } from "@/composables/useCurrentUser";
import { useGameSession } from "@/composables/useGameSession";
import { getQuestionSets } from "@/services/api";
import type { QuestionSet } from "@/types/apiResponses";
import { DEFAULT_DURATION_SECONDS } from "@/utils/gameConfig";

const router = useRouter();
const { startNewSession } = useGameSession();
const { isSignedIn } = useCurrentUser();

const durationSeconds = ref(DEFAULT_DURATION_SECONDS);
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
    selectedSetIds.value = questionSets.value.map((set) => set.id);
  } catch {
    setsError.value = "Could not load question sets.";
  } finally {
    setsLoading.value = false;
  }
});

async function handleStart() {
  if (selectedSetIds.value.length === 0) {
    return;
  }

  starting.value = true;
  startError.value = null;

  try {
    const createdSession = await startNewSession({
      durationSeconds: durationSeconds.value,
      selectedQuestionSetIds: selectedSetIds.value,
    });

    if (!createdSession) {
      startError.value = "Failed to start session. Please try again.";
      return;
    }

    await router.push(`/game/${createdSession.sessionId}`);
  } catch {
    startError.value = "Failed to start session. Please try again.";
  } finally {
    starting.value = false;
  }
}
</script>

<style scoped>
.home__layout {
  display: grid;
  gap: var(--space-4);
  justify-items: center;
}

.home__controls {
  width: min(100%, 38rem);
}

.home__status-banner {
  width: min(100%, 38rem);
  margin: var(--space-4) auto 0;
}

.home__login-banner {
  width: min(100%, 38rem);
  margin-top: var(--space-4);
  margin-inline: auto;
}

.home__login-banner-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  width: 100%;
  padding: 0.8rem 0.95rem;
  border-radius: var(--radius-xl);
  background: var(--color-warning-soft);
  border: 1px solid color-mix(in srgb, var(--color-warning) 34%, var(--color-border));
  box-shadow: var(--shadow-card);
}

.home__login-hint-text {
  color: var(--color-warning-strong);
  font-weight: 700;
}

@media (max-width: 767px) {
  .home__login-banner-inner {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
