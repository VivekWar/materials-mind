<div align="center">

# Materials Mind

**AI-powered materials engineering assistant — a hybrid Retrieval-Augmented Generation system that grounds Google Gemini's recommendations in a real PostgreSQL + `pgvector` materials database, with automated hallucination guardrails.**

[![CI](https://github.com/VivekWar/materials-mind/actions/workflows/ci.yml/badge.svg)](https://github.com/VivekWar/materials-mind/actions/workflows/ci.yml)
[![Deploy Backend](https://github.com/VivekWar/materials-mind/actions/workflows/deploy-backend.yml/badge.svg)](https://github.com/VivekWar/materials-mind/actions/workflows/deploy-backend.yml)
[![Deploy Frontend](https://github.com/VivekWar/materials-mind/actions/workflows/deploy-frontend.yml/badge.svg)](https://github.com/VivekWar/materials-mind/actions/workflows/deploy-frontend.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-pgvector-4169E1?logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-cache-DC382D?logo=redis&logoColor=white)
![Gemini](https://img.shields.io/badge/Google_Gemini-API-8E75B2?logo=googlegemini&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)

[Architecture](#architecture-overview) · [RAG Pipeline](#system-design--the-rag-pipeline) · [API Docs](#api-documentation) · [Testing](#testing) · [Setup](#local-development-setup)

</div>

---

## At a Glance

- **Hybrid retrieval, not a vector-search wrapper.** `pgvector` cosine similarity and LLM-extracted SQL constraint filtering run **concurrently via goroutines**, get merged and deduplicated, then **reranked by domain relevance** before ever reaching the LLM.
- **Hallucination guardrails, not blind trust.** Every recommendation is schema-validated, and every cited material ID is cross-checked against what was actually retrieved. A mismatch triggers an automatic repair-prompt retry instead of silently returning unverified output.
- **Production concerns handled, not ignored.** Graceful shutdown on `SIGTERM`, cost-control quotas enforced consistently across every LLM-backed route, a CI test gate on every pull request, and Redis-cached responses to cut redundant API spend.

## Table of Contents

- [Overview](#overview)
- [Problem Statement](#problem-statement)
- [Key Features](#key-features)
- [Architecture Overview](#architecture-overview)
- [System Design — The RAG Pipeline](#system-design--the-rag-pipeline)
- [Technology Stack](#technology-stack)
- [Folder Structure](#folder-structure)
- [Database Design](#database-design)
- [API Documentation](#api-documentation)
- [Testing](#testing)
- [Authentication & Authorization](#authentication--authorization)
- [Environment Variables](#environment-variables)
- [Local Development Setup](#local-development-setup)
- [Development Workflow & CI](#development-workflow--ci)
- [Deployment](#deployment)
- [Monitoring & Logging](#monitoring--logging)
- [Security Considerations](#security-considerations)
- [Performance Optimizations](#performance-optimizations)
- [Known Trade-offs & Next Steps](#known-trade-offs--next-steps)
- [Contributing](#contributing)
- [License](#license)

## Overview

Materials Mind is a technical intelligence console designed to translate complex engineering constraints into deterministic, physics-grounded material recommendations. It acts as an AI-powered materials scientist, leveraging a hybrid Retrieval-Augmented Generation (RAG) architecture to query an extensive database of material properties and provide actionable insights for hardware and mechanical engineering applications.

## Problem Statement

Selecting the optimal material for a specific engineering application (e.g., aerospace, automotive, high-temperature manufacturing) traditionally requires manual cross-referencing across disconnected datasheets, supplier catalogs, and academic papers. Engineers must balance multiple constraints (yield strength, thermal conductivity, density, cost) simultaneously. Materials Mind automates this process by combining the reasoning capabilities of Large Language Models (LLMs) with deterministically indexed vector data, ensuring that material recommendations are not hallucinated but are explicitly grounded in accurate, real-world data.

## Key Features

* **Hybrid RAG Semantic Search:** Combines `pgvector` similarity matching with strict SQL constraint filtering (e.g., enforcing `yield_strength >= X` or `max_density <= Y`), run **concurrently** via goroutines so total latency is bounded by the slower path, not the sum of both.
* **Domain-Aware Reranking:** When a user specifies an industry domain (Aerospace, Automotive, etc.), retrieved candidates are stably reordered so category-matching materials lead the LLM's context window.
* **Hallucination Guardrails:** Every LLM recommendation is schema-validated and cross-checked so cited material IDs must exist in the actual retrieved candidate set; a mismatch triggers an automatic repair-prompt retry instead of silently returning bad data.
* **Engineering Intelligence Chat:** Persistent chat workspaces that maintain context for complex, multi-turn material engineering discussions.
* **Deterministic Property Serialization:** Complete exposure of 28+ physical, thermal, and electrical material properties directly to the UI, handling missing values and zero states explicitly without `omitempty` issues.
* **Popup OAuth Flow:** A frictionless, cross-domain authentication mechanism using `window.postMessage`, avoiding the pitfalls of third-party cookie blocking and messy URL tokens.
* **Interactive Chat Console:** A high-fidelity, Titanium/Geist-themed UI built for professional engineering standards.

## Architecture Overview

Materials Mind operates on a decoupled client-server architecture. The frontend is a React Single Page Application (SPA) communicating via a RESTful API to a Golang backend. The backend acts as the orchestrator, integrating with a PostgreSQL vector database for hybrid queries, Redis for caching, and Google Gemini for LLM reasoning.

```mermaid
graph TD
    Client[Browser Client - CF Pages]
    Backend[Go Backend - HF Space]
    Neon[(Neon PostgreSQL + pgvector)]
    Redis[(Upstash Redis)]
    Gemini[Google Gemini API]
    OAuth[Google OAuth]

    Client -- REST API / Bearer Token --> Backend
    Client -- Popup Login --> OAuth
    OAuth -- OAuth Callback --> Backend
    Backend -- Vector/SQL Query --> Neon
    Backend -- Context Caching --> Redis
    Backend -- Augmented Prompts --> Gemini
```

## System Design — The RAG Pipeline

When an engineer submits a query, `SearchService.ProcessSearch` (`backend/services/search_service.go`) runs:

1. **Cache check:** the normalized query is hashed (SHA-256) and checked against Redis first — an identical query never re-hits Gemini.
2. **Parallel retrieval:** two goroutines run at the same time:
   - **Vector path** — the query is embedded via `gemini-embedding-001` and matched against `materials.embedding` using pgvector's HNSW index (`ORDER BY embedding <=> $1`).
   - **Intent + keyword path** — a separate Gemini call extracts structured constraints (min yield strength, max density, operating temperature) into JSON, which become parameterized SQL `WHERE` clauses.
3. **Merge & rerank:** results from both paths are deduplicated by material ID, then stably reordered so materials whose category matches the requested industry domain are considered first.
4. **Augmentation:** the top candidates are serialized into a structured prompt context — the LLM is explicitly told not to invent properties.
5. **Generation:** Gemini returns structured JSON (recommendation, trade-offs, confidence score, cited source IDs, natural-language report).
6. **Validation:** the response is schema-checked and every cited source ID must exist in the retrieved candidate set; on failure, a repair prompt is sent and the generation is retried (bounded attempts) before the request fails loudly rather than returning an unverified answer.
7. **Stream:** the validated result streams to the frontend over Server-Sent Events, then is cached in Redis (30 min TTL) and persisted to Postgres.

## Technology Stack

| Layer | Choices |
|---|---|
| Frontend | React 18, Vite, TypeScript, Zustand (state), Tailwind CSS, Lucide Icons |
| Backend | Go 1.25+, Gin (HTTP), pgx/v5 (Postgres driver), Goth (OAuth) |
| Database | Neon Serverless PostgreSQL + `pgvector` (HNSW index) |
| Cache / Rate Limiting | Upstash Redis |
| AI Provider | Google Gemini API (generation + `gemini-embedding-001` embeddings) |
| Testing | Go `testing` (unit) + Postman collection (manual/API-level) |
| CI/CD | GitHub Actions — test gate on PRs, auto-deploy on merge to `main` |
| Deployment | Cloudflare Pages (frontend), Hugging Face Spaces / Docker (backend) |

<details>
<summary><strong>Folder Structure</strong> (click to expand)</summary>

```text
/
├── backend/                  # Golang API Server
│   ├── db/                   # Database & Redis initialization
│   ├── domain/                # Core data models (Chat, Material, etc.)
│   ├── handlers/              # HTTP route handlers (Gin)
│   ├── middleware/            # CORS, Rate Limiting, JWT Auth
│   ├── repositories/           # Data access layer (SQL/pgx)
│   ├── services/               # Business logic (Gemini, Hybrid RAG) + unit tests
│   ├── utils/                  # JWT, backoff, OAuth init + unit tests
│   ├── Dockerfile             # Container configuration
│   └── main.go                # Application entrypoint & router
├── frontend/                 # React SPA
│   ├── public/                # Static assets & _redirects (CF routing)
│   ├── src/
│   │   ├── api/                # Axios/fetch client and SSE parsing
│   │   ├── components/         # Reusable UI elements (Hero, Panel)
│   │   ├── pages/               # View components (Auth, Chat, Home)
│   │   ├── store/                # Zustand global state
│   │   └── styles/               # Tailwind configuration & global CSS
│   ├── package.json            # Node dependencies
│   └── vite.config.ts          # Bundler configuration
├── data/                     # Utilities for embedding generation/DB seeding
├── postman/                  # Postman collection + environment for manual API testing
├── .github/workflows/        # CI (test gate) + CD (deploy) pipelines
└── docker-compose.yml         # Local development infrastructure
```

</details>

## Database Design

The schema uses a relational model optimized for high-dimensional vector search and flexible property storage.

```mermaid
erDiagram
    users {
        bigint id PK
        string email
        string full_name
        string provider
    }
    chats {
        bigint id PK
        bigint user_id FK
        string title
        boolean is_active
        timestamp created_at
    }
    messages {
        bigint id PK
        bigint chat_id FK
        string sender_role
        jsonb content
        int tokens_used
    }
    materials {
        bigint id PK
        string name
        string category
        jsonb specific_properties
        vector embedding
    }
    
    users ||--o{ chats : "creates"
    chats ||--o{ messages : "contains"
```

## API Documentation

The API follows RESTful principles, secured via Bearer JWTs. A ready-to-import **Postman collection** covering every route below lives in [`postman/`](postman/) — see [Testing](#testing) for how to use it.

**Daily usage caps** (cost control on the Gemini-backed paths): 10 new chats/day and 30 messages/day per user, enforced server-side on both `/api/search` and `/api/chat/followup` and returned via `GET /api/auth/me`.

<details>
<summary><strong>Full route list</strong> (click to expand)</summary>

### Public Routes
* `GET /health` - Liveness probe.
* `GET /healthz` - Deep infrastructure readiness probe (tests DB and Redis connectivity).
* `GET /api/auth/:provider/login` - Initiates Google OAuth.
* `GET /api/auth/:provider/callback` - Completes OAuth, returns popup `postMessage` script.

### Protected Routes (Requires `Authorization: Bearer <token>`)
* `GET /api/auth/me` - Validates session and returns current user context (including daily usage counts).
* `POST /api/auth/logout` - Terminates session.
* `POST /api/search` - Executes hybrid RAG against the material DB. Returns an SSE stream.
* `POST /api/chat/followup` - Processes multi-turn conversation context.
* `POST /api/chat/create` - Initializes a new persistent chat session.
* `GET /api/chat/list` - Retrieves all active chats for the authenticated user.
* `GET /api/chat/:chat_id/messages` - Retrieves the message history for a specific chat.
* `POST /api/chat/:chat_id/messages` - Appends a new message to the chat.
* `POST /api/chat/:chat_id/archive` - Soft-deletes a chat session.
* `POST /api/chat/:chat_id/title/generate` - Generates and persists a short chat title from the first message.

</details>

## Testing

Testing is split into two layers, matched to what's actually practical to automate without spinning up Postgres/Redis/a live Gemini key:

**1. Automated unit tests (`go test ./...`, run on every PR via CI)**
Cover pure business logic that doesn't touch external services:
- `backend/utils/jwt_test.go` — token generation/verification roundtrip, rejects tampered or wrongly-signed tokens.
- `backend/utils/backoff_test.go` — retry/give-up/context-cancellation behavior of the exponential backoff helper used for all Gemini calls.
- `backend/services/gemini_test.go` — embedding input normalization and truncation.
- `backend/services/search_service_test.go` — structured-recommendation JSON parsing, source-citation validation, and domain-aware reranking.

Run locally:
```bash
cd backend
go vet ./...
go test ./... -v
```

**2. Manual/API-level testing (Postman)**
The repository ships a Postman collection (`postman/MaterialsMind.postman_collection.json`) and environment (`postman/MaterialsMind.postman_environment.json`) covering every route in `main.go`, organized into Health / Auth / Search & RAG / Chat Management folders. Import both into Postman, then:
1. Point the `base_url` environment variable at your running backend (defaults to `http://localhost:8080`).
2. Sign in through the actual frontend once, copy the `auth_token` value out of browser LocalStorage, and paste it into the `auth_token` environment variable (there's no password-grant endpoint to script around Google OAuth).
3. Run requests individually, or use Postman's Collection Runner for a manual smoke pass across all endpoints.

**What's not covered yet:** repository-layer and handler-layer tests (they need a real Postgres instance — a `testcontainers`-based integration suite is the natural next step) and end-to-end browser tests of the OAuth + chat flow. See [Known Trade-offs & Next Steps](#known-trade-offs--next-steps).

## Authentication & Authorization

The application uses a secure, cross-domain optimized OAuth 2.0 implementation.
1. The user clicks "Sign In", opening a popup window directed to `/api/auth/google/login`.
2. Google authenticates the user and redirects to the backend callback.
3. The backend issues a JWT and returns an HTML snippet to the popup window.
4. The popup script executes `window.opener.postMessage({ token: "..." }, "FRONTEND_ORIGIN")`.
5. The frontend SPA listens for the message, extracts the JWT, saves it to `localStorage`, and closes the popup.
6. Subsequent requests include the JWT in the `Authorization: Bearer` header.

*Decision Context:* A popup-based `postMessage` flow was explicitly chosen over a standard URL redirect (`?token=...`) to prevent tokens from leaking into browser history/referrers and to bypass aggressive cross-domain third-party cookie blocking in modern browsers (Safari, Chrome Incognito).

<details>
<summary><strong>Environment Variables</strong> (click to expand)</summary>

### Backend (`.env`)
```bash
# Core Services
PORT=8080
GIN_MODE=release

# Database
DATABASE_URL=postgres://user:password@neon.tech/dbname

# Redis
REDIS_URL=redis://default:password@upstash.io:6379

# AI Integrations
GEMINI_API_KEY=your_gemini_key

# Authentication
JWT_SECRET=super_secure_random_string
GOOGLE_CLIENT_ID=your_google_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your_google_secret
BACKEND_URL=https://api.materials-mind.com
FRONTEND_ORIGIN=https://materials-mind.pages.dev
```

### Frontend (`.env`)
```bash
VITE_API_URL=https://api.materials-mind.com/api
VITE_GOOGLE_CLIENT_ID=your_google_id.apps.googleusercontent.com
```

</details>

## Local Development Setup

1. **Clone the repository.**
2. **Start Infrastructure:**
   ```bash
   docker-compose up -d postgres redis
   ```
3. **Run Backend:**
   ```bash
   cd backend
   go run main.go
   ```
4. **Run Frontend:**
   ```bash
   cd frontend
   npm install
   npm run dev
   ```
5. **(Optional) Import the Postman collection** from `postman/` to exercise the API without the UI — see [Testing](#testing).

## Development Workflow & CI

Features should be developed in feature branches and merged via Pull Requests. Every PR against `main` triggers `.github/workflows/ci.yml`, which:
- Runs `go vet`, `go build`, and `go test ./...` for the backend.
- Runs `tsc --noEmit` and `vite build` for the frontend.

Nothing merges silently untested — this is a hard gate, separate from the deploy workflows below.

## Deployment

Deployment is fully automated via GitHub Actions (`.github/workflows/deploy-frontend.yml` and `deploy-backend.yml`), triggered on push to `main`.
* **Frontend:** Pushed directly to Cloudflare Pages via Wrangler. The `_redirects` file ensures the SPA router correctly handles paths like `/chat`.
* **Backend:** Pushed to Hugging Face Spaces via a Docker deployment. Large binaries are strictly excluded via `.gitignore` and `.dockerignore`.

## Monitoring & Logging

* The application provides a deep `/healthz` endpoint returning HTTP 503 if the database or cache connections degrade.
* The backend logs critical application panics, failed database constraints, and rate-limiting triggers natively via Gin's logging middleware.
* The server performs a graceful shutdown on `SIGTERM`/`SIGINT` (15s drain window), so container rotations on Hugging Face Spaces or Kubernetes don't kill in-flight Gemini streams mid-response.

## Security Considerations

* **CORS:** Strictly bound to `FRONTEND_ORIGIN` in production; wildcard/localhost/preview origins are only accepted outside of `GIN_MODE=release`.
* **Rate Limiting:** Per-user (or per-IP, pre-auth) rate limiting via Upstash Redis protects the expensive LLM execution paths from abuse or scraping.
* **Daily Quotas:** Message and chat-creation caps are enforced consistently on both the initial search and every follow-up turn, so a single day's Gemini spend per user is bounded end-to-end.
* **Data Sanitization:** The frontend implements robust token removal on logout to guarantee complete state clearance; Markdown responses are rendered through `rehype-sanitize`.
* **SQL Injection:** The `pgxpool` driver exclusively uses parameterized statements and strict type casting for all user inputs.
* **Secrets:** No credentials are committed — `.env` is git-ignored everywhere, and `.env.hf.example` / `frontend/.env.production` only ever contain placeholders or public (non-secret) client IDs.

## Performance Optimizations

* **Parallel Retrieval:** Vector search, intent extraction, and keyword search run concurrently via goroutines rather than sequentially.
* **Hybrid Execution:** By allowing SQL `WHERE` constraints to pre-filter rows before vector similarity is executed, query latencies on large datasets are significantly reduced.
* **Redis Caching:** Minimizes redundant LLM API calls (SHA-256-keyed, 30 min TTL) and stabilizes latency for repeated queries.
* **Vite Bundling:** Frontend assets are optimized, tree-shaken, and minified during the CI build process.

## Known Trade-offs & Next Steps

Honest list of what's intentionally not solved yet, and why:

* **Single LLM provider.** All generation and embedding calls go through Gemini with no fallback. An outage or hard rate-limit currently fails the request rather than failing over to a second provider. Worth doing behind an `LLMProvider` interface, but deliberately not added speculatively without a second provider to actually test against.
* **No retrieval-quality evaluation harness.** There's no `precision@k`/labeled query set to measure whether the vector+keyword retrieval is actually surfacing the *right* materials, only that it returns *some* validated materials. This is the natural next step for anyone extending the ML side of this project.
* **No integration/E2E tests.** Unit tests cover pure logic (see [Testing](#testing)); the repository/handler layers (need a live Postgres) and the OAuth + chat UI flow (needs a browser) aren't automated yet.
* **No OpenAPI/Swagger spec.** The Postman collection and the API docs above are the source of truth for now; a generated spec would be the next step if the API surface grows.
* **No CONTRIBUTING.md** documenting the local DB seeding process (`data/seed_db.py`) in detail beyond the inline script comments.

## Contributing

This started as a personal/portfolio project, but issues and pull requests are welcome — particularly around the retrieval-quality evaluation harness or LLM-provider fallback work described above. Please open an issue before starting a large change.

## License

Released under the [MIT License](LICENSE) — free to use, fork, or adapt for your own materials-selection tooling.

---

<div align="center">

Built by [VivekWar](https://github.com/VivekWar)

</div>
