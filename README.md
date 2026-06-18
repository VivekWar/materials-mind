# Materials Mind

## Overview
Materials Mind is a technical intelligence console designed to translate complex engineering constraints into deterministic, physics-grounded material recommendations. It acts as an AI-powered materials scientist, leveraging a hybrid Retrieval-Augmented Generation (RAG) architecture to query an extensive database of material properties and provide actionable insights for hardware and mechanical engineering applications.

## Problem Statement
Selecting the optimal material for a specific engineering application (e.g., aerospace, automotive, high-temperature manufacturing) traditionally requires manual cross-referencing across disconnected datasheets, supplier catalogs, and academic papers. Engineers must balance multiple constraints (yield strength, thermal conductivity, density, cost) simultaneously. Materials Mind automates this process by combining the reasoning capabilities of Large Language Models (LLMs) with deterministically indexed vector data, ensuring that material recommendations are not hallucinated but are explicitly grounded in accurate, real-world data.

## Key Features
* **Hybrid RAG Semantic Search:** Combines pgvector similarity matching with strict SQL constraint filtering (e.g., enforcing `yield_strength >= X` or `max_density <= Y`).
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

## System Design
The system is built around a Hybrid RAG pipeline. When an engineer submits a query:
1. **Intent Extraction:** The query is parsed to identify hard constraints (e.g., "high temperature", "density < 5").
2. **Retrieval:** The backend queries Neon PostgreSQL using both `pgvector` similarity (`ORDER BY embedding <=> $1`) and SQL `WHERE` clauses for hard boundaries.
3. **Augmentation:** The retrieved `MaterialCandidate` records are serialized into a highly structured JSON context.
4. **Generation:** Google Gemini synthesizes an engineering rationale explicitly citing the provided data, returning both a natural language summary and structured JSON (risk flags, confidence, key findings).

## Technology Stack
* **Frontend:** React 18, Vite, TypeScript, Zustand (State), Tailwind CSS, Lucide Icons.
* **Backend:** Golang 1.22+, Gin (HTTP Framework), pgx/v5 (PostgreSQL Driver), Goth (OAuth).
* **Database:** Neon Serverless PostgreSQL with `pgvector` extension.
* **Cache/Rate Limiting:** Upstash Redis.
* **AI Provider:** Google Gemini API (`gemini-3-flash`).
* **Deployment:** Cloudflare Pages (Frontend), Hugging Face Spaces / Docker (Backend), GitHub Actions (CI/CD).

## Folder Structure
```text
/
├── backend/                  # Golang API Server
│   ├── db/                   # Database & Redis initialization
│   ├── domain/               # Core data models (Chat, Material, etc.)
│   ├── handlers/             # HTTP route handlers (Gin)
│   ├── middleware/           # CORS, Rate Limiting, JWT Auth
│   ├── repositories/         # Data access layer (SQL/pgx)
│   ├── services/             # Business logic (Gemini, Hybrid RAG)
│   ├── utils/                # JWT generation, token utilities
│   ├── Dockerfile            # Container configuration
│   └── main.go               # Application entrypoint & router
├── frontend/                 # React SPA
│   ├── public/               # Static assets & _redirects (CF routing)
│   ├── src/
│   │   ├── api/              # Axios client and data fetching
│   │   ├── components/       # Reusable UI elements (Hero, Panel)
│   │   ├── pages/            # View components (Auth, Chat, Home)
│   │   ├── store/            # Zustand global state
│   │   └── styles/           # Tailwind configuration & global CSS
│   ├── package.json          # Node dependencies
│   └── vite.config.ts        # Bundler configuration
├── data/                     # Utilities for embedding generation/DB seeding
├── .github/workflows/        # Automated CI/CD pipelines
└── docker-compose.yml        # Local development infrastructure
```

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
The API follows RESTful principles, secured via Bearer JWTs.

### Public Routes
* `GET /health` - Liveness probe.
* `GET /healthz` - Deep infrastructure readiness probe (tests DB and Redis connectivity).
* `GET /api/auth/:provider/login` - Initiates Google OAuth.
* `GET /api/auth/:provider/callback` - Completes OAuth, returns Popup `postMessage` script.

### Protected Routes (Requires `Authorization: Bearer <token>`)
* `GET /api/auth/me` - Validates session and returns current user context.
* `POST /api/auth/logout` - Terminates session.
* `POST /api/search` - Executes hybrid RAG against the material DB.
* `POST /api/chat/followup` - Processes multi-turn conversation context.
* `POST /api/chat/create` - Initializes a new persistent chat session.
* `GET /api/chat/list` - Retrieves all active chats for the authenticated user.
* `GET /api/chat/:chat_id/messages` - Retrieves the message history for a specific chat.
* `POST /api/chat/:chat_id/messages` - Appends a new message to the chat.
* `POST /api/chat/:chat_id/archive` - Soft-deletes a chat session.

