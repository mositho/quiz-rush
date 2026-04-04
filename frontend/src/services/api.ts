import type { StartSessionRequest, UpdateCurrentUserRequest } from "@/types/apiRequests";
import type {
  LeaderboardList,
  LinkAccountResult,
  PublicUser,
  QuestionSet,
  ScoreDetail,
  Session,
  SubmitAnswerResult,
  UserScoreList,
  UserStatsProfile,
} from "@/types/apiResponses";
import { getAccessToken, refreshKeycloakToken } from "./keycloak";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || "/api";

export class ApiError extends Error {
  status: number;
  body: string;

  constructor(message: string, status: number, body: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

function buildApiUrl(path: string) {
  const normalizedBaseUrl = apiBaseUrl.endsWith("/") ? apiBaseUrl : `${apiBaseUrl}/`;
  const normalizedPath = path.startsWith("/") ? path.slice(1) : path;
  const hasAbsoluteBaseUrl = /^https?:\/\//i.test(normalizedBaseUrl);

  if (hasAbsoluteBaseUrl) {
    return new URL(normalizedPath, normalizedBaseUrl).toString();
  }

  return new URL(normalizedPath, window.location.origin + normalizedBaseUrl).toString();
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const hasSession = await refreshKeycloakToken();

  if (hasSession) {
    const accessToken = getAccessToken();

    if (accessToken) {
      headers.set("Authorization", `Bearer ${accessToken}`);
    }
  }

  const response = await fetch(buildApiUrl(path), {
    ...init,
    headers,
  });

  if (!response.ok) {
    const body = await response.text();
    throw new ApiError(`API request failed with status ${response.status}`, response.status, body);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export async function startSession(request: StartSessionRequest): Promise<Session> {
  return apiFetch<Session>("/game/sessions", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });
}

export async function getSession(sessionId: string): Promise<Session> {
  return apiFetch<Session>(`/game/sessions/${sessionId}`);
}

export async function getQuestionSets(): Promise<QuestionSet[]> {
  return apiFetch<QuestionSet[]>("/game/question-sets");
}

export async function submitAnswer(
  sessionId: string,
  answerIndex: number
): Promise<{ session: Session; result: SubmitAnswerResult }> {
  return apiFetch<{ session: Session; result: SubmitAnswerResult }>(
    `/game/sessions/${sessionId}/answers`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ selectedAnswerIndex: answerIndex }),
    }
  );
}

export async function finishSession(sessionId: string): Promise<Session> {
  return apiFetch<Session>(`/game/sessions/${sessionId}/finish`, {
    method: "POST",
  });
}

export async function quitSession(sessionId: string): Promise<Session> {
  return apiFetch<Session>(`/game/sessions/${sessionId}/quit`, {
    method: "POST",
  });
}

export async function getCurrentUser(): Promise<PublicUser> {
  return apiFetch<PublicUser>("/game/users/me");
}

export async function updateCurrentUser(request: UpdateCurrentUserRequest): Promise<PublicUser> {
  return apiFetch<PublicUser>("/game/users/me", {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });
}

export async function getUserStats(publicUserId: string): Promise<UserStatsProfile> {
  return apiFetch<UserStatsProfile>(`/game/users/${publicUserId}/stats`);
}

export async function getUserScores(publicUserId: string): Promise<UserScoreList> {
  return apiFetch<UserScoreList>(`/game/users/${publicUserId}/scores`);
}

export async function getScore(scoreId: string): Promise<ScoreDetail> {
  return apiFetch<ScoreDetail>(`/game/scores/${scoreId}`);
}

export async function getLeaderboard(
  configurationKey?: string,
  limit = 20
): Promise<LeaderboardList> {
  const query = new URLSearchParams();
  query.set("limit", String(limit));

  if (configurationKey) {
    query.set("configurationKey", configurationKey);
  }

  return apiFetch<LeaderboardList>(`/game/leaderboards?${query.toString()}`);
}

export async function linkAccount(sessionId: string): Promise<LinkAccountResult> {
  return apiFetch<LinkAccountResult>(`/game/sessions/${sessionId}/link-account`, {
    method: "POST",
  });
}
