# Quiz Rush

## Environment setup

The project uses three env files:

- Root `.env` for the Postgres container credentials used by Docker Compose
- `backend/.env` for the Go API
- `frontend/.env` for Vite frontend variables

Tracked example files are included alongside them:

- `.env.example`
- `backend/.env.example`
- `frontend/.env.example`

### First-time setup

1. Copy the example files if you want a fresh local setup:
   `cp .env.example .env`
2. Copy the backend example:
   `cp backend/.env.example backend/.env`
3. Copy the frontend example:
   `cp frontend/.env.example frontend/.env`

### Running with Docker Compose

Start everything with:

```sh
docker compose up --build
```

Important details:

- The backend connects to Postgres with the Docker service hostname `postgres`
- The frontend uses `http://localhost:8080` because the browser talks to the backend through the host machine, not the Docker service name

### Running services outside Docker

If you run the backend directly on your machine instead of inside Docker, update `backend/.env` so the database host is reachable from your host OS, for example:

```env
DATABASE_URL=postgres://quiz_rush:quiz_rush_dev_password@localhost:5432/quiz_rush?sslmode=disable
```
