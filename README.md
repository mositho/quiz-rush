# Quiz Rush

[![Frontend](https://github.com/mositho/quiz-rush/actions/workflows/frontend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/frontend-ci.yml)
[![Game Backend](https://github.com/mositho/quiz-rush/actions/workflows/game-backend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/game-backend-ci.yml)
[![Questions Backend](https://github.com/mositho/quiz-rush/actions/workflows/questions-backend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/questions-backend-ci.yml)

[Miro](https://miro.com/app/board/uXjVGt7dlRA=/?focusWidget=3458764665738994468)

## Architecture

High-level structure of the current codebase:

```mermaid
classDiagram
  class VueFrontend {
    Router
    Views
    useGameSession
  }
  class ApiClient {
    apiFetch()
    startSession()
    getSession()
    submitAnswer()
  }
  class KeycloakService {
    initKeycloak()
    refreshKeycloakToken()
    getAccessToken()
  }
  class FrontendNginx {
    serveFrontend()
    proxyApi()
    proxyAccount()
  }
  class GameBackend {
    ChiRouter
    AuthMiddleware
    GameHandler
  }
  class GameService {
    StartSession()
    loadQuestions()
  }
  class GameSessionEngine {
    NewSession()
    Sync()
    CurrentQuestion()
    SubmitAnswer()
    ScoreResult()
  }
  class SessionRepository {
    CreateSession()
    LoadSession()
    UpdateLockedSession()
    CreateScore()
  }
  class QuestionsAPIClient {
    LoadQuestionsBySetID()
  }
  class QuestionsBackend {
    ChiRouter
    SetHandler
    SetIndexer
  }
  class QuestionSetFiles {
    JSON files
  }
  class Postgres
  class Keycloak

  VueFrontend --> ApiClient : gameplay requests
  VueFrontend --> KeycloakService : sign in and token refresh
  VueFrontend --> FrontendNginx : served by
  ApiClient --> FrontendNginx : /api
  KeycloakService --> Keycloak : OIDC flow
  FrontendNginx --> GameBackend : proxy /api and /health
  FrontendNginx --> Keycloak : proxy /account
  GameBackend --> GameService : create new sessions
  GameBackend --> GameSessionEngine : sync and answer gameplay
  GameBackend --> SessionRepository : load and save session state
  GameService --> QuestionsAPIClient : fetch question sets
  GameService --> GameSessionEngine : build session state
  QuestionsAPIClient --> QuestionsBackend : /api/sets/{id}
  QuestionsBackend --> QuestionSetFiles : load questions
  SessionRepository --> GameSessionEngine : persist Session
  SessionRepository --> Postgres : sessions and scores
```

## Environment setup

The project is Docker-first. In the normal development flow, you do not need per-service `.env` files.

The env files are used like this:

- Optional root `.env`
  Docker Compose override file for local containerized runs. Use it only if you want to override defaults such as passwords, host ports, or `KEYCLOAK_*` / `VITE_*` values.
- `.env.prod`
  Production-only compose input used with `docker compose --env-file .env.prod ...`.
- `questions-backend/.env`
  Optional and only needed when running the questions API directly on your machine.
- `game-backend/.env`
  Optional and only needed when running the game API directly on your machine.
- `frontend/.env`
  Optional and only needed when running the frontend directly on your machine.

Tracked example files are included alongside them:

- `questions-backend/.env.example`
- `game-backend/.env.example`
- `frontend/.env.example`

### One-time repo setup

Enable the tracked Git hooks for this clone with:

```sh
make setup
```

### Recommended development workflow

Use Docker Compose with the dev override:

```sh
make dev
```

Equivalent raw command:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

Then open:

- Frontend: `http://localhost:5173`
- Game backend: `http://localhost:8080`
- Keycloak: `http://localhost:8082/account`

### Optional Docker overrides

Create a root `.env` only if you want to override Docker Compose defaults locally.

### Running with Docker Compose

Start everything with:

```sh
docker compose up --build
```

### Development compose override (Vue HMR)

For Docker-based frontend development with Vue hot module replacement, use the new dev override file.

Start the stack with:

```sh
make dev
```

Then open the frontend at `http://localhost:5173`.

Exposed development endpoints:

- Game backend: `http://localhost:8080`
- Keycloak: `http://localhost:8082/account`

Notes:

- Frontend source is bind-mounted from `./frontend` into the container.
- `node_modules` is kept in a Docker volume to avoid host/container binary conflicts.
- Frontend uses direct service URLs via `VITE_API_BASE_URL` and `VITE_KEYCLOAK_URL` (configured in `docker-compose.dev.yml`).
- Keycloak realm dev config allows `http://localhost:5173` as redirect origin for the `quiz-rush-app` client.
- Dev override uses a dedicated Keycloak Postgres volume to avoid stale realm config from other compose profiles.
- If an older dev Keycloak volume exists, recreate it once: `docker compose -f docker-compose.yml -f docker-compose.dev.yml down -v keycloak keycloak-postgres`.

### Production compose override

The production setup uses a second compose file that overrides local development defaults.

Use this command order so production settings win:

```sh
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Production-specific behavior in the override:

- No service publishes host ports
- Keycloak runs in production mode with `start --import-realm`
- Required secrets and URLs fail fast when missing
- Keycloak imports the file selected by `KEYCLOAK_IMPORT_FILE`
- `game-backend` and `questions-backend` are built as container images (no source-code bind mount required)
- Restart policy is bounded (`on-failure:5`) to prevent endless crash loops during debugging
- `game-backend` retries OIDC startup to wait for Keycloak readiness

Minimum required variables in `.env.prod`:

- `GAME_POSTGRES_PASSWORD`
- `KEYCLOAK_DB_PASSWORD`
- `KEYCLOAK_ADMIN_PASSWORD`
- `KEYCLOAK_HOSTNAME`
- `KEYCLOAK_IMPORT_FILE=./keycloak/realm-export.prod.json`
- `KEYCLOAK_ISSUER_URL`
- `CORS_ALLOWED_ORIGIN`
- `VITE_KEYCLOAK_URL`
- `AUTH_INIT_MAX_WAIT=180s`
- `AUTH_INIT_RETRY_INTERVAL=5s`

Update `keycloak/realm-export.prod.json` with your real frontend domain before deployment.

If you changed Postgres usernames/database names and see errors like `FATAL: role ... does not exist`, recreate the database volumes once:

```sh
docker compose down -v
docker compose up --build
```

Important details:

- The backends connect to Postgres with Docker service hostnames
- Game backend integration tests use Testcontainers with `postgres:18-alpine` and require Docker locally and in CI
- The frontend is built as static assets and served by Nginx
- Nginx is the single public entry point on `http://localhost`
- All requests starting with `/api` are proxied by Nginx to the game backend inside Docker
- All requests starting with `/account` are proxied by Nginx to Keycloak inside Docker
- Keycloak is served behind the same public origin under `/account`

Quick checks after startup:

- Frontend: `http://localhost`
- API health: `http://localhost/health`
- Keycloak discovery: `http://localhost/account/realms/quiz-rush/.well-known/openid-configuration`

### Keycloak defaults

Docker Compose starts Keycloak under `/account` with preconfigured defaults so the frontend and backend use the same public base URL.

- Public Keycloak base URL: `http://localhost/account`
- Realm: `quiz-rush`
- Client ID: `quiz-rush-app`
- Self-registration is enabled (`Sign up` on the login page)
- Login with email is disabled (`username` login only)

If you change `keycloak/realm-export.json`, recreate Keycloak data so import is applied again:

```sh
docker compose down -v
docker compose up --build
```

For local development, Docker Compose provides fallback defaults for these sensitive values:

- `KEYCLOAK_ADMIN_PASSWORD=changeme`
- `KEYCLOAK_DB_PASSWORD=changeme`

Override them in the root `.env` before sharing the environment or using it outside local development.

### Running services outside Docker

Host-run workflows are optional. Keep them only if they help your debugging or test loop.

Copy examples only for the services you want to run directly:

```sh
cp game-backend/.env.example game-backend/.env
cp questions-backend/.env.example questions-backend/.env
cp frontend/.env.example frontend/.env
```

If you run services directly on your machine instead of inside Docker:

- `game-backend` needs Postgres. Use this in `game-backend/.env`:

```env
DATABASE_URL=postgres://quiz_rush_game:quiz_rush_game@localhost:5433/quiz_rush_game?sslmode=disable
```

- `questions-backend` is file-backed (`questionsets/*.json`) and does not require a Postgres `DATABASE_URL`.

For local game backend to questions backend calls outside Docker, keep this in `game-backend/.env`:

```env
QUESTIONS_API_BASE_URL=http://localhost:8081
```

For local frontend development outside Docker, point the app to the standalone services in `frontend/.env`, for example:

```env
VITE_API_BASE_URL=http://localhost:8080/api
VITE_KEYCLOAK_URL=http://localhost/account
VITE_KEYCLOAK_REALM=quiz-rush
VITE_KEYCLOAK_CLIENT_ID=quiz-rush-app
```

## Bruno API Collections


- [bruno/game-backend](/home/moritz/workspace/school/quiz-rush/bruno/game-backend)
  Anonymous smoke flow for the game API through the public Docker entrypoint.
- [bruno/questions-backend](/home/moritz/workspace/school/quiz-rush/bruno/questions-backend)
  Smoke flow for the questions API on its direct Docker-exposed port.

There is also a short overview in [bruno/README.md](/home/moritz/workspace/school/quiz-rush/bruno/README.md).

Recommended usage:

1. Start the dev stack:
   `make dev`
2. Open either collection folder in Bruno.
3. Select the matching environment for that collection.
   For the game backend, `direct` matches the dev compose stack best.
4. Run the requests in order or run the whole collection.

Current collection coverage:

- Game backend: smoke flow, Keycloak-backed auth setup, authenticated session flow, user scores, score lookup, user stats, leaderboard
- Questions backend: `00 Smoke` flow with health, list sets, fetch set questions

For the authenticated Bruno requests, the dev realm export enables direct grants on `quiz-rush-app` and includes a reusable test user in [keycloak/realm-export.json](/home/moritz/workspace/school/quiz-rush/keycloak/realm-export.json). Production stays stricter in [keycloak/realm-export.prod.json](/home/moritz/workspace/school/quiz-rush/keycloak/realm-export.prod.json), where direct grants remain disabled. If Keycloak was already initialized before this change, recreate the dev Keycloak data once so the imported realm is applied again:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml down -v keycloak keycloak-postgres
make dev
```

In Bruno, keep `username` and `password` in the selected game-backend environment, or mark them as secrets in Bruno's UI if your installed version supports that. The login request then populates the runtime `accessToken` automatically.
