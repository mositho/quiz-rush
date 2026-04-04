<template>
  <section class="page leaderboard">
    <div class="section-stack">
      <div v-if="loadingEntries" class="state-message">Loading leaderboard...</div>
      <div v-else-if="entriesError" class="state-message state-message--error">
        {{ entriesError }}
      </div>

      <SurfaceCard v-else>
        <div v-if="pinnedVisibleEntry" class="leaderboard__pinned">
          <span class="stacked-label__title">Your best</span>
          <button
            class="leaderboard__pinned-link"
            type="button"
            @click="scrollToEntry(pinnedVisibleEntry.scoreId)"
          >
            Rank #{{ pinnedVisibleEntry.rank }}
          </button>
        </div>

        <div ref="filtersRef" class="leaderboard__filters">
          <div class="leaderboard__filter-header">
            <button
              class="leaderboard__filter-trigger"
              type="button"
              @click="toggleFilterMenu('timer')"
            >
              Timer
            </button>

            <div v-if="openFilterMenu === 'timer'" class="leaderboard__filter-menu">
              <button
                v-for="preset in timerPresets"
                :key="preset"
                class="leaderboard__filter-option"
                :class="{ 'leaderboard__filter-option--active': durationSeconds === preset }"
                type="button"
                @click="selectTimer(preset)"
              >
                {{ formatDurationLabel(preset) }}
              </button>
            </div>
          </div>

          <div class="leaderboard__filter-header">
            <button
              class="leaderboard__filter-trigger"
              type="button"
              @click="toggleFilterMenu('sets')"
            >
              Question sets
            </button>

            <div
              v-if="openFilterMenu === 'sets'"
              class="leaderboard__filter-menu leaderboard__filter-menu--wide"
            >
              <div v-if="loadingSets" class="state-message">Loading question sets...</div>
              <div v-else-if="setsError" class="state-message state-message--error">
                {{ setsError }}
              </div>
              <button
                v-for="questionSet in questionSets"
                :key="questionSet.id"
                class="leaderboard__filter-option"
                :class="{
                  'leaderboard__filter-option--active': selectedSetIds.includes(questionSet.id),
                }"
                type="button"
                @click="toggleSet(questionSet.id)"
              >
                {{ questionSet.name }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="entries.length === 0" class="state-message">
          No leaderboard entries yet for the currently selected filters.
        </div>

        <div v-else class="leaderboard__mobile-list">
          <article
            v-for="entry in entries"
            :key="entry.scoreId"
            :data-score-id="entry.scoreId"
            class="leaderboard__mobile-entry"
            :class="{
              'leaderboard__mobile-entry--mine':
                entry.player.publicUserId === currentUser?.publicUserId,
            }"
          >
            <div class="leaderboard__mobile-top">
              <span class="leaderboard__mobile-rank">#{{ entry.rank }}</span>
              <RouterLink
                class="leaderboard__player-link"
                :to="`/profile/${entry.player.publicUserId}`"
              >
                {{ entry.player.displayName }}
              </RouterLink>
              <strong class="leaderboard__mobile-score">{{ entry.score }}</strong>
            </div>
            <div class="leaderboard__mobile-stats">
              <div class="stacked-label">
                <span class="stacked-label__title">Correct</span>
                <span class="stacked-label__value">{{
                  scoreStatsMap[entry.scoreId]?.correct ?? "..."
                }}</span>
              </div>
              <div class="stacked-label">
                <span class="stacked-label__title">Wrong</span>
                <span class="stacked-label__value">{{
                  scoreStatsMap[entry.scoreId]?.wrong ?? "..."
                }}</span>
              </div>
              <div class="stacked-label">
                <span class="stacked-label__title">Rank</span>
                <span class="stacked-label__value">#{{ entry.rank }}</span>
              </div>
            </div>
          </article>
        </div>

        <div v-if="entries.length > 0" class="leaderboard__table-wrap">
          <table class="content-table leaderboard__table">
            <thead>
              <tr>
                <th>Rank</th>
                <th>Player</th>
                <th>Total score</th>
                <th>Correct</th>
                <th>Wrong</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="entry in entries"
                :key="entry.scoreId"
                :data-score-id="entry.scoreId"
                :class="{
                  'leaderboard__row--mine': entry.player.publicUserId === currentUser?.publicUserId,
                }"
              >
                <td>#{{ entry.rank }}</td>
                <td>
                  <RouterLink
                    class="leaderboard__player-link"
                    :to="`/profile/${entry.player.publicUserId}`"
                  >
                    {{ entry.player.displayName }}
                  </RouterLink>
                </td>
                <td>{{ entry.score }}</td>
                <td>{{ scoreStatsMap[entry.scoreId]?.correct ?? "..." }}</td>
                <td>{{ scoreStatsMap[entry.scoreId]?.wrong ?? "..." }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </SurfaceCard>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { RouterLink } from "vue-router";
import SurfaceCard from "@/components/SurfaceCard.vue";
import { useCurrentUser } from "@/composables/useCurrentUser";
import { getLeaderboard, getQuestionSets, getScore, getUserScores } from "@/services/api";
import type { LeaderboardEntry, QuestionSet, ScoreSummary } from "@/types/apiResponses";
import {
  DEFAULT_DURATION_SECONDS,
  LEADERBOARD_LIMIT,
  TIMER_PRESETS,
  buildConfigurationKey,
  findBestScoreForConfig,
  formatDurationLabel,
} from "@/utils/gameConfig";

const { currentUser } = useCurrentUser();

const durationSeconds = ref(DEFAULT_DURATION_SECONDS);
const selectedSetIds = ref<string[]>([]);
const questionSets = ref<QuestionSet[]>([]);
const loadingSets = ref(false);
const setsError = ref<string | null>(null);
const loadingEntries = ref(false);
const entriesError = ref<string | null>(null);
const entries = ref<LeaderboardEntry[]>([]);
const ownScores = ref<ScoreSummary[]>([]);
const scoreStatsMap = ref<Record<string, { correct: number; wrong: number }>>({});
const openFilterMenu = ref<"timer" | "sets" | null>(null);
const filtersRef = ref<HTMLElement | null>(null);
const timerPresets = [...TIMER_PRESETS];

const configurationKey = computed(() =>
  buildConfigurationKey(durationSeconds.value, selectedSetIds.value)
);
const pinnedScore = computed(() => findBestScoreForConfig(ownScores.value, configurationKey.value));
const pinnedVisibleEntry = computed(() => {
  if (!pinnedScore.value) {
    return null;
  }

  return entries.value.find((entry) => entry.scoreId === pinnedScore.value?.scoreId) ?? null;
});

onMounted(async () => {
  window.addEventListener("pointerdown", handleWindowPointerDown);
  await loadQuestionSets();
  await loadOwnScores();
});

onUnmounted(() => {
  window.removeEventListener("pointerdown", handleWindowPointerDown);
});

watch(configurationKey, async () => {
  if (selectedSetIds.value.length > 0) {
    await loadEntries();
  }
});

watch(
  () => currentUser.value?.publicUserId,
  async () => {
    await loadOwnScores();
  }
);

async function loadQuestionSets() {
  loadingSets.value = true;
  setsError.value = null;

  try {
    questionSets.value = await getQuestionSets();
    selectedSetIds.value = questionSets.value.map((set) => set.id);
    await loadEntries();
  } catch {
    setsError.value = "Could not load question sets.";
  } finally {
    loadingSets.value = false;
  }
}

async function loadEntries() {
  loadingEntries.value = true;
  entriesError.value = null;

  try {
    const leaderboard = await getLeaderboard(configurationKey.value, LEADERBOARD_LIMIT);
    entries.value = leaderboard.entries;
    await hydrateVisibleScoreStats();
  } catch {
    entriesError.value = "Could not load leaderboard entries.";
    entries.value = [];
    scoreStatsMap.value = {};
  } finally {
    loadingEntries.value = false;
  }
}

async function loadOwnScores() {
  if (!currentUser.value) {
    ownScores.value = [];
    return;
  }

  try {
    ownScores.value = (await getUserScores(currentUser.value.publicUserId)).scores;
  } catch (scoresError) {
    console.error("Failed to load current user scores", scoresError);
    ownScores.value = [];
  }
}

async function hydrateVisibleScoreStats() {
  const nextStats: Record<string, { correct: number; wrong: number }> = {};

  await Promise.all(
    entries.value.map(async (entry) => {
      try {
        const detail = await getScore(entry.scoreId);
        nextStats[entry.scoreId] = {
          correct: detail.correctQuestions,
          wrong: detail.wrongQuestions,
        };
      } catch {
        nextStats[entry.scoreId] = {
          correct: 0,
          wrong: 0,
        };
      }
    })
  );

  scoreStatsMap.value = nextStats;
}

function toggleFilterMenu(menu: "timer" | "sets") {
  openFilterMenu.value = openFilterMenu.value === menu ? null : menu;
}

function handleWindowPointerDown(event: PointerEvent) {
  if (!openFilterMenu.value) {
    return;
  }

  const target = event.target;
  if (!(target instanceof Node)) {
    return;
  }

  if (filtersRef.value?.contains(target)) {
    return;
  }

  openFilterMenu.value = null;
}

function selectTimer(value: number) {
  durationSeconds.value = value;
  openFilterMenu.value = null;
}

function toggleSet(questionSetId: string) {
  selectedSetIds.value = selectedSetIds.value.includes(questionSetId)
    ? selectedSetIds.value.filter((entry) => entry !== questionSetId)
    : [...selectedSetIds.value, questionSetId];
}

function scrollToEntry(scoreId: string) {
  const target = document.querySelector<HTMLElement>(`[data-score-id="${scoreId}"]`);
  target?.scrollIntoView({ behavior: "smooth", block: "center" });
}
</script>

<style scoped>
.leaderboard__pinned {
  display: grid;
  gap: var(--space-1);
}

.leaderboard__pinned-link {
  justify-self: start;
  padding: 0;
  background: transparent;
  color: var(--color-primary-strong);
  font-weight: 800;
  cursor: pointer;
}

.leaderboard__filters {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}

.leaderboard__filter-header {
  position: relative;
  vertical-align: top;
}

.leaderboard__filter-trigger {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 2.6rem;
  padding: 0.65rem 0.9rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-pill);
  background: var(--color-surface-alt);
  color: var(--color-heading);
  font-size: 0.82rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  cursor: pointer;
  transition:
    background var(--transition-fast),
    border-color var(--transition-fast),
    transform var(--transition-fast);
}

