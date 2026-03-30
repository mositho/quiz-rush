# Quiz Rush

## Environment setup

The project uses four env files:

- Root `.env` for the two Postgres container credentials used by Docker Compose
- `questions-backend/.env` for the questions API
- `game-backend/.env` for the game API
- `frontend/.env` for Vite frontend variables

Tracked example files are included alongside them:

- `.env.example`
- `questions-backend/.env.example`
- `game-backend/.env.example`
- `frontend/.env.example`

### First-time setup

1. Copy the example files if you want a fresh local setup:
   `cp .env.example .env`
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

Important details:

- The backend connects to Postgres with the Docker service hostname `postgres`
- The frontend is built as static assets and served by Nginx
- Nginx is the single public entry point on `http://localhost:5173`
- All requests starting with `/api` are proxied by Nginx to the backend service inside Docker
- The backend is not exposed directly on a host port in this setup

Quick checks after startup:

- Frontend: `http://localhost:5173`
- API health: `http://localhost:5173/api/health`

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
