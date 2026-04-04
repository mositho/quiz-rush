<template>
  <div class="app-shell" :class="{ 'app-shell--with-active-banner': showActiveBanner }">
    <header class="app-shell__header">
      <div class="app-shell__header-inner">
        <div class="app-shell__brand">
          <button
            class="app-shell__burger"
            type="button"
            :aria-expanded="menuOpen"
            aria-label="Open navigation"
            @click="menuOpen = !menuOpen"
          >
            <span></span>
            <span></span>
            <span></span>
          </button>
        </div>

        <RouterLink class="app-shell__home-link app-shell__home-link--center" to="/">
          <span class="app-shell__logo" aria-hidden="true"></span>
          <span class="app-shell__title">Quiz Rush</span>
        </RouterLink>

        <div class="app-shell__auth">
          <template v-if="!authState.initialized || (!currentUserReady && currentUserLoading)">
            <span class="app-shell__status">Checking session...</span>
          </template>
          <template
            v-else-if="authState.initialized && currentUserReady && isSignedIn && currentUser"
          >
            <div ref="accountMenuRef" class="app-shell__account-menu">
              <button
                class="button button--ghost button--compact"
                type="button"
                :aria-expanded="accountMenuOpen"
                aria-label="Open account menu"
                @click="accountMenuOpen = !accountMenuOpen"
              >
                {{ currentUser.displayName }}
              </button>
              <div v-if="accountMenuOpen" class="app-shell__account-popover">
                <RouterLink
                  class="app-shell__account-item"
                  :to="`/profile/${currentUser.publicUserId}`"
                  @click="accountMenuOpen = false"
                >
                  Profile
                </RouterLink>
                <button
                  class="app-shell__account-item app-shell__account-item--danger"
                  type="button"
                  @click="handleLogout"
                >
                  Sign out
                </button>
              </div>
            </div>
          </template>
          <template v-else>
            <button class="button button--ghost button--compact" type="button" @click="handleLogin">
              Login
            </button>
          </template>
        </div>
      </div>
    </header>

    <div v-if="showActiveBanner" class="app-shell__active-banner" role="status" aria-live="polite">
      <div class="app-shell__active-banner-inner">
        <div class="app-shell__active-banner-info">
          <span class="app-shell__active-banner-label">Game in progress:</span>
          <span v-if="activeSessionTimerLabel" class="app-shell__active-banner-timer">
            {{ activeSessionTimerLabel }}
          </span>
        </div>
        <div class="app-shell__active-banner-actions">
          <RouterLink
            class="button button--primary button--compact app-shell__active-banner-button"
            :to="`/game/${activeSessionId}`"
          >
            Rejoin session
          </RouterLink>
          <button
            class="button button--danger button--compact app-shell__active-banner-button"
            type="button"
            @click="handleAbandonSession"
          >
            Abandon session
          </button>
        </div>
      </div>
    </div>

    <transition name="fade">
      <div v-if="menuOpen" class="app-shell__overlay" @click="menuOpen = false"></div>
    </transition>

    <aside class="app-shell__drawer" :class="{ 'app-shell__drawer--open': menuOpen }">
      <div class="app-shell__drawer-header">
        <div>
          <p class="page__eyebrow">Navigate</p>
          <h2 class="app-shell__drawer-title">Quiz Rush</h2>
        </div>
      </div>

      <nav class="app-shell__nav">
        <RouterLink
          v-for="item in navigationItems"
          :key="item.to"
          class="app-shell__nav-link"
          :to="item.to"
          @click="menuOpen = false"
        >
          {{ item.label }}
        </RouterLink>
      </nav>

      <div v-if="isSignedIn && currentUser" class="app-shell__drawer-profile">
        <span class="stacked-label__title">Display name</span>
        <div class="app-shell__drawer-profile-row">
          <input
            v-model="displayNameDraft"
            class="app-shell__edit-input"
            type="text"
            maxlength="40"
            autocomplete="nickname"
            :placeholder="currentUser.displayName"
          />
          <button
            class="button button--secondary button--compact"
            type="button"
            :disabled="currentUserSaving || !displayNameChanged"
            @click="handleUpdateDisplayName"
          >
            Update
          </button>
        </div>
        <p v-if="displayNameError" class="app-shell__edit-error app-shell__edit-error--drawer">
          {{ displayNameError }}
        </p>
      </div>
    </aside>

    <main class="app-shell__content">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { ApiError } from "@/services/api";
