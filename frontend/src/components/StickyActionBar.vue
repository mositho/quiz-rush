<template>
  <div class="sticky-action-bar">
    <div class="sticky-action-bar__inner">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.sticky-action-bar {
  position: sticky;
  bottom: var(--space-4);
  z-index: 8;
  margin-top: var(--space-4);
}

.sticky-action-bar__inner {
  display: flex;
  justify-content: center;
  padding: 0;
}
</style>
