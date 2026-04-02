# Bruno Collections

Recommended structure: keep one Bruno collection per backend.

- `bruno/game-backend`
  Game API smoke, auth, and score/leaderboard tests.
- `bruno/questions-backend`
  Questions API tests with the same folder structure, currently only a smoke layer.

Why this split:

- each backend gets its own environment variables and base URL
- collections stay small and focused
- auth-heavy game flows do not complicate the questions API collection
- future CI can run collections independently

Recommended folder structure inside each collection:

- `00 Smoke`
  Fast sanity checks that prove the backend is up and the main path works.
- `01+`
  Additional areas as the collection grows, such as auth, integration, or negative cases.

Sensitive values:

- store static credentials like `username` and `password` as Bruno environment variables by default, or mark them as secrets in Bruno's UI if your installed version supports it
- keep short-lived values like `accessToken` as runtime variables created during request execution

Current environments:

- `bruno/game-backend/environments/public.bru`
  Uses `http://localhost` when the game API is reachable through a public entrypoint like docker
- `bruno/game-backend/environments/direct.bru`
  Uses the game backend directly on `http://localhost:8080` plus Keycloak on `http://localhost:8082/account`
- `bruno/questions-backend/environments/direct.bru`
  Uses the direct questions backend port at `http://localhost:8081`