import { refreshCurrentUser, useCurrentUser } from "@/composables/useCurrentUser";
import { refreshTrackedActiveSession, useActiveSessionTracker } from "@/composables/useGameSession";
import { logoutFromKeycloak, loginWithKeycloak } from "@/services/keycloak";

const menuOpen = ref(false);
const accountMenuOpen = ref(false);
const accountMenuRef = ref<HTMLElement | null>(null);
const route = useRoute();
const router = useRouter();
const {
  authState,
  currentUser,
  currentUserLoading,
  currentUserReady,
  currentUserSaving,
  isSignedIn,
  saveDisplayName,
} = useCurrentUser();
const { activeSessionId, activeSession, pendingSessionNavigationId, abandonTrackedSession } =
  useActiveSessionTracker();
let pollIntervalId: number | null = null;
const displayNameDraft = ref("");
const displayNameError = ref<string | null>(null);
const activeTimerNow = ref(Date.now());
let activeTimerIntervalId: number | null = null;

const showActiveBanner = computed(() => {
  if (!activeSessionId.value) {
    return false;
  }

  if (pendingSessionNavigationId.value === activeSessionId.value) {
    return false;
  }

  return !(route.name === "game" && route.params.sessionId === activeSessionId.value);
});

const navigationItems = computed(() => {
  const items = [
    { label: "Play", to: "/" },
    { label: "Leaderboard", to: "/leaderboard" },
  ];

  if (isSignedIn.value && currentUser.value) {
    items.push({ label: "Profile", to: `/profile/${currentUser.value.publicUserId}` });
  }

  return items;
});

const displayNameChanged = computed(() => {
  const currentDisplayName = currentUser.value?.displayName ?? "";
  return (
    displayNameDraft.value.trim() !== "" && displayNameDraft.value.trim() !== currentDisplayName
  );
});

