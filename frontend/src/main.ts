import { createApp } from "vue";
import App from "./App.vue";
import "./assets/styles/index.css";
import { initCurrentUser } from "./composables/useCurrentUser";
import { router } from "./router";
import { initKeycloak } from "./services/keycloak";

createApp(App).use(router).mount("#app");

void initKeycloak()
  .then(() => initCurrentUser())
  .catch((error) => {
    console.error("Failed to initialize auth", error);
  });