.leaderboard__filter-trigger:hover {
  transform: translateY(-1px);
  background: color-mix(in srgb, var(--color-surface-alt) 78%, white);
  border-color: color-mix(in srgb, var(--color-primary) 28%, var(--color-border));
}

.leaderboard__mobile-list {
  display: grid;
  gap: var(--space-3);
}

.leaderboard__mobile-entry {
  display: grid;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface-alt);
}

.leaderboard__mobile-entry--mine {
  background: color-mix(in srgb, var(--color-primary-soft) 60%, white);
}

.leaderboard__mobile-top {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: var(--space-3);
}

.leaderboard__mobile-rank,
.leaderboard__mobile-score {
  color: var(--color-heading);
  font-weight: 800;
}

.leaderboard__mobile-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-3);
}

.leaderboard__table-wrap {
  width: 100%;
  display: none;
}

.leaderboard__table {
  min-width: 0;
  table-layout: fixed;
}

.leaderboard__table :deep(th),
.leaderboard__table :deep(td) {
  white-space: normal;
  word-break: break-word;
}

.leaderboard__filter-menu {
  position: absolute;
  top: calc(100% + 0.5rem);
  left: 0;
  z-index: 5;
  display: grid;
  gap: var(--space-2);
  min-width: 10rem;
  max-width: 18rem;
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  box-shadow: var(--shadow-card);
}

.leaderboard__filter-menu--wide {
  min-width: 14rem;
}

.leaderboard__filter-current {
  color: var(--color-text-muted);
  font-size: 0.85rem;
  font-weight: 600;
}

.leaderboard__filter-option {
  padding: 0.55rem 0.7rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-heading);
  text-align: left;
  cursor: pointer;
}

.leaderboard__filter-option--active {
  background: var(--color-primary-soft);
  border-color: color-mix(in srgb, var(--color-primary) 55%, var(--color-border));
}

.leaderboard__row--mine {
  background: color-mix(in srgb, var(--color-primary-soft) 60%, white);
}

.leaderboard__player-link {
  font-weight: 800;
  color: var(--color-primary-strong);
}

@media (min-width: 768px) {
  .leaderboard__mobile-list {
    display: none;
  }

  .leaderboard__table-wrap {
    display: block;
  }
}
</style>