## Authentication & Authorization
The application uses a secure, cross-domain optimized OAuth 2.0 implementation.
1. The user clicks "Sign In", opening a popup window directed to `/api/auth/google/login`.
2. Google authenticates the user and redirects to the backend callback.
3. The backend issues a JWT and returns an HTML snippet to the popup window.
4. The popup script executes `window.opener.postMessage({ token: "..." }, "FRONTEND_ORIGIN")`.
5. The frontend SPA listens for the message, extracts the JWT, saves it to `localStorage`, and closes the popup.
6. Subsequent requests include the JWT in the `Authorization: Bearer` header.

*Decision Context:* A popup-based `postMessage` flow was explicitly chosen over a standard URL redirect (`?token=...`) to prevent tokens from leaking into browser history/referers and to bypass aggressive cross-domain third-party cookie blocking in modern browsers (Safari, Chrome Incognito).

## Environment Variables

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
```

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

## Development Workflow
All commits to the `main` branch trigger automated GitHub Actions pipelines. Features should be developed in feature branches and merged via Pull Requests. The backend requires running `go mod tidy` and testing before pushing to prevent CI failures.

## Deployment
Deployment is fully automated via GitHub Actions (`.github/workflows/deploy-frontend.yml` and `deploy-backend.yml`).
* **Frontend:** Pushed directly to Cloudflare Pages via Wrangler. The `_redirects` file ensures the SPA router correctly handles paths like `/chat`.
* **Backend:** Pushed to Hugging Face Spaces via a Docker deployment. Large binaries are strictly excluded via `.gitignore` and `.dockerignore`.

## Monitoring & Logging
* The application provides a deep `/healthz` endpoint returning HTTP 503 if the database or cache connections degrade.
* The backend logs critical application panics, failed database constraints, and rate-limiting triggers natively via Gin's logging middleware.

## Security Considerations
* **CORS:** Strictly bound to `FRONTEND_ORIGIN`, explicitly rejecting wildcard configurations to prevent CSRF.
* **Rate Limiting:** IP-based rate limiting via Upstash Redis protects the expensive LLM execution paths from abuse or scraping.
* **Data Sanitization:** The frontend implements robust token removal on logout to guarantee complete state clearance.
* **SQL Injection:** The `pgxpool` driver exclusively uses parameterized statements and strict type casting for all user inputs.

## Performance Optimizations
* **Hybrid Execution:** By allowing SQL `WHERE` constraints to pre-filter rows before vector similarity is executed, query latencies on large datasets are significantly reduced.
* **Redis Caching:** Minimizes redundant LLM API calls and stabilizes latency.
* **Vite Bundling:** Frontend assets are optimized, tree-shaken, and minified during the CI build process.

## Documentation Gaps
* Missing comprehensive OpenAPI/Swagger definitions for the REST API.
* Missing a definitive `CONTRIBUTING.md` outlining the local DB seeding process (`data/seed_db.py`).
* Lacking explicit documentation on how to rotate the `JWT_SECRET` gracefully across active sessions.

---

## Engineering Review

### Issues Found

**1. Lack of Automated Test Coverage**
* **Severity:** High
* **Issue:** The repository currently lacks an automated test suite (unit/integration tests) for both the Go backend and React frontend.
* **Recommendation:** Implement `go test` for critical business logic (JWT validation, RAG pipeline construction) and Cypress/Playwright for end-to-end testing of the OAuth flow and chat workspace.

**2. Missing Graceful Shutdown on Backend**
* **Severity:** Medium
* **Issue:** The `main.go` uses `r.Run(":" + port)` which does not gracefully handle `SIGTERM` signals. Deployments on Hugging Face or Kubernetes may instantly kill active LLM streams or database transactions during container rotation.
* **Recommendation:** Implement an `http.Server` with `Shutdown(ctx)` listening to `os.Interrupt` and `syscall.SIGTERM`.

**3. Single Provider Fallback Risk**
* **Severity:** Medium
* **Issue:** The backend logic heavily couples business logic directly to `services.InitGemini()`. If the Gemini API experiences an outage, the entire core product fails.
* **Recommendation:** Implement an interface abstraction over the LLM generation (e.g., `LLMProvider`) and implement a multi-tier fallback engine (OpenRouter, Anthropic, etc.) to handle rate-limiting or service outages automatically.
