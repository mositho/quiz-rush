import { createApp } from "vue";
import App from "./App.vue";
import "./assets/styles/index.css";
import { router } from "./router";
import { initKeycloak } from "./services/keycloak";

async function bootstrap() {
  await initKeycloak();

  createApp(App).use(router).mount("#app");
}

void bootstrap();
