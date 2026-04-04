import { computed, readonly, ref } from "vue";
import { ApiError, getCurrentUser, updateCurrentUser } from "@/services/api";
import { authState, initKeycloak } from "@/services/keycloak";

const currentUser = ref<{
  publicUserId: string;
  displayName: string;
} | null>(null);
const currentUserLoading = ref(false);
const currentUserReady = ref(false);
const currentUserSaving = ref(false);

export async function refreshCurrentUser() {
  currentUserLoading.value = true;

  try {
    await initKeycloak();

    if (!authState.authenticated) {
      currentUser.value = null;
      return null;
    }

    currentUser.value = await getCurrentUser();
    return currentUser.value;
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 401) {
      console.error("Failed to refresh current user", error);
    }
    currentUser.value = null;
    return null;
  } finally {
    currentUserReady.value = true;
    currentUserLoading.value = false;
  }
}

export async function initCurrentUser() {
  if (!currentUserReady.value) {
    await refreshCurrentUser();
  }

  return currentUser.value;
}

export function useCurrentUser() {
  async function saveDisplayName(displayName: string) {
    currentUserSaving.value = true;

    try {
      const updatedUser = await updateCurrentUser({ displayName });
      currentUser.value = updatedUser;
      return updatedUser;
    } finally {
      currentUserSaving.value = false;
    }
  }

  return {
    authState: readonly(authState),
    currentUser: readonly(currentUser),
    currentUserReady: readonly(currentUserReady),
    currentUserLoading: readonly(currentUserLoading),
    currentUserSaving: readonly(currentUserSaving),
    isSignedIn: computed(() => authState.authenticated && Boolean(currentUser.value)),
    refreshCurrentUser,
    saveDisplayName,
  };
}
