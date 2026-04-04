import { ref, readonly } from "vue";
import type { Session, SubmitAnswerResult } from "@/types/apiResponses";
import type { StartSessionRequest } from "@/types/apiRequests";
import {
  finishSession as apiFinishSession,
  getSession,
  quitSession as apiQuitSession,
  startSession as apiStartSession,
  submitAnswer as apiSubmitAnswer,
} from "@/services/api";

const ACTIVE_SESSION_STORAGE_KEY = "quiz-rush.active-session-id";

const session = ref<Session | null>(null);
const loading = ref(false);
const submitting = ref(false);
const error = ref<string | null>(null);
const answerResult = ref<SubmitAnswerResult | null>(null);
const activeSessionId = ref<string | null>(readActiveSessionId());
const activeSessionStatus = ref<Session["status"] | null>(null);
const activeSession = ref<Session | null>(null);
const pendingSessionNavigationId = ref<string | null>(null);

function readActiveSessionId() {
  if (typeof window === "undefined") {
    return null;
  }

  return window.localStorage.getItem(ACTIVE_SESSION_STORAGE_KEY);
}

function writeActiveSessionId(sessionId: string | null) {
  if (typeof window === "undefined") {
    return;
  }

  if (sessionId) {
    window.localStorage.setItem(ACTIVE_SESSION_STORAGE_KEY, sessionId);
  } else {
    window.localStorage.removeItem(ACTIVE_SESSION_STORAGE_KEY);
  }
}

function syncTrackedActiveSession(nextSession: Session | null) {
  if (nextSession && nextSession.status !== "finished") {
    activeSessionId.value = nextSession.sessionId;
    activeSessionStatus.value = nextSession.status;
    activeSession.value = nextSession;
    writeActiveSessionId(nextSession.sessionId);
    return;
  }

  activeSessionStatus.value = nextSession?.status ?? null;
  activeSession.value = nextSession?.status !== "finished" ? nextSession : null;

  if (activeSessionId.value && nextSession?.sessionId === activeSessionId.value) {
    activeSessionId.value = null;
    writeActiveSessionId(null);
  }
}

export function resetGameSessionState() {
  answerResult.value = null;
}

function clearTrackedActiveSession() {
  activeSessionId.value = null;
  activeSessionStatus.value = null;
  activeSession.value = null;
  writeActiveSessionId(null);
}

export async function refreshTrackedActiveSession() {
  const trackedSessionId = activeSessionId.value ?? readActiveSessionId();
  if (!trackedSessionId) {
    activeSessionId.value = null;
    activeSessionStatus.value = null;
    return null;
  }

  try {
    const trackedSession = await getSession(trackedSessionId);
    syncTrackedActiveSession(trackedSession);
    return trackedSession;
  } catch {
    activeSessionId.value = null;
    activeSessionStatus.value = null;
    writeActiveSessionId(null);
    return null;
  }
}

export function useActiveSessionTracker() {
  async function abandonTrackedSession() {
    const trackedSessionId = activeSessionId.value ?? readActiveSessionId();
    if (!trackedSessionId) {
      clearTrackedActiveSession();
      return null;
    }

    try {
      const endedSession = await apiQuitSession(trackedSessionId);
      syncTrackedActiveSession(endedSession);
      return endedSession;
    } catch {
      clearTrackedActiveSession();
      return null;
    }
  }

  return {
    activeSessionId: readonly(activeSessionId),
    activeSessionStatus: readonly(activeSessionStatus),
    activeSession: readonly(activeSession),
    pendingSessionNavigationId: readonly(pendingSessionNavigationId),
    refreshTrackedActiveSession,
    abandonTrackedSession,
  };
}

export function useGameSession() {
  async function startNewSession(request: StartSessionRequest) {
    loading.value = true;
    error.value = null;
    answerResult.value = null;

    try {
      session.value = await apiStartSession(request);
      syncTrackedActiveSession(session.value);
      pendingSessionNavigationId.value = session.value.sessionId;
      return session.value;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to start new session";
      session.value = null;
      return null;
    } finally {
      loading.value = false;
    }
  }

  async function confirmAnswer(answerIndex: number) {
    if (!session.value?.sessionId || submitting.value) {
      return null;
    }

    submitting.value = true;
    error.value = null;

    try {
      const response = await apiSubmitAnswer(session.value.sessionId, answerIndex);
      session.value = response.session;
      answerResult.value = response.result;
      syncTrackedActiveSession(response.session);
      return response;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to submit answer";
      return null;
    } finally {
      submitting.value = false;
    }
  }

  async function loadSession(sessionId: string) {
    loading.value = true;
    error.value = null;

    try {
      session.value = await getSession(sessionId);
      syncTrackedActiveSession(session.value);
      if (pendingSessionNavigationId.value === sessionId) {
        pendingSessionNavigationId.value = null;
      }
      return session.value;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to load session";
      session.value = null;
      if (activeSessionId.value === sessionId) {
        syncTrackedActiveSession(null);
      }
      return null;
    } finally {
      loading.value = false;
    }
  }

  async function endSession() {
    if (!session.value?.sessionId) {
      return null;
    }

    loading.value = true;
    error.value = null;

    try {
      session.value = await apiFinishSession(session.value.sessionId);
      syncTrackedActiveSession(session.value);
      return session.value;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to finish session";
      return null;
    } finally {
      loading.value = false;
    }
  }

  async function quitSession() {
    if (!session.value?.sessionId) {
      return null;
    }

    loading.value = true;
    error.value = null;

    try {
      session.value = await apiQuitSession(session.value.sessionId);
      syncTrackedActiveSession(session.value);
      return session.value;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Failed to quit session";
      return null;
    } finally {
      loading.value = false;
    }
  }

  return {
    session: readonly(session),
    loading: readonly(loading),
    submitting: readonly(submitting),
    error: readonly(error),
    answerResult: readonly(answerResult),
    pendingSessionNavigationId: readonly(pendingSessionNavigationId),
    startNewSession,
    confirmAnswer,
    loadSession,
    endSession,
    quitSession,
    resetGameSessionState,
  };
}
