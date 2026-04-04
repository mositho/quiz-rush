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
      path: "/game/:sessionId",
      name: "game",
      component: () => import("../views/GameView.vue"),
    },
    {
      path: "/profile",
      name: "my-profile",
      component: () => import("../views/ProfileView.vue"),
    },
    {
      path: "/profile/:publicUserId",
      name: "profile",
      component: () => import("../views/ProfileView.vue"),
      props: true,
    },
    {
      path: "/leaderboard",
      name: "leaderboard",
      component: () => import("../views/LeaderboardView.vue"),
    },
  ],
});
