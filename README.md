# Quiz Rush

[![Frontend](https://github.com/mositho/quiz-rush/actions/workflows/frontend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/frontend-ci.yml)
[![Game Backend](https://github.com/mositho/quiz-rush/actions/workflows/game-backend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/game-backend-ci.yml)
[![Questions Backend](https://github.com/mositho/quiz-rush/actions/workflows/questions-backend-ci.yml/badge.svg?branch=main)](https://github.com/mositho/quiz-rush/actions/workflows/questions-backend-ci.yml)

[Miro](https://miro.com/app/board/uXjVGt7dlRA=/?focusWidget=3458764665738994468)

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

If you run either backend directly on your machine instead of inside Docker, update its `.env` file so the database host is reachable from your host OS. For example:

```env
DATABASE_URL=postgres://quiz_rush_questions:quiz_rush_questions_dev_password@localhost:5432/quiz_rush_questions?sslmode=disable
DATABASE_URL=postgres://quiz_rush_game:quiz_rush_game_dev_password@localhost:5433/quiz_rush_game?sslmode=disable
```

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
