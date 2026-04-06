<template>
  <section class="page">
    <div v-if="needsLogin" class="section-stack">
      <SurfaceCard>
        <p class="page__subtitle">Sign in to view your own profile.</p>
        <div class="inline-actions">
          <button class="button button--primary" type="button" @click="handleLogin">Login</button>
          <button class="button button--secondary" type="button" @click="handleRegister">
            Register
          </button>
        </div>
      </SurfaceCard>
    </div>

    <div v-else class="section-stack">
      <div v-if="loading" class="state-message">Loading profile...</div>
      <div v-else-if="error" class="state-message state-message--error">{{ error }}</div>

      <template v-if="!loading && !error && statsProfile && scoreList">
        <SurfaceCard v-if="canEditProfile" class="profile__update-row">
          <div class="profile__update-field">
            <div class="profile__update-label-row">
              <span class="stacked-label__title">{{ DISPLAY_NAME_LABEL }}</span>
              <p
                v-if="displayNameError || displayNameSuccess"
                class="profile__update-feedback"
                :class="
                  displayNameError
                    ? 'state-message state-message--error'
                    : 'state-message state-message--success'
                "
              >
                {{ displayNameError || displayNameSuccess }}
              </p>
            </div>
            <div class="profile__update-input-row">
              <input
                v-model="displayNameDraft"
                class="profile__name-input"
                type="text"
                maxlength="40"
                autocomplete="nickname"
                :placeholder="statsProfile.displayName"
              />
              <button
                class="button button--primary button--compact profile__update-action"
                type="button"
                :disabled="displayNameSubmitting || !displayNameChanged"
                @click="handleUpdateDisplayName"
              >
                {{ displayNameSubmitting ? "Updating..." : "Update" }}
              </button>
            </div>
          </div>
        </SurfaceCard>

        <SurfaceCard>
          <div class="profile__card-header">
            <div>
              <p class="page__eyebrow">Player card</p>
              <h2 class="profile__display-name">{{ statsProfile.displayName }}</h2>
            </div>
          </div>

          <div class="profile__stats-grid">
            <StatTile label="Games played" :value="statsProfile.stats.gamesPlayed" />
            <StatTile label="Best score" :value="statsProfile.stats.bestScore" />
            <StatTile label="Average score" :value="statsProfile.stats.averageScore.toFixed(1)" />
            <StatTile label="Correct answers" :value="statsProfile.stats.totalCorrectQuestions" />
          </div>
        </SurfaceCard>

        <SurfaceCard>
          <div class="profile__card-header">
            <div>
              <p class="page__eyebrow">Game history</p>
              <h2 class="profile__section-title">Latest runs</h2>
            </div>
            <RouterLink class="button button--ghost" to="/leaderboard">Open leaderboard</RouterLink>
          </div>

          <div v-if="recentScores.length === 0" class="state-message">
            No finished sessions yet.
          </div>
          <div v-else class="table-scroll">
            <table class="content-table">
              <thead>
                <tr>
                  <th>Finished</th>
                  <th>Score</th>
                  <th>Correct</th>
                  <th>Wrong</th>
                  <th>Timer</th>
                  <th>Sets</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="score in recentScores" :key="score.scoreId">
                  <td>{{ formatDate(score.finishedAt) }}</td>
                  <td>{{ score.score }}</td>
                  <td>{{ score.correctQuestions }}</td>
                  <td>{{ score.wrongQuestions }}</td>
                  <td>{{ formatDurationLabel(score.durationSeconds) }}</td>
                  <td>{{ describeQuestionSets(score.selectedQuestionSetIds, questionSets) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </SurfaceCard>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { RouterLink, useRoute } from "vue-router";
import StatTile from "@/components/StatTile.vue";
import SurfaceCard from "@/components/SurfaceCard.vue";
import { getQuestionSets, getUserScores, getUserStats } from "@/services/api";
import { DISPLAY_NAME_LABEL, useDisplayNameForm } from "@/composables/useDisplayNameForm";
import { useCurrentUser } from "@/composables/useCurrentUser";
import { loginWithKeycloak, registerWithKeycloak } from "@/services/keycloak";
import type { QuestionSet, UserScoreList, UserStatsProfile } from "@/types/apiResponses";
import { describeQuestionSets, formatDurationLabel } from "@/utils/gameConfig";

const route = useRoute();
const { currentUser, currentUserReady } = useCurrentUser();

const loading = ref(false);
const error = ref<string | null>(null);
const statsProfile = ref<UserStatsProfile | null>(null);
const scoreList = ref<UserScoreList | null>(null);
const questionSets = ref<QuestionSet[]>([]);

const targetPublicUserId = computed(() => {
  const fromRoute = route.params.publicUserId;
  if (typeof fromRoute === "string" && fromRoute.length > 0) {
    return fromRoute;
  }

  return currentUser.value?.publicUserId ?? null;
});

const needsLogin = computed(
  () => route.name === "my-profile" && currentUserReady.value && !currentUser.value
);
const canEditProfile = computed(
  () =>
    Boolean(currentUser.value) &&
    Boolean(targetPublicUserId.value) &&
    currentUser.value?.publicUserId === targetPublicUserId.value
);
const recentScores = computed(() => scoreList.value?.scores.slice(0, 10) ?? []);
const profileDisplayName = computed(() => statsProfile.value?.displayName ?? "");
const {
  displayNameDraft,
  displayNameChanged,
  displayNameError,
  submitDisplayName,
  displayNameSubmitting,
  displayNameSuccess,
} = useDisplayNameForm({
  currentDisplayName: profileDisplayName,
  successMessage: "Display name updated.",
});

onMounted(async () => {
  await loadQuestionSets();
  await loadProfile();
});

watch([targetPublicUserId, currentUserReady], async () => {
  await loadProfile();
});

async function loadQuestionSets() {
  try {
    questionSets.value = await getQuestionSets();
  } catch (questionSetsError) {
    console.error("Failed to load question sets", questionSetsError);
  }
}

async function loadProfile() {
  if (!targetPublicUserId.value || needsLogin.value) {
    statsProfile.value = null;
    scoreList.value = null;
    return;
  }

  loading.value = true;
  error.value = null;

  try {
    const [stats, scores] = await Promise.all([
      getUserStats(targetPublicUserId.value),
      getUserScores(targetPublicUserId.value),
    ]);
    statsProfile.value = stats;
    scoreList.value = scores;
  } catch {
    error.value = "Could not load this profile right now.";
  } finally {
    loading.value = false;
  }
}

async function handleLogin() {
  await loginWithKeycloak(route.fullPath);
}

async function handleRegister() {
  await registerWithKeycloak(route.fullPath);
}

async function handleUpdateDisplayName() {
  const updatedUser = await submitDisplayName();
  if (!updatedUser) {
    return;
  }
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}
</script>

<style scoped>
.profile__card-header {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.profile__display-name,
.profile__section-title {
  margin: 0.3rem 0 0;
  color: var(--color-heading);
}

.profile__stats-grid {
  display: grid;
  gap: var(--space-3);
}

.profile__update-row {
  display: grid;
  gap: var(--space-2);
}

.profile__update-field {
  display: grid;
  gap: var(--space-2);
}

.profile__update-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.profile__update-input-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.profile__update-feedback {
  margin: 0;
  margin-left: auto;
  padding: 0.35rem 0.65rem;
  font-size: 0.78rem;
  line-height: 1.2;
  width: auto;
  max-width: 16rem;
}

.profile__update-action {
  align-self: end;
}

.profile__name-input {
  width: min(100%, 22rem);
  min-height: 2.85rem;
  padding: 0.7rem 0.9rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-pill);
  background: var(--color-surface);
  color: var(--color-heading);
}

@media (max-width: 767px) {
  .profile__update-label-row {
    align-items: center;
  }

  .profile__name-input {
    width: min(100%, 22rem);
  }
}

@media (min-width: 768px) {
  .profile__stats-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
