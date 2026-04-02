import { createRouter, createWebHistory } from "vue-router";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "home",
      component: () => import("../views/HomeView.vue"),
    },
    {
      path: "/login",
      name: "login",
      component: () => import("../views/LoginRegisterView.vue"),
    },
    {
      path: "/game/:sessionId",
      name: "gameplay",
      component: () => import("../views/GameplayView.vue"),
    },
  ],
});
