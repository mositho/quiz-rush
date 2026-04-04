<template>
  <section class="page game">
    <div v-if="loading && !session" class="state-message">Loading session...</div>
    <div v-else-if="error && !session" class="state-message state-message--error">{{ error }}</div>

    <template v-else-if="session">
      <div
        v-if="session.status !== 'finished'"
        :key="`${session.sessionId}-active`"
        class="game__active"
      >
        <SurfaceCard class="game__surface">
          <div class="game__sticky-top">
            <div class="game__topbar">
              <div class="game__score" :class="{ 'game__score--flash': scoreFlash }">
                <strong>{{ session.currentScore }}</strong>
              </div>
            </div>

            <TimerBar
              :ends-at="session.endsAt"
              :duration-seconds="session.durationSeconds"
              :flash-negative="timerFlash"
              :question-number="displayedQuestion ? displayedQuestion.position + 1 : undefined"
              @expired="handleTimerExpired"
            />
          </div>

          <QuestionCard
            v-if="displayedQuestion"
            :question="displayedQuestion"
            :disabled="submitting || feedbackLocked"
            :selected-index="selectedAnswerIndex"
            :was-correct="selectedAnswerCorrect"
            @answer-selected="handleAnswer"
          />

          <p v-if="error" class="state-message state-message--error">{{ error }}</p>

          <div class="game__footer-actions">
            <button
              class="button button--danger"
              type="button"
              :disabled="loading"
              @click="handleFinishEarly"
            >
              Finish early
            </button>
          </div>
        </SurfaceCard>
      </div>

      <div v-else :key="`${session.sessionId}-finished`" class="section-stack">
        <GameResultCard
          :session="session"
          :question-sets="questionSets"
          :leaderboard-action-label="leaderboardActionLabel"
          :save-action="saveAction"
          :message="resultMessage"
          :message-tone="resultMessageTone"
          @save-with-login="handleSaveWithLogin"
        />
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import GameResultCard from "@/components/GameResultCard.vue";
import QuestionCard from "@/components/QuestionCard.vue";
import SurfaceCard from "@/components/SurfaceCard.vue";
import TimerBar from "@/components/TimerBar.vue";
import { useCurrentUser } from "@/composables/useCurrentUser";
import { resetGameSessionState, useGameSession } from "@/composables/useGameSession";
import { getLeaderboard, getQuestionSets, getUserScores, linkAccount } from "@/services/api";
import { loginWithKeycloak } from "@/services/keycloak";
import type { LeaderboardEntry, Question, QuestionSet, ScoreSummary } from "@/types/apiResponses";
import { LEADERBOARD_LIMIT, buildConfigurationKey } from "@/utils/gameConfig";

const PENDING_LINK_SESSION_KEY = "quiz-rush.pending-link-session-id";

const route = useRoute();
const {
  session,
  loading,
  submitting,
  error,
  answerResult,
  loadSession,
  confirmAnswer,
  endSession,
} = useGameSession();
const { authState, isSignedIn, currentUser } = useCurrentUser();

const questionSets = ref<QuestionSet[]>([]);
const displayedQuestion = ref<Question | null>(null);
const selectedAnswerIndex = ref<number | null>(null);
const selectedAnswerCorrect = ref<boolean | null>(null);
const feedbackLocked = ref(false);
const scoreFlash = ref(false);
const timerFlash = ref(false);
const resultMessage = ref<string | null>(null);
const resultMessageTone = ref<"neutral" | "error">("neutral");
const leaderboardEntries = ref<LeaderboardEntry[]>([]);
const ownScores = ref<ScoreSummary[]>([]);
const linkedScoreId = ref<string | null>(null);
const timerExpiredSyncing = ref(false);

const sessionConfigKey = computed(() => {
  if (!session.value) {
    return "";
  }

  return buildConfigurationKey(session.value.durationSeconds, session.value.selectedQuestionSetIds);
});

const saveAction = computed<"none" | "prompt-auth" | "linked">(() => {
  if (!session.value || session.value.status !== "finished") {
    return "none";
  }

  if (linkedScoreId.value || finishedSessionScore.value) {
    return "linked";
  }

  return isSignedIn.value ? "none" : "prompt-auth";
});

const finishedSessionScore = computed(() => {
  if (!session.value || session.value.status !== "finished") {
    return null;
  }

  return ownScores.value.find((score) => score.sessionId === session.value?.sessionId) ?? null;
});

const leaderboardActionLabel = computed(() => {
  const matchingVisibleEntry = finishedSessionScore.value
    ? leaderboardEntries.value.find(
        (entry) => entry.scoreId === finishedSessionScore.value?.scoreId
      )
    : null;

  if (matchingVisibleEntry) {
    return `Rank #${matchingVisibleEntry.rank}`;
  }

  return "Leaderboard";
});

onMounted(async () => {
  await Promise.all([loadCurrentSession(), loadQuestionSets()]);
});

watch(
  () => route.params.sessionId,
  async (nextSessionId, previousSessionId) => {
    if (typeof nextSessionId !== "string" || nextSessionId === previousSessionId) {
      return;
    }

    await loadCurrentSession();
  }
);

watch(
  () => session.value?.status,
  async (status) => {
    if (status === "finished") {
      await loadFinishedState();
      await tryAutoLinkPendingScore();
    }
  },
  { immediate: true }
);

