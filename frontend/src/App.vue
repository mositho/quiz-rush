<template>
  <AppShell>
    <section v-if="booting" class="app-boot page">
      <div class="app-boot__panel">
        <span class="app-boot__pulse" aria-hidden="true"></span>
        <p class="page__eyebrow">Starting Up</p>
        <h1 class="app-boot__title">Checking session...</h1>
        <p class="app-boot__copy">Loading Quiz Rush...</p>
      </div>
    </section>
    <router-view v-else />
  </AppShell>
</template>

<script setup lang="ts">
import { computed } from "vue";
import AppShell from "@/components/AppShell.vue";
import { useCurrentUser } from "@/composables/useCurrentUser";

const { authState, currentUserReady } = useCurrentUser();

const booting = computed(() => !authState.initialized || !currentUserReady);
</script>

<style scoped>
.app-boot {
  display: grid;
  min-height: calc(100vh - var(--header-height));
  place-items: center;
}

.app-boot__panel {
  width: min(100%, 28rem);
  display: grid;
  gap: var(--space-3);
  justify-items: center;
  padding: var(--space-6);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background:
    radial-gradient(
      circle at top,
      color-mix(in srgb, var(--color-primary-soft) 55%, white),
      transparent 60%
    ),
    var(--color-surface);
  box-shadow: var(--shadow-card);
  text-align: center;
}

.app-boot__pulse {
  width: 0.95rem;
  height: 0.95rem;
  border-radius: 999px;
  background: var(--color-primary);
  box-shadow: 0 0 0 0 color-mix(in srgb, var(--color-primary) 40%, transparent);
  animation: app-boot-pulse 1.4s ease-out infinite;
}

.app-boot__title {
  margin: 0;
  font-family: var(--font-heading);
  font-size: clamp(1.5rem, 4vw, 2rem);
  line-height: 1.05;
  color: var(--color-heading);
}

.app-boot__copy {
  margin: 0;
  max-width: 24rem;
  color: var(--color-text-muted);
}

@keyframes app-boot-pulse {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--color-primary) 35%, transparent);
  }
  70% {
    transform: scale(1);
    box-shadow: 0 0 0 0.9rem color-mix(in srgb, var(--color-primary) 0%, transparent);
  }
  100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--color-primary) 0%, transparent);
  }
}
</style>
