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

- The questions backend serves package and question data on `http://localhost:8081`
- The game backend serves game routes on `http://localhost:8080` and calls the questions backend over HTTP
- The questions backend uses its own Postgres database on `localhost:5432`
- The game backend uses its own Postgres database on `localhost:5433`
- The backend `.env` files are local-run defaults (`localhost` hosts and distinct ports), while Docker Compose overrides networking values for containers
- Inside Docker Compose, the game backend reaches the questions backend at `http://questions-backend:8080`
- Inside Docker Compose, the questions backend reaches `questions-postgres:5432` and the game backend reaches `game-postgres:5432`
- The frontend still uses `http://localhost:8080` for the game backend unless you explicitly wire it to call the questions API directly

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
