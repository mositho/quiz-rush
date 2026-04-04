<template>
  <SurfaceCard>
    <div class="result-card__score-block">
      <span class="result-card__score">{{ session.currentScore }}</span>
    </div>

    <p class="result-card__summary">{{ performanceSummary }}</p>
    <p class="result-card__meta-line">{{ setupSummary }}</p>

    <p v-if="message && messageTone === 'error'" class="state-message state-message--error">
      {{ message }}
    </p>

    <div class="result-card__actions">
      <div class="result-card__actions-row">
        <RouterLink class="button button--primary" to="/">Play again</RouterLink>
        <RouterLink class="button button--ghost" to="/leaderboard">
          {{ leaderboardActionLabel }}
        </RouterLink>
      </div>
      <template v-if="saveAction === 'prompt-auth'">
        <button class="result-card__save-cta" type="button" @click="emit('saveWithLogin')">
          Login to save your score
        </button>
      </template>
    </div>
  </SurfaceCard>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { RouterLink } from "vue-router";
import SurfaceCard from "@/components/SurfaceCard.vue";
import type { QuestionSet, Session } from "@/types/apiResponses";
import { describeQuestionSets, formatDurationLabel } from "@/utils/gameConfig";

const props = defineProps<{
  session: Session;
  questionSets: QuestionSet[];
  leaderboardActionLabel: string;
  saveAction: "none" | "prompt-auth" | "linked";
  message?: string | null;
  messageTone?: "neutral" | "error";
}>();

const emit = defineEmits<{
  (e: "saveWithLogin"): void;
}>();

const questionSetSummary = computed(() =>
  describeQuestionSets(props.session.selectedQuestionSetIds, props.questionSets)
);

const performanceSummary = computed(
  () => `${props.session.correctQuestions} correct • ${props.session.wrongQuestions} wrong`
);

const setupSummary = computed(
  () => `${formatDurationLabel(props.session.durationSeconds)} • ${questionSetSummary.value}`
);
</script>

<style scoped>
.result-card__score {
  color: var(--color-heading);
  font-size: 3rem;
  font-weight: 900;
}

.result-card__score-block {
  display: grid;
  justify-items: center;
  gap: 0;
}

.result-card__summary,
.result-card__meta-line {
  margin: 0;
  text-align: center;
}

.result-card__summary {
  color: var(--color-heading);
  font-size: 1.05rem;
  font-weight: 800;
}

.result-card__meta-line {
  color: var(--color-text-muted);
}

.result-card__actions {
  display: grid;
  justify-items: center;
  gap: var(--space-3);
}

.result-card__actions-row {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--space-3);
}

.result-card__save-cta {
  width: 100%;
  max-width: 26rem;
  min-height: 2.85rem;
  padding: 0.8rem 0.95rem;
  border: 1px solid color-mix(in srgb, var(--color-warning) 34%, var(--color-border));
  border-radius: var(--radius-xl);
  background: var(--color-warning-soft);
  color: var(--color-warning-strong);
  font-weight: 800;
  cursor: pointer;
}
</style>
