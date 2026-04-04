import Keycloak from "keycloak-js";
import { reactive } from "vue";

function createKeycloakInstance() {
  return new Keycloak({
    url: import.meta.env.VITE_KEYCLOAK_URL,
    realm: import.meta.env.VITE_KEYCLOAK_REALM,
    clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID,
  });
}

let keycloak = createKeycloakInstance();

export const authState = reactive({
  initialized: false,
  authenticated: false,
  username: "",
});

let initPromise: Promise<boolean> | null = null;

function syncAuthState() {
  authState.initialized = true;
  authState.authenticated = Boolean(keycloak.authenticated);
  authState.username = keycloak.tokenParsed?.preferred_username || keycloak.tokenParsed?.name || "";
}

function clearAuthState() {
  authState.initialized = true;
  authState.authenticated = false;
  authState.username = "";
}

async function initWithSilentCheckSso() {
  return keycloak.init({
    onLoad: "check-sso",
    checkLoginIframe: false,
    pkceMethod: "S256",
    silentCheckSsoRedirectUri: `${window.location.origin}/silent-check-sso.html`,
  });
}

async function initWithRedirectCheckSso() {
  keycloak = createKeycloakInstance();
  return keycloak.init({
    onLoad: "check-sso",
    checkLoginIframe: false,
    pkceMethod: "S256",
  });
}

export async function initKeycloak() {
  if (!initPromise) {
    initPromise = initWithSilentCheckSso()
      .then((authenticated: boolean) => {
        syncAuthState();
        return authenticated;
      })
      .catch(async (error) => {
        console.warn(
          "Silent Keycloak check-sso failed; retrying with redirect-based session check.",
          error
        );

        try {
          const authenticated = await initWithRedirectCheckSso();
          syncAuthState();
          return authenticated;
        } catch (fallbackError) {
          console.warn(
            "Keycloak redirect check-sso failed; continuing without an active session.",
            fallbackError
          );
          clearAuthState();
          return false;
        }
      });
  }

  return initPromise;
}

function getRedirectUri(target?: string) {
  if (!target) {
    return window.location.href;
  }

  try {
    return new URL(target, window.location.origin).toString();
  } catch {
    return window.location.href;
  }
}

export async function loginWithKeycloak(redirectUri?: string) {
  await initKeycloak();
  await keycloak.login({
    redirectUri: getRedirectUri(redirectUri),
  });
}

export async function registerWithKeycloak(redirectUri?: string) {
  await initKeycloak();
  await keycloak.register({
    redirectUri: getRedirectUri(redirectUri),
  });
}

export async function logoutFromKeycloak() {
  await keycloak.logout({
    redirectUri: window.location.origin,
  });
}

export async function refreshKeycloakToken(minValidity = 30) {
  await initKeycloak();

  if (!keycloak.authenticated) {
    syncAuthState();
    return false;
  }

  try {
    await keycloak.updateToken(minValidity);
    syncAuthState();
    return true;
  } catch {
    keycloak.clearToken();
    syncAuthState();
    return false;
  }
}

export function getAccessToken() {
  return keycloak.token || "";
}
