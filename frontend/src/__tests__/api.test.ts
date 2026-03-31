import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { ApiError, apiFetch } from "../services/api"

vi.mock("../services/keycloak", () => ({
  refreshKeycloakToken: vi.fn().mockResolvedValue(false),
  getAccessToken: vi.fn().mockReturnValue("")
}))

describe("ApiError", () => {
  it("sets the name, message, status and body correctly", () => {
    const error = new ApiError("something went wrong", 404, "not found")

    expect(error.name).toBe("ApiError")
    expect(error.message).toBe("something went wrong")
    expect(error.status).toBe(404)
    expect(error.body).toBe("not found")
  })

  it("is an instance of Error", () => {
    const error = new ApiError("error", 500, "")

    expect(error).toBeInstanceOf(Error)
    expect(error).toBeInstanceOf(ApiError)
  })

  it("accepts an empty body", () => {
    const error = new ApiError("error", 400, "")

    expect(error.body).toBe("")
  })
})

describe("apiFetch", () => {
  const mockFetch = vi.fn()

  beforeEach(() => {
    vi.stubGlobal("fetch", mockFetch)
    Object.defineProperty(window, "location", {
      value: { origin: "http://localhost", href: "http://localhost/" },
      writable: true
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.clearAllMocks()
  })

  it("throws ApiError when the response is not ok", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 404,
      text: async () => "not found"
    })

    await expect(apiFetch("/leaderboard/demo")).rejects.toBeInstanceOf(ApiError)
  })

  it("includes the response status in the thrown ApiError", async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 403,
      text: async () => "forbidden"
    })

    let thrown: unknown
    try {
      await apiFetch("/api/results")
    } catch (e) {
      thrown = e
    }

    expect(thrown).toBeInstanceOf(ApiError)
    expect((thrown as ApiError).status).toBe(403)
    expect((thrown as ApiError).body).toBe("forbidden")
  })

  it("returns parsed JSON for a successful response", async () => {
    const responseData = { packageSlug: "demo", entries: [] }
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => responseData
    })

    const result = await apiFetch<typeof responseData>("/leaderboard/demo")

    expect(result).toEqual(responseData)
  })

  it("returns undefined for a 204 No Content response", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 204,
      json: async () => ({})
    })

    const result = await apiFetch("/results")

    expect(result).toBeUndefined()
  })

  it("does not send an Authorization header when there is no active session", async () => {
    const { refreshKeycloakToken } = await import("../services/keycloak")
    vi.mocked(refreshKeycloakToken).mockResolvedValue(false)

    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({})
    })

    await apiFetch("/leaderboard/demo")

    const requestInit = mockFetch.mock.calls[0][1] as RequestInit
    const headers = new Headers(requestInit.headers)
    expect(headers.has("Authorization")).toBe(false)
  })

  it("sends a Bearer Authorization header when the user has an active session", async () => {
    const { refreshKeycloakToken, getAccessToken } =
      await import("../services/keycloak")
    vi.mocked(refreshKeycloakToken).mockResolvedValue(true)
    vi.mocked(getAccessToken).mockReturnValue("test-access-token")

    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({})
    })

    await apiFetch("/results")

    const requestInit = mockFetch.mock.calls[0][1] as RequestInit
    const headers = new Headers(requestInit.headers)
    expect(headers.get("Authorization")).toBe("Bearer test-access-token")
  })

  it("passes through custom request options to fetch", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ status: "created" })
    })

    await apiFetch("/results", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({})
    })

    const [, requestInit] = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(requestInit.method).toBe("POST")
    expect(requestInit.body).toBe(JSON.stringify({}))
  })
})
