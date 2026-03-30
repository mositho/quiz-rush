<template>
  <main class="home-view">
    <h1>Quiz Rush</h1>
    <p class="home-view__status">
      <span v-if="authState.initialized && authState.authenticated">
        Signed in as {{ authState.username }}.
      </span>
      <span v-else-if="authState.initialized">Not signed in.</span>
      <span v-else>Checking session...</span>
    </p>
    <div class="home-view__actions">
      <button
        v-if="!authState.authenticated"
        type="button"
        @click="handleLogin"
      >
        Sign in with Keycloak
      </button>
      <button v-else type="button" @click="handleLogout">Sign out</button>
    </div>
  </main>
</template>

<script setup lang="ts">
import {
  authState,
  loginWithKeycloak,
  logoutFromKeycloak
} from "../services/keycloak"

function handleLogin() {
  void loginWithKeycloak()
}

function handleLogout() {
  void logoutFromKeycloak()
}
</script>

<style scoped>
.home-view {
  min-height: 100vh;
  display: grid;
  place-content: center;
  gap: 1rem;
  text-align: center;
}

.home-view__status {
  margin: 0;
}

.home-view__actions {
  display: flex;
  justify-content: center;
}

.home-view__actions button {
  padding: 0.8rem 1.2rem;
  border: 0;
  border-radius: 999px;
  background: #101828;
  color: #fff;
  cursor: pointer;
}
</style>
