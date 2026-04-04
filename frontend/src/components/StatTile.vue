<template>
  <article class="stat-tile">
    <span class="stat-tile__label">{{ label }}</span>
    <strong class="stat-tile__value">{{ value }}</strong>
    <span v-if="hint" class="stat-tile__hint">{{ hint }}</span>
  </article>
</template>

<script setup lang="ts">
defineProps<{
  label: string;
  value: string | number;
  hint?: string;
}>();
</script>

<style scoped>
.stat-tile {
  display: grid;
  gap: var(--space-2);
  padding: var(--space-4);
  border-radius: var(--radius-lg);
  background: var(--color-surface-alt);
}

.stat-tile__label,
.stat-tile__hint {
  color: var(--color-text-muted);
  font-size: 0.88rem;
}

.stat-tile__value {
  font-size: 1.5rem;
  color: var(--color-heading);
}
</style>
