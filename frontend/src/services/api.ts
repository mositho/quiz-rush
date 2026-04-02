import { getAccessToken, refreshKeycloakToken } from "./keycloak";
import type { StartSessionRequest } from "@/types/apiRequests";
import type { Session, SubmitAnswerResult } from "@/types/apiResponses";

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
      body: JSON.stringify({ answerIndex }),
    }
  );
}
