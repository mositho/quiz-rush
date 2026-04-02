<template>
  <div class="login-register-container">
    <h1>Login/Register</h1>
    <form>
      <h2>login</h2>
      <label for="username">Username:</label>
      <input id="username" v-model="username" type="text" name="username" />
      <label for="password">Password:</label>
      <input id="password" v-model="password" type="password" name="password" />
    </form>
    <button type="button" @click="Login">Login</button>
    <button type="button" @click="Register">Register</button>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

const username = ref("");
const password = ref("");

const login = async () => {
  try {
    const response = await fetch("/api/login", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        username: username.value,
        password: password.value,
      }),
    });
    if (response.ok) {
      // Handle successful login
    } else {
      // Handle login error
      //Login failed, show error message ask to register or try again
    }
  } catch (error) {
    console.error("Error during login:", error);
  }
};
const register = async () => {
  try {
    const response = await fetch("/api/register", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ username, password }),
    });
    if (response.ok) {
      // Handle successful registration
    } else {
      //just a comment to pass for now
    }
  } catch (error) {
    console.error("Error during registration:", error);
  }
};

function Login() {
  return login();
}

function Register() {
  return register();
}
</script>

<style scoped>
@import "../assets/styles/base.css";
form {
  margin-bottom: 0.5rem;
}
button {
  margin-top: 0;
}
</style>