const activeSessionTimerLabel = computed(() => {
  if (!activeSession.value) {
    return "";
  }

  const remainingMs = Math.max(
    0,
    new Date(activeSession.value.endsAt).getTime() - activeTimerNow.value
  );
  const totalSeconds = Math.ceil(remainingMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${String(seconds).padStart(2, "0")} left`;
});

onMounted(async () => {
  window.addEventListener("pointerdown", handleWindowPointerDown);
  await refreshTrackedActiveSession();
  syncActiveSessionPolling();
  syncActiveTimer();
});

function handleWindowPointerDown(event: PointerEvent) {
  if (!accountMenuOpen.value) {
    return;
  }

  const target = event.target;
  if (!(target instanceof Node)) {
    return;
  }

  if (accountMenuRef.value?.contains(target)) {
    return;
  }

  accountMenuOpen.value = false;
}

watch(
  () => [route.fullPath, authState.authenticated],
  async () => {
    accountMenuOpen.value = false;
    await refreshTrackedActiveSession();
    syncActiveSessionPolling();
  }
);

watch(activeSessionId, () => {
  syncActiveSessionPolling();
  syncActiveTimer();
});

watch(
  currentUser,
  (nextUser) => {
    displayNameDraft.value = nextUser?.displayName ?? "";
  },
  { immediate: true }
);

async function handleLogin() {
  accountMenuOpen.value = false;
  await loginWithKeycloak(route.fullPath);
}

async function handleLogout() {
  accountMenuOpen.value = false;
  const confirmed = window.confirm("Sign out now?");
  if (!confirmed) {
    return;
  }

  await logoutFromKeycloak();
  await refreshCurrentUser();
}

async function handleUpdateDisplayName() {
  const nextDisplayName = displayNameDraft.value.trim();
  if (!nextDisplayName) {
    displayNameError.value = "Display name cannot be empty.";
    return;
  }

  try {
    await saveDisplayName(nextDisplayName);
    displayNameError.value = null;
  } catch (saveError) {
    if (saveError instanceof ApiError) {
      try {
        const payload = JSON.parse(saveError.body) as { error?: string };
        displayNameError.value = payload.error || "Could not update display name right now.";
      } catch {
        displayNameError.value = "Could not update display name right now.";
      }
      return;
    }

    displayNameError.value = "Could not update display name right now.";
  }
}

async function handleAbandonSession() {
  const confirmed = window.confirm(
    "Abandon the current session? This will end your in-progress run."
  );
  if (!confirmed) {
    return;
  }

  const abandonedSession = await abandonTrackedSession();
  menuOpen.value = false;

  if (route.name === "game" && abandonedSession) {
    await router.push("/");
  }
}

function syncActiveSessionPolling() {
  if (pollIntervalId !== null) {
    window.clearInterval(pollIntervalId);
    pollIntervalId = null;
  }

  if (!activeSessionId.value) {
    return;
  }

  if (route.name === "game" && route.params.sessionId === activeSessionId.value) {
    return;
  }

  pollIntervalId = window.setInterval(() => {
    void refreshTrackedActiveSession();
  }, 5000);
}

function syncActiveTimer() {
  if (activeTimerIntervalId !== null) {
    window.clearInterval(activeTimerIntervalId);
    activeTimerIntervalId = null;
  }

  if (!activeSession.value) {
    return;
  }

  activeTimerNow.value = Date.now();
  activeTimerIntervalId = window.setInterval(() => {
    activeTimerNow.value = Date.now();
  }, 1000);
}

onUnmounted(() => {
  window.removeEventListener("pointerdown", handleWindowPointerDown);
  if (pollIntervalId !== null) {
    window.clearInterval(pollIntervalId);
  }
  if (activeTimerIntervalId !== null) {
    window.clearInterval(activeTimerIntervalId);
  }
});
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
}

.app-shell__header {
  position: fixed;
  inset: 0 0 auto;
  z-index: 30;
  padding: var(--space-3) var(--content-padding);
  background: color-mix(in srgb, var(--color-bg) 84%, white);
  border-bottom: 1px solid rgba(218, 200, 174, 0.8);
  backdrop-filter: blur(16px);
}

.app-shell__header-inner {
  width: 100%;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: var(--space-3);
}

.app-shell__brand,
.app-shell__home-link,
.app-shell__auth {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.app-shell__brand {
  justify-self: start;
}

.app-shell__home-link--center {
  justify-self: center;
}

.app-shell__burger {
  display: inline-grid;
  gap: 0.22rem;
  padding: 0.7rem;
  border-radius: var(--radius-pill);
  background: var(--color-surface);
  box-shadow: var(--shadow-card);
  cursor: pointer;
}

.app-shell__burger span {
  display: block;
  width: 1rem;
  height: 2px;
  border-radius: var(--radius-pill);
  background: var(--color-heading);
}

.app-shell__home-link {
  min-width: 0;
}

.app-shell__logo {
  width: 1.9rem;
  height: 1.9rem;
  border-radius: 0.7rem;
  background: var(--color-primary);
  box-shadow: inset 0 0 0 2px rgba(255, 255, 255, 0.35);
}

.app-shell__title {
  font-family: var(--font-heading);
  font-size: 1.15rem;
  font-weight: 900;
  color: var(--color-heading);
}

.app-shell__auth {
  justify-self: end;
  justify-content: end;
  flex-wrap: wrap;
}

.app-shell__status {
  font-size: 0.9rem;
  color: var(--color-text-muted);
}

.app-shell__account-menu {
  position: relative;
}

.app-shell__account-popover {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  z-index: 32;
  display: grid;
  gap: var(--space-1);
  min-width: 11rem;
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  box-shadow: var(--shadow-card);
}

.app-shell__account-item {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 2.5rem;
  padding: 0.6rem 0.75rem;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-heading);
  font-weight: 700;
  text-align: left;
  cursor: pointer;
}

.app-shell__account-item:hover {
  background: var(--color-surface-alt);
}

.app-shell__account-item--danger {
  color: var(--color-danger);
}

.app-shell__edit-input {
  min-width: 10rem;
  min-height: 2.6rem;
  padding: 0.55rem 0.8rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-pill);
  background: var(--color-surface);
  color: var(--color-heading);
}

.app-shell__edit-error {
  color: var(--color-danger);
  font-size: 0.88rem;
}

.app-shell__edit-error--drawer {
  width: auto;
  margin: 0;
}

.app-shell__content {
  min-height: 100vh;
}

.app-shell__active-banner {
  position: fixed;
  top: calc(var(--header-height) + 0.25rem);
  inset-inline: 0;
  z-index: 20;
  width: min(100%, calc(var(--page-max-width) + (var(--content-padding) * 2)));
  margin: 0 auto;
  padding-inline: var(--content-padding);
}

.app-shell__active-banner-inner {
  width: 100%;
  max-width: 40rem;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: 0.7rem 0.85rem;
  border-radius: var(--radius-xl);
  background: var(--color-warning-soft);
  border: 1px solid color-mix(in srgb, var(--color-warning) 34%, var(--color-border));
  box-shadow: var(--shadow-card);
}

.app-shell__active-banner-info {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.app-shell__active-banner-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: end;
  gap: var(--space-2);
}

.app-shell__active-banner-button {
  min-width: 9.75rem;
}

.app-shell__active-banner-timer {
  color: var(--color-warning-strong);
  font-weight: 700;
}

.app-shell__active-banner-label {
  color: var(--color-heading);
  font-weight: 800;
}

.app-shell__overlay {
  position: fixed;
  inset: 0;
  z-index: 24;
  background: var(--color-overlay);
}

.app-shell__drawer {
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 25;
  display: flex;
  flex-direction: column;
  width: min(20rem, calc(100vw - 2.5rem));
  padding: calc(var(--header-height) + var(--space-2)) var(--space-4) var(--space-4);
  background: var(--color-surface);
  border-right: 1px solid var(--color-border);
  box-shadow: var(--shadow-float);
  transform: translateX(-104%);
  transition: transform var(--transition-medium);
}

.app-shell__drawer--open {
  transform: translateX(0);
}

.app-shell__drawer-header {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

.app-shell__drawer-title {
  margin: 0.25rem 0 0;
  color: var(--color-heading);
}

.app-shell__nav {
  display: grid;
  gap: var(--space-3);
  overflow-y: auto;
  padding-right: var(--space-1);
}

.app-shell__nav-link {
  padding: var(--space-4);
  border-radius: var(--radius-lg);
  background: var(--color-surface-alt);
  color: var(--color-heading);
  font-weight: 700;
}

.app-shell__drawer-profile {
  display: grid;
  gap: var(--space-2);
  margin-top: auto;
  padding-top: var(--space-5);
  position: sticky;
  bottom: 0;
  background: var(--color-surface);
}

.app-shell__drawer-profile-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-2);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--transition-medium);
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 767px) {
  .app-shell__auth {
    gap: var(--space-2);
  }

  .app-shell__title {
    font-size: 1rem;
  }

  .app-shell__user {
    flex-wrap: wrap;
    justify-content: end;
  }

  .app-shell__edit-input {
    min-width: 8rem;
  }

  .app-shell__active-banner-inner {
    display: grid;
    justify-items: center;
    text-align: center;
  }

  .app-shell__active-banner-info {
    justify-content: center;
    gap: var(--space-2);
  }

  .app-shell__active-banner-actions {
    justify-content: center;
  }

  .app-shell__active-banner-timer {
    justify-self: auto;
  }
}

:deep(.page) {
  padding-top: calc(var(--header-height) + var(--space-6));
}

.app-shell--with-active-banner :deep(.page) {
  padding-top: calc(var(--header-height) + var(--space-6) + 4.5rem);
}
</style>
