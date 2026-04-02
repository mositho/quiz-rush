import { ref, readonly } from "vue";
import type { Session } from "@/types/apiResponses";
import type { StartSessionRequest } from "@/types/apiRequests";
import { startSession as apiStartSession, submitAnswer as apiSubmitAnswer, getSession } from "@/services/api";
import { router } from "@/router";

const session = ref<Session | null>(null);
const loading = ref(false);
const submitting = ref(false);
const error = ref<string | null>(null);

export function useGameSession() {
  async function startNewSession(request: StartSessionRequest) {
    loading.value = true;
    error.value = null;
    try {
      session.value = await apiStartSession(request);
      console.log("Started new session:", session.value);
      router.push(`/game/${session.value.sessionId}`);
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to start new session";
      session.value = null;
    } finally {
      loading.value = false;
    }
  }


  async function confirmAnswer(answerIndex: number) {
    if (!session.value?.sessionId || submitting.value) return;

    submitting.value = true;
    error.value = null;
    try {
      session.value = (await apiSubmitAnswer(session.value.sessionId, answerIndex)).session;
      //SomeFeedback that the answer was correct or wrong could be implemented here
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to submit answer";
    } finally {
      submitting.value = false;
    }
  }

  async function loadSession(sessionId: string) {
    loading.value = true;
    error.value = null;
    try {
      session.value = await getSession(sessionId);
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to load session";
      session.value = null;
    } finally {
      loading.value = false;
    }
  }
  return {
    session: readonly(session),
    loading: readonly(loading),
    submitting: readonly(submitting),
    error: readonly(error),
    startNewSession,
    confirmAnswer,
    loadSession,
  };
}

