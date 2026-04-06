import { computed, ref, toValue, watch, type MaybeRefOrGetter } from "vue";
import { ApiError } from "@/services/api";
import { useCurrentUser } from "@/composables/useCurrentUser";

export const DISPLAY_NAME_LABEL = "Display name";

type UseDisplayNameFormOptions = {
  currentDisplayName: MaybeRefOrGetter<string>;
  successMessage?: string | null;
};

const EMPTY_DISPLAY_NAME_ERROR = "Display name cannot be empty.";
const DISPLAY_NAME_SAVE_ERROR = "Could not update display name right now.";

function getDisplayNameSaveErrorMessage(error: unknown) {
  if (error instanceof ApiError) {
    try {
      const payload = JSON.parse(error.body) as { error?: string };
      return payload.error || DISPLAY_NAME_SAVE_ERROR;
    } catch {
      return DISPLAY_NAME_SAVE_ERROR;
    }
  }

  return DISPLAY_NAME_SAVE_ERROR;
}

export function useDisplayNameForm(options: UseDisplayNameFormOptions) {
  const { saveDisplayName } = useCurrentUser();

  const displayNameDraft = ref("");
  const displayNameError = ref<string | null>(null);
  const displayNameSuccess = ref<string | null>(null);
  const displayNameSubmitting = ref(false);

  const displayNameChanged = computed(() => {
    const currentDisplayName = toValue(options.currentDisplayName);
    return (
      displayNameDraft.value.trim() !== "" && displayNameDraft.value.trim() !== currentDisplayName
    );
  });

  watch(
    () => toValue(options.currentDisplayName),
    (nextDisplayName) => {
      displayNameDraft.value = nextDisplayName;
    },
    { immediate: true }
  );

  async function submitDisplayName() {
    const nextDisplayName = displayNameDraft.value.trim();
    if (!nextDisplayName) {
      displayNameError.value = EMPTY_DISPLAY_NAME_ERROR;
      displayNameSuccess.value = null;
      return null;
    }

    displayNameSubmitting.value = true;
    displayNameError.value = null;
    displayNameSuccess.value = null;

    try {
      const updatedUser = await saveDisplayName(nextDisplayName);
      displayNameDraft.value = updatedUser.displayName;

      displayNameSuccess.value = options.successMessage ?? null;
      return updatedUser;
    } catch (error) {
      displayNameError.value = getDisplayNameSaveErrorMessage(error);
      return null;
    } finally {
      displayNameSubmitting.value = false;
    }
  }

  return {
    displayNameDraft,
    displayNameChanged,
    displayNameError,
    displayNameSuccess,
    displayNameSubmitting,
    submitDisplayName,
  };
}
