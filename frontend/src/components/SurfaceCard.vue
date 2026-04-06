<template>
  <section class="surface-card">
    <slot />
  </section>
</template>

<style scoped>
.surface-card {
  display: grid;
  gap: var(--space-4);
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
}
</style>
