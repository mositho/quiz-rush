<template>
  <div class="timer-bar" :class="timerClasses">
    <div class="timer-bar__meta">
      <span v-if="questionNumber !== undefined" class="pill">Question {{ questionNumber }}</span>
      <div class="timer-bar__status">
        <span class="timer-bar__label">Time left</span>
        <span class="timer-bar__value">{{ formattedTimeLeft }}</span>
      </div>
    </div>
    <div class="timer-bar__track" aria-hidden="true">
      <div class="timer-bar__fill" :style="{ width: `${progressPercent}%` }"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";

const props = defineProps<{
  endsAt: string;
  durationSeconds: number;
  flashNegative?: boolean;
  questionNumber?: number;
}>();

const emit = defineEmits<{
  (e: "expired"): void;
}>();

const now = ref(Date.now());
const emittedExpired = ref(false);
let intervalId: number | null = null;

const endTimeMs = computed(() => new Date(props.endsAt).getTime());
const durationMs = computed(() => Math.max(props.durationSeconds, 1) * 1000);
const remainingMs = computed(() => Math.max(0, endTimeMs.value - now.value));
const progressRatio = computed(() => remainingMs.value / durationMs.value);
const progressPercent = computed(() => Math.max(0, Math.min(100, progressRatio.value * 100)));

const timerClasses = computed(() => ({
  "timer-bar--caution": progressRatio.value <= 0.5,
  "timer-bar--warning": progressRatio.value <= 0.25,
  "timer-bar--negative": props.flashNegative,
}));

const formattedTimeLeft = computed(() => {
  const totalSeconds = Math.ceil(remainingMs.value / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  return `${minutes}:${String(seconds).padStart(2, "0")}`;
});

function tick() {
  now.value = Date.now();

  if (remainingMs.value === 0 && !emittedExpired.value) {
    emittedExpired.value = true;
    emit("expired");
  }
}

function stopTimer() {
  if (intervalId !== null) {
    clearInterval(intervalId);
    intervalId = null;
  }
}

function startTimer() {
  stopTimer();
  emittedExpired.value = false;
  tick();

  if (remainingMs.value > 0) {
    intervalId = window.setInterval(tick, 1000);
  }
}

watch(() => props.endsAt, startTimer);

onMounted(startTimer);
onUnmounted(stopTimer);
</script>

<style scoped>
.timer-bar {
  display: grid;
  gap: var(--space-3);
}

.timer-bar__meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-3);
}

.timer-bar__status {
  display: grid;
  justify-items: end;
  gap: 0.15rem;
}

.timer-bar__label {
  color: var(--color-text-muted);
  font-size: 0.92rem;
}

.timer-bar__value {
  color: var(--color-heading);
  font-weight: 800;
  font-size: 1.1rem;
}

.timer-bar--negative .timer-bar__value {
  animation: timer-pop 600ms ease;
}

.timer-bar__track {
  height: 0.95rem;
  border-radius: var(--radius-pill);
  background: color-mix(in srgb, var(--color-border) 65%, white);
  overflow: hidden;
}

.timer-bar__fill {
  height: 100%;
  background: var(--color-primary);
  transition:
    width 0.9s linear,
    background-color 500ms ease,
    transform var(--transition-fast);
}

.timer-bar--caution .timer-bar__fill {
  background: var(--color-caution);
}

.timer-bar--warning .timer-bar__fill {
  background: var(--color-warning);
}

.timer-bar--negative .timer-bar__fill {
  animation: timer-shake 500ms ease;
}

@keyframes timer-shake {
  0%,
  100% {
    transform: translateX(0);
  }
  25% {
    transform: translateX(-4px);
  }
  50% {
    transform: translateX(4px);
  }
  75% {
    transform: translateX(-2px);
  }
}

@keyframes timer-pop {
  0%,
  100% {
    transform: scale(1);
    color: var(--color-heading);
    background: transparent;
  }
  50% {
    transform: scale(1.04);
    color: var(--color-danger);
    background: var(--color-danger-soft);
  }
}
</style>
