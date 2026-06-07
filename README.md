# Materials Mind

Materials Mind is a full-stack materials recommendation assistant that combines a React frontend, a Go backend, PostgreSQL with vector search, Redis caching, and Gemini-powered retrieval and response generation.

## What It Does

- Accepts a user materials query from the frontend.
- Authenticates requests with an HttpOnly session cookie.
- Uses Redis for rate limiting and caching.
- Searches PostgreSQL vector embeddings for relevant materials.
- Generates structured, grounded recommendations with Gemini.
- Streams the final response back to the UI.

## Tech Stack

- Frontend: React, Vite, TypeScript
- Backend: Go, Gin
- Database: PostgreSQL + pgvector
- Cache: Redis Stack
- LLM: Gemini API

## Local Setup

### 1. Prerequisites

- Go installed
- Node.js installed
- Docker installed
- Gemini API key

### 2. Environment

Create a `.env` file in the repository root:

```env
GEMINI_API_KEY=your_gemini_api_key
DATABASE_URL=postgres://postgres:password123@localhost:5433/materialmind
PORT=8080
JWT_SECRET=replace_with_a_long_random_secret
GOOGLE_CLIENT_ID=your_google_oauth_client_id.apps.googleusercontent.com
FRONTEND_ORIGIN=http://localhost:5173
VITE_API_URL=http://localhost:8080/api
VITE_GOOGLE_CLIENT_ID=your_google_oauth_client_id.apps.googleusercontent.com
```

### 3. Start PostgreSQL and Redis

Use the provided Docker Compose file:

```bash
docker compose up -d
```

### 4. Start the Backend

```bash
cd backend
go run .
```

### 5. Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

Open the Vite dev server URL in your browser.

## Project Structure

```text
backend/   Go API, middleware, database access, Gemini integration
data/      Material datasets, ETL scripts, schema
frontend/  React application and UI components
```

## Notes

- The repository ignores `.env`, `node_modules`, build artifacts, logs, and editor files through `.gitignore`.
- The backend loads the root `.env` file automatically when started from the `backend/` directory.