watch(
  () => authState.authenticated,
  async (authenticated) => {
    if (authenticated) {
      await tryAutoLinkPendingScore();
    }
  }
);

watch(
  () => session.value?.currentQuestion,
  (nextQuestion) => {
    if (!feedbackLocked.value) {
      displayedQuestion.value = nextQuestion ?? null;
    }
  },
  { immediate: true }
);

async function loadCurrentSession() {
  const sessionId = route.params.sessionId as string;
  if (!sessionId) {
    return;
  }

  resetFeedback();
  await loadSession(sessionId);
}

async function loadQuestionSets() {
  try {
    questionSets.value = await getQuestionSets();
  } catch (questionSetsError) {
    console.error("Failed to load question sets", questionSetsError);
  }
}

async function loadFinishedState() {
  if (!session.value) {
    return;
  }

  try {
    leaderboardEntries.value = (
      await getLeaderboard(sessionConfigKey.value, LEADERBOARD_LIMIT)
    ).entries;
  } catch (leaderboardError) {
    console.error("Failed to load leaderboard context", leaderboardError);
    leaderboardEntries.value = [];
  }

  if (currentUser.value) {
    try {
      ownScores.value = (await getUserScores(currentUser.value.publicUserId)).scores;
    } catch (scoresError) {
      console.error("Failed to load own scores", scoresError);
      ownScores.value = [];
    }
  } else {
    ownScores.value = [];
  }
}

async function handleAnswer(index: number) {
  if (!session.value || feedbackLocked.value) {
    return;
  }

  selectedAnswerIndex.value = index;
  feedbackLocked.value = true;

  const response = await confirmAnswer(index);
  if (!response) {
    feedbackLocked.value = false;
    selectedAnswerIndex.value = null;
    return;
  }

  selectedAnswerCorrect.value = response.result.correct;

  if (response.result.correct) {
    scoreFlash.value = true;
    window.setTimeout(() => {
      scoreFlash.value = false;
    }, 700);
  } else {
    timerFlash.value = true;
    window.setTimeout(() => {
      timerFlash.value = false;
    }, 700);
  }

  window.setTimeout(() => {
    resetFeedback();
  }, 1000);
}

async function handleFinishEarly() {
  const confirmed = window.confirm("Finish this session early? Your run will end immediately.");
  if (!confirmed) {
    return;
  }

  await endSession();
}

async function handleTimerExpired() {
  if (!session.value || timerExpiredSyncing.value || session.value.status === "finished") {
    return;
  }

  timerExpiredSyncing.value = true;
  try {
    await loadSession(session.value.sessionId);
  } finally {
    timerExpiredSyncing.value = false;
  }
}

async function handleSaveWithLogin() {
  if (!session.value) {
    return;
  }

  sessionStorage.setItem(PENDING_LINK_SESSION_KEY, session.value.sessionId);
  await loginWithKeycloak(window.location.href);
}

async function tryAutoLinkPendingScore() {
  if (!session.value || session.value.status !== "finished" || !isSignedIn.value) {
    return;
  }

  const pendingSessionId = sessionStorage.getItem(PENDING_LINK_SESSION_KEY);
  if (pendingSessionId !== session.value.sessionId) {
    resultMessage.value = null;
    return;
  }

  try {
    const linkedAccount = await linkAccount(session.value.sessionId);
    linkedScoreId.value = linkedAccount.scoreId;
    resultMessage.value = null;
    sessionStorage.removeItem(PENDING_LINK_SESSION_KEY);
    await loadFinishedState();
  } catch (linkError) {
    console.error("Failed to link account", linkError);
    resultMessage.value =
      "We could not save this score automatically. Please try again from your profile.";
    resultMessageTone.value = "error";
  }
}

function resetFeedback() {
  selectedAnswerIndex.value = null;
  selectedAnswerCorrect.value = answerResult.value?.correct ?? null;
  feedbackLocked.value = false;
  displayedQuestion.value = session.value?.currentQuestion ?? null;
  resetGameSessionState();
  selectedAnswerCorrect.value = null;
}
</script>

<style scoped>
.game__active {
  display: grid;
}

.game__surface {
  gap: var(--space-4);
}

.game__sticky-top {
  display: grid;
  align-self: start;
}

.game__topbar {
  display: grid;
  justify-items: center;
  gap: var(--space-3);
}

.game__score {
  display: grid;
  justify-items: center;
  align-self: center;
}

.game__score strong {
  font-size: 2.2rem;
  color: var(--color-heading);
}

.game__score--flash {
  animation: score-pop 600ms ease;
}

.game__footer-actions {
  display: flex;
  justify-content: center;
}

@media (max-width: 767px) {
  .game__surface {
    padding: var(--space-4);
    overflow: clip;
  }

  .game__sticky-top {
    position: sticky;
    top: var(--header-height);
    z-index: 2;
    isolation: isolate;
    gap: var(--space-2);
    margin-inline: calc(var(--space-4) * -1);
    padding: 0 var(--space-4) var(--space-2);
    background: var(--color-surface);
    border-radius: inherit;
  }

  .game__topbar {
    gap: var(--space-2);
  }

  .game__score strong {
    line-height: 1;
  }
}

@keyframes score-pop {
  0%,
  100% {
    transform: scale(1);
    background: transparent;
  }
  50% {
    transform: scale(1.04);
    background: var(--color-success-soft);
  }
}
</style>
