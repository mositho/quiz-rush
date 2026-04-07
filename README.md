# Quiz Rush

[![CI](https://github.com/mositho/quiz-rush/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/ci.yml)
[![Build And Deploy Images](https://github.com/mositho/quiz-rush/actions/workflows/build-image.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/build-image.yml)

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

## Pipeline

The GitHub Actions pipeline has two linked workflows:

- `ci.yml` runs checks on pushes and pull requests for the app, backend, compose, and workflow files.
- `build-image.yml` runs only after a successful `CI` workflow on a `main` branch push, then builds and pushes Docker images and triggers a Coolify redeploy webhook.

```mermaid
sequenceDiagram
  autonumber
  actor Dev as Developer
  participant GH as GitHub
  participant CI as CI workflow
  participant Build as Build workflow
  participant Scan as Secret Scanning
  participant FE as Frontend
  participant GB as Game Backend
  participant QB as Questions Backend
  participant GO as Reusable Go workflow
  participant Reg as Container Registry
  participant PR as Pull Request
  participant Coolify as Coolify

  Dev->>GH: Push or open/update PR<br/>for frontend, backend, compose, or workflow files
  GH->>CI: Trigger top-level CI workflow
  Note over CI: Concurrency group cancels older runs<br/>for the same PR/ref

  par Secret scan
    CI->>Scan: Run gitleaks scan
    Scan-->>CI: Scan result
  and Frontend checks
    CI->>FE: Run frontend checks
    FE->>FE: Install, audit, lint, format check, build
    FE-->>CI: Frontend result
  and Game backend checks
    CI->>GB: Start game backend workflow
    GB->>GO: Reuse Go backend pipeline
    GO->>GO: Format, lint, vuln check, tests, coverage
    alt Pull request
      GO->>PR: Update coverage comment
    end
    GO-->>GB: Game backend result
    GB-->>CI: Backend result
  and Questions backend checks
    CI->>QB: Start questions backend workflow
    QB->>GO: Reuse Go backend pipeline
    GO->>GO: Format, lint, vuln check, tests, coverage
    alt Pull request
      GO->>PR: Update coverage comment
    end
    GO-->>QB: Questions backend result
    QB-->>CI: Backend result
  end

  CI-->>GH: Combined CI status
  GH-->>Dev: CI result on commit or PR

  alt Successful push to main
    GH->>Build: Trigger via workflow_run after CI completes
    Note over Build: Runs only if CI concluded successfully<br/>and the original event was a push
    Build->>Build: Checkout commit and prepare Docker build
    Build->>Reg: Build and push frontend and backend images
    Build->>Reg: Tag and push latest images
    Build->>Coolify: POST deploy webhook
    Note over Coolify: Redeploy starts on the server
  end
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

Compose files are split by concern:

- `docker-compose.yml`
  Unopinionated base definitions with shared service wiring, environment defaults, volumes, and dependencies.
- `docker-compose.dev.yml`
  Local development runtime overrides with bind mounts, host ports, and HMR.
- `docker-compose.build.yml`
  Image build definitions used by CI to build and push application images.
- `docker-compose.coolify.yml`
  Coolify/production runtime overrides that pull prebuilt images from the registry.

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

The base file on its own is intentionally incomplete for the application services. Combine it with one of the environment-specific overrides above.

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

### Coolify compose override

The Coolify setup uses a dedicated runtime override that pulls the prebuilt application images from the registry.

Use this command order so production settings win:

```sh
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.coolify.yml up -d
```

Coolify-specific behavior in the override:

- No service publishes host ports
- Keycloak runs in production mode with `start --import-realm`
- Required secrets and URLs fail fast when missing
- Keycloak imports the file selected by `KEYCLOAK_IMPORT_FILE`
- `frontend`, `game-backend` and `questions-backend` are pulled from the container registry
- All app services can be pinned to one release via `IMAGE_TAG` (for example the Git commit SHA)
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
- `AUTH_INIT_MAX_WAIT=180s`
- `AUTH_INIT_RETRY_INTERVAL=5s`

Optional image variables in `.env.prod`:

- `IMAGE_TAG=latest`
- `FRONTEND_IMAGE=ghcr.io/mositho/quiz-rush-frontend`
- `GAME_BACKEND_IMAGE=ghcr.io/mositho/quiz-rush-game-backend`
- `QUESTIONS_BACKEND_IMAGE=ghcr.io/mositho/quiz-rush-questions-backend`

Build images in CI with the dedicated build override:

```sh
docker compose -f docker-compose.yml -f docker-compose.build.yml build frontend game-backend questions-backend
```

Update `keycloak/realm-export.prod.json` with your real frontend domain before deployment.

If you changed Postgres usernames/database names and see errors like `FATAL: role ... does not exist`, recreate the database volumes once:

```sh
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.coolify.yml down -v
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.coolify.yml up -d
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

- Frontend: `https://your-public-domain`
- API health: `https://your-public-domain/health`
- Keycloak discovery: `https://your-public-domain/account/realms/quiz-rush/.well-known/openid-configuration`

### Keycloak defaults

Docker Compose starts Keycloak under `/account` with preconfigured defaults so the frontend and backend use the same public base URL.

- Public Keycloak base URL: `http://localhost/account`
- Realm: `quiz-rush`
- Client ID: `quiz-rush-app`
- Self-registration is enabled (`Sign up` on the login page)
- Login with email is disabled (`username` login only)

If you change `keycloak/realm-export.json`, recreate Keycloak data so import is applied again:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml down -v
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
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
