import Keycloak from "keycloak-js";
import { reactive } from "vue";

const keycloak = new Keycloak({
  url: import.meta.env.VITE_KEYCLOAK_URL,
  realm: import.meta.env.VITE_KEYCLOAK_REALM,
  clientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID,
});

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

export async function initKeycloak() {
  if (!initPromise) {
    initPromise = keycloak
      .init({
        onLoad: "check-sso",
        checkLoginIframe: false,
        pkceMethod: "S256",
        silentCheckSsoRedirectUri: `${window.location.origin}/silent-check-sso.html`,
      })
      .then((authenticated: boolean) => {
        syncAuthState();
        return authenticated;
      });
  }

  return initPromise;
}

export async function loginWithKeycloak() {
  await initKeycloak();
  await keycloak.login({
    redirectUri: window.location.href,
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
