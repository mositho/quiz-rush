import { getAccessToken, refreshKeycloakToken } from "./keycloak"

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || "/api"

function buildApiUrl(path: string) {
  const normalizedBaseUrl = apiBaseUrl.endsWith("/")
    ? apiBaseUrl
    : `${apiBaseUrl}/`
  const normalizedPath = path.startsWith("/") ? path.slice(1) : path

  return new URL(
    normalizedPath,
    window.location.origin + normalizedBaseUrl
  ).toString()
}

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {}
): Promise<T> {
  const headers = new Headers(init.headers)
  const hasSession = await refreshKeycloakToken()

  if (hasSession) {
    const accessToken = getAccessToken()

    if (accessToken) {
      headers.set("Authorization", `Bearer ${accessToken}`)
    }
  }

  const response = await fetch(buildApiUrl(path), {
    ...init,
    headers
  })

  if (!response.ok) {
    throw new Error(`API request failed with status ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}
