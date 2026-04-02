# Quiz Rush

[![Frontend](https://github.com/mositho/quiz-rush/actions/workflows/frontend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/frontend-ci.yml)
[![Game Backend](https://github.com/mositho/quiz-rush/actions/workflows/game-backend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/game-backend-ci.yml)
[![Questions Backend](https://github.com/mositho/quiz-rush/actions/workflows/questions-backend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/questions-backend-ci.yml)

[Miro](https://miro.com/app/board/uXjVGt7dlRA=/?focusWidget=3458764665738994468)

## Architecture

Application structure from the current codebase:

```mermaid
classDiagram
  class VueApp {
    +bootstrap()
  }
  class Router {
    +showHomeView()
    +showLoginView()
  }
  class HomeView {
    +loadLeaderboard()
    +createResult()
  }
  class LoginRegisterView {
    +login()
    +register()
  }
  class ApiClient {
    +apiFetch(path, init)
    +buildApiUrl(path)
  }
  class KeycloakService {
    +initKeycloak()
    +loginWithKeycloak()
    +logoutFromKeycloak()
    +refreshKeycloakToken()
    +getAccessToken()
  }
  class FrontendNginx {
    +serveIndexHtml()
    +proxyApi()
    +proxyHealth()
    +proxyAccount()
  }
  class GameRouter {
    +health()
    +createSession()
    +getSessionById()
    +submitAnswer()
    +finishSession()
    +quitSession()
    +linkAccount()
    +getScoreById()
    +getLeaderboards()
    +getCurrentUser()
    +getUserScores()
    +getUserStats()
  }
  class OIDCAuthMiddleware {
    +authenticateRequest()
    +AuthenticatedUserFromContext()
  }
  class GameHandler {
    +StartSession()
    +GetSession()
    +SubmitAnswer()
    +FinishSession()
    +QuitSession()
    +LinkAccount()
    +GetScore()
    +GetLeaderboard()
    +GetCurrentUser()
    +GetUserScores()
    +GetUserStats()
  }
  class GameService {
    +StartSession(ctx, duration, setIDs, now)
    +BuildConfigurationKey(duration, setIDs)
    +loadQuestions(ctx, setIDs)
  }
  class Session {
    +Status
    +FinishReason
    +StartedAt
    +EndsAt
    +CooldownUntil
    +CurrentScore
    +CurrentQuestionIndex
    +SessionQuestions
    +Sync(now)
    +CurrentQuestion(now)
    +SubmitAnswer(now, selectedAnswerIndex)
    +Finish(now, reason)
    +ScoreResult()
  }
  class SessionRepository {
    +EnsureUserProfile()
    +CreateSession()
    +LoadSession()
    +UpdateSession()
    +CreateScore()
    +LinkAnonymousSessionScore()
    +GetScore()
    +GetLeaderboard()
    +GetUserScores()
    +GetUserStats()
  }
  class QuestionsAPIClient {
    +LoadQuestionsBySetID(ctx, setID)
  }
  class QuestionsRouter {
    +health()
    +listSets()
    +getSetById()
  }
  class QuestionsHandler {
    +GetSets()
    +GetSetQuestions()
  }
  class SetIndexer {
    +LoadAllMetadata()
    +ListSets()
    +LoadQuestionsByID(id)
  }
  class Keycloak
  class Postgres
  class QuestionSetFiles {
    +loadJsonFiles()
  }

  VueApp --> Router : mounts
  Router --> HomeView : route /
  Router --> LoginRegisterView : route /login
  HomeView --> ApiClient : uses
  HomeView --> KeycloakService : sign in/out state
  LoginRegisterView --> FrontendNginx : calls /api/login,/api/register
  ApiClient --> KeycloakService : bearer token refresh
  VueApp --> FrontendNginx : served by
  ApiClient --> FrontendNginx : /api requests
  KeycloakService --> Keycloak : OIDC browser flow
  FrontendNginx --> GameRouter : proxy /api,/health
  FrontendNginx --> Keycloak : proxy /account
  GameRouter --> OIDCAuthMiddleware : optional auth
  GameRouter --> GameHandler : route handlers
  GameHandler --> GameService : start/load gameplay
  GameHandler --> SessionRepository : persistence
  GameService --> QuestionsAPIClient : load set questions
  QuestionsAPIClient --> QuestionsRouter : GET /api/sets/{id}
  QuestionsRouter --> QuestionsHandler : route handlers
  QuestionsHandler --> SetIndexer : list/load sets
  SetIndexer --> QuestionSetFiles : read JSON files
  SessionRepository --> Postgres : store sessions,scores,profiles
  GameHandler --> Session : mutate and serialize
  GameService --> Session : creates
```

Session start and answer flow:

```mermaid
sequenceDiagram
  participant U as User
  participant V as Vue HomeView
  participant K as KeycloakService
  participant N as Frontend Nginx
  participant R as GameRouter
  participant M as OIDCAuthMiddleware
  participant H as GameHandler
  participant Svc as GameService
  participant Sess as Session
  participant Q as QuestionsAPIClient
  participant QB as Questions Backend
  participant Repo as SessionRepository
  participant DB as Postgres

  U->>V: Start session
  V->>K: refreshKeycloakToken()
  K-->>V: access token or anonymous state
  V->>N: POST /api/game/sessions
  N->>R: proxy request
  R->>M: inspect Authorization header
  M-->>R: authenticated user or anonymous request
  R->>H: StartSession()
  H->>Svc: StartSession(ctx, duration, setIDs, now)
  Svc->>Q: LoadQuestionsBySetID(setID)
  Q->>QB: GET /api/sets/{id}
  QB-->>Q: question list
  Q-->>Svc: questions
  Svc-->>H: Session
  H->>Repo: EnsureUserProfile() when signed in
  H->>Repo: CreateSession(session,...)
  Repo->>DB: insert game_sessions + game_session_questions
  DB-->>Repo: committed
  H-->>V: 201 session snapshot + currentQuestion

  U->>V: Submit answer
  V->>K: refreshKeycloakToken()
  V->>N: POST /api/game/sessions/{id}/answers
  N->>R: proxy request
  R->>M: validate bearer token when present
  R->>H: SubmitAnswer()
  H->>Repo: LoadSession(sessionId)
  Repo->>DB: read session + question rows
  DB-->>Repo: persisted state
  Repo-->>H: Session
  H->>Sess: SubmitAnswer(now, selectedAnswerIndex)
  Sess-->>H: AnswerResult
  H->>Repo: UpdateSession(session)
  Repo->>DB: update session + question rows
  alt session finished
    H->>Repo: CreateScore(session.ScoreResult())
    Repo->>DB: insert game_scores
  end
  H-->>V: 200 session + answer result
```

## Recommended delivery pipeline

The repo already has fast service-specific CI workflows. A good next step is to keep those path-based checks, then add one end-to-end gate that proves the whole stack still works together before anything is released.

- Pull requests: run only the affected service jobs (`frontend`, `game-backend`, `questions-backend`) for quick feedback.
- Integration gate: after service checks pass, build the Docker Compose stack and run smoke tests against the real multi-service setup with Postgres and Keycloak.
- Release on `main`: publish versioned Docker images for `frontend`, `game-backend`, and `questions-backend` only after the integration gate passes.
- Deployment: roll the exact same image digests to staging first, rerun smoke tests, then promote to production with a manual approval step.
- Operations hygiene: keep Dependabot for dependency bumps and run database migrations as part of backend deployment before traffic is shifted.

Recommended CI/CD sequence:

```mermaid
sequenceDiagram
  actor Dev as Developer
  participant GH as GitHub Actions
  participant F as Frontend CI
  participant Q as Questions CI
  participant G as Game CI
  participant I as Integration Gate
  participant C as Docker Compose Stack
  participant R as Container Registry
  participant S as Staging
  participant P as Production

  Dev->>GH: Push branch or open pull request
  par Changed frontend files
    GH->>F: pnpm install, lint, format:check, build
    F-->>GH: pass/fail
  and Changed questions service files
    GH->>Q: gofmt, golangci-lint, go test ./...
    Q-->>GH: pass/fail
  and Changed game service files
    GH->>G: gofmt, golangci-lint, go test ./...
    G-->>GH: pass/fail
  end

  GH->>I: Start only if all required checks pass
  I->>C: Build and boot frontend + backends + Postgres + Keycloak
  I->>C: Run smoke tests for health, auth, sets, session flow
  C-->>I: integration result

  alt Branch is main and integration passed
    GH->>R: Build and publish immutable Docker images
    GH->>S: Deploy same image digests to staging
    S-->>GH: post-deploy smoke tests pass
    GH->>P: Manual approval, then promote same digests
    P-->>Dev: production deployment complete
  else Pull request or non-main branch
    GH-->>Dev: report CI status only
  end
```

## Environment setup

The project uses three tracked env files plus one optional local override file:

- Optional root `.env` for Docker Compose variable overrides such as DB passwords, host ports, `QUESTIONS_API_BASE_URL`, `VITE_API_BASE_URL`, `KEYCLOAK_*`, and `CORS_ALLOWED_ORIGIN`
- `questions-backend/.env` for the questions API when running it outside Docker
- `game-backend/.env` for the game API when running it outside Docker
- `frontend/.env` for frontend variables

Tracked example files are included alongside them:

- `questions-backend/.env.example`
- `game-backend/.env.example`
- `frontend/.env.example`

### First-time setup

1. Optional: create a root `.env` only if you want to override Docker Compose defaults locally.
2. Copy the questions backend example:
   `cp questions-backend/.env.example questions-backend/.env`
3. Copy the game backend example:
   `cp game-backend/.env.example game-backend/.env`
4. Copy the frontend example:
   `cp frontend/.env.example frontend/.env`

### Running with Docker Compose

Start everything with:

```sh
docker compose up --build
```

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
