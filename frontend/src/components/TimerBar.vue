<template>
  <div class="timer-bar" :class="{ 'timer-bar--warning': progressRatio <= 0.25 }">
    <div class="timer-bar__meta">
      <span class="timer-bar__label">Time left</span>
      <span class="timer-bar__value">{{ formattedTimeLeft }}</span>
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
}>();

const now = ref(Date.now());
let intervalId: number | null = null;

const endTimeMs = computed(() => new Date(props.endsAt).getTime());
const durationMs = computed(() => Math.max(props.durationSeconds, 1) * 1000);

const remainingMs = computed(() => Math.max(0, endTimeMs.value - now.value));
const progressRatio = computed(() => remainingMs.value / durationMs.value);
const progressPercent = computed(() => Math.max(0, Math.min(100, progressRatio.value * 100)));

const formattedTimeLeft = computed(() => {
  const totalSeconds = Math.ceil(remainingMs.value / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  return `${minutes}:${String(seconds).padStart(2, "0")}`;
});

function tick() {
  now.value = Date.now();
}

function stopTimer() {
  if (intervalId !== null) {
    window.clearInterval(intervalId);
    intervalId = null;
  }
}

function startTimer() {
  stopTimer();
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
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.timer-bar__meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.95rem;
}

.timer-bar__label {
  color: var(--text);
  opacity: 0.75;
}

.timer-bar__value {
  font-weight: 700;
  color: var(--text-h);
}

.timer-bar__track {
  height: 0.75rem;
  border-radius: 999px;
  background: color-mix(in srgb, var(--text) 12%, transparent);
  overflow: hidden;
}

.timer-bar__fill {
  height: 100%;
  background: linear-gradient(90deg, #22c55e 0%, #84cc16 100%);
  transition:
    width 0.9s linear,
    background 0.2s ease;
}

.timer-bar--warning .timer-bar__fill {
  background: linear-gradient(90deg, #f97316 0%, #ef4444 100%);
}
</style>
