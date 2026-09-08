# Pollify — Project Reference

A polling and voting service with anonymous polls, multi-choice and free-text answers, and quorum-based community moderation. Backend in Go, frontend in React/TypeScript, persistence in PostgreSQL 17, packaged as Docker containers.

This document is the long-form reference. For the 60-second pitch see `.kilo/plans/TLDR.md`. For deployment instructions see `DEPLOY.md`. For the original implementation plan see `.kilo/plans/1776941384248-brave-moon.md`.

---

## 1. Goals & guiding principles

The service exists to gather opinions and run votes. The plan picks a small set of principles that drive every design choice:

| Principle | What it means in practice |
| --- | --- |
| **Separation of participation and choice** | The fact "user X took part in poll Y" is stored in one table; the actual votes are stored in another. This makes anonymity expressible at the schema level, not just at the API. |
| **Universal vote model** | A single `votes` table represents option picks **and** free-text answers — the application chooses which column to populate. Multi-choice is just multiple rows in `votes` with the same `(poll_id, user_id)`. |
| **Configurable polls** | Anonymity, multi-choice (with `max_choices`), and free-text are flags on `polls`. Multi-choice and free-text are **mutually exclusive** in v1 to keep results aggregation tractable. |
| **Limited moderation** | Admins cannot edit content directly. They only act on user reports, and only as part of a quorum. |
| **Quorum moderation** | A single admin cannot resolve a report. Two matching decisions (default) close it; the second admin reads the first decision but cannot self-review. |
| **Trust guarantees that survive admin access** | Anonymous votes cannot be re-linked to users via the API. Cast votes are immutable. |

---

## 2. Architecture

```
┌──────────────────┐  HTTP / JSON   ┌──────────────────┐  pgx/v5   ┌──────────────────┐
│ Frontend SPA     │  Bearer JWT    │ Backend API      │  pool      │ PostgreSQL 17    │
│ React + TS       │ ─────────────► │ Go monolith      │ ─────────► │ 7 tables         │
│ react-router-dom │                │ chi + pgx + jwt  │            │ 3 triggers       │
│ Vite (dev)       │                │ hexagonal layout │            │ 9 indexes        │
└──────────────────┘                └──────────────────┘            └──────────────────┘
        │                                   │
        ▼                                   ▼
   served by Caddy in prod            backend/migrations/
   (reverse-proxy /api/*)             one .up.sql / one .down.sql
```

### Backend layout

The backend is a single Go binary, structured by **business component**, not by technical layer. Each component lives directly under `internal/`:

```
backend/
├── cmd/api/main.go               – entry point; calls app.Run
├── internal/
│   ├── users/        core/{models,ports,errors,service}
│   │                 adapters/{postgres, http}
│   ├── polls/        core/{models,ports,errors,service,validators}
│   │                 adapters/{postgres, http}
│   ├── voting/       core/{models,ports,errors,service,policies}
│   │                 adapters/{postgres, http}
│   ├── moderation/   core/{models,ports,errors,service,quorum}
│   │                 adapters/{postgres, http}
│   ├── platform/     auth/         (JWT issuer, HMAC verifier, role guard)
│   │                 httpserver/   (HTTP lifecycle wrapper)
│   │                 postgres/     (pgx pool + migrations)
│   │                 app/          (router composition)
│   │                 config/, clock/, logging/, openapi/, postgrestest/
│   └── pkg/apierror/             – unified ErrorResponse envelope
├── migrations/                    – numbered SQL migrations
└── tests/integration/api_test.go – end-to-end smoke against real DB
```

Each component's `core/` package defines pure-domain types: models, ports (interfaces), services (use cases), errors, validators or policies. Adapters depend on `core/` interfaces — they are inverted. The platform layer wires everything.

### Frontend layout

```
frontend/
├── src/
│   ├── api/              – TypeScript types mirroring the OpenAPI + fetch wrapper
│   ├── auth/             – AuthContext (token in localStorage, /me hydration)
│   ├── pages/            – Login, Register, PollList, CreatePoll,
│   │                       PollDetail, PollResults, ModerationQueue, ReportDetail
│   ├── App.tsx           – routes + RequireAuth / RequireAdmin guards
│   ├── main.tsx          – BrowserRouter bootstrap
│   └── index.css         – design system (custom properties, status pills, etc.)
├── tests/e2e/            – Playwright specs (auth, poll-flow, moderation)
├── playwright.config.ts
├── Dockerfile            – multi-stage: node build → caddy serve
├── Caddyfile             – reverse-proxies /api + SPA fallback
└── vite.config.ts        – dev proxy /api → 127.0.0.1:8080
```

---

## 3. Domain model & database schema

### 3.1 Tables

| Table | Purpose | Notable columns / constraints |
| --- | --- | --- |
| `users` | Identity + role | `email UNIQUE`, `role IN ('USER','ADMIN')` |
| `polls` | Poll metadata + settings | Schedule, anonymity, multi-choice, custom-answer flags. Three CHECKs (see below). |
| `options` | Predefined choices for option-based polls | `poll_id` FK with `ON DELETE CASCADE` |
| `poll_participants` | Records that a user took part in a poll | **PK = (poll_id, user_id)** — the uniqueness invariant for "1 user, 1 vote" |
| `votes` | The actual vote(s) | `option_id` **XOR** `custom_text`; `user_id` may be NULL |
| `reports` | User-filed moderation reports | `status` is a Postgres enum |
| `report_reviews` | Admin decisions on a report | **PK = (report_id, admin_id)** — one review per admin per report |

### 3.2 Enum types

```sql
CREATE TYPE report_status   AS ENUM ('OPEN', 'IN_REVIEW', 'RESOLVED', 'REJECTED');
CREATE TYPE review_decision AS ENUM ('APPROVE', 'REJECT');
```

These are real Postgres enum types — domain types at the storage layer.

### 3.3 CHECK constraints

```sql
-- polls
CHECK (end_at > start_at)
CHECK (NOT is_multiple_choice OR max_choices > 1)
CHECK (NOT (is_multiple_choice AND allow_custom_answer))

-- votes (mutual exclusion of payload kinds)
CHECK (
    (option_id IS NOT NULL AND custom_text IS NULL) OR
    (option_id IS NULL     AND custom_text IS NOT NULL)
)

-- users
CHECK (role IN ('USER', 'ADMIN'))
```

### 3.4 Triggers

Three plpgsql triggers act as a second line of defence — they fire even on direct DB access:

| Trigger | What it enforces |
| --- | --- |
| `report_reviews_admin_only` | The actor on every INSERT/UPDATE has `role = 'ADMIN'`. |
| `report_reviews_no_self_review` | Rejects a review row where `admin_id = reports.created_by` for that report. |
| `votes_option_matches_poll` | A non-null `votes.option_id` must point to an `options` row with the same `poll_id`. |

### 3.5 Indexes

```sql
polls(created_by)
polls(is_hidden, start_at, end_at)        -- list-page filters
options(poll_id)
votes(poll_id), votes(user_id)
reports(poll_id), reports(created_by), reports(status)
```

### 3.6 Anonymity by structure

This is the headline design choice and worth a detailed look.

```
poll_participants                  votes
─────────────────                  ─────
poll_id, user_id, voted_at         id, poll_id, option_id|custom_text, user_id?, created_at
PK (poll_id, user_id)              user_id is NULLABLE
```

For **non-anonymous polls** the application sets `votes.user_id = the actor`. For **anonymous polls** the application leaves `votes.user_id = NULL`. In both cases `poll_participants` records that the user participated, so:

- The "1 user = 1 vote" invariant is enforced by the composite PK.
- For anonymous polls, no row in `votes` is linked to a user. The mapping does not exist anywhere.
- The API for anonymous polls never returns `voter_details`, never accepts `include_voters=true`. The voting service refuses such requests at the domain boundary.

This decouples uniqueness (PK on `poll_participants`) from authorship (`votes.user_id`) — anonymity is a structural property of the schema, not a runtime guard.

---

## 4. Authentication & authorization

### 4.1 Tokens

- `POST /api/v1/auth/register` and `POST /api/v1/auth/login` issue an **access token** (HMAC-SHA256 JWT) plus an opaque refresh token.
- Access token TTL: 1 hour. Refresh tokens are placeholders in v1 — no refresh endpoint yet.
- Tokens carry `sub` (user id), `role`, `iat`, `nbf`, `exp`.
- The frontend stores both in `localStorage` and sends `Authorization: Bearer <access_token>` on every protected request.

### 4.2 Roles

Two roles only:

- **`USER`** (default) — every account starts here. `POST /auth/register` hardcodes `role = USER`. There is no API path that escalates a user to ADMIN.
- **`ADMIN`** — granted out-of-band by whoever owns the database (a SQL `UPDATE users SET role='ADMIN' WHERE email='…'`). After promotion the user must sign out and back in to mint a fresh JWT carrying the new role.

The decision to keep admin promotion strictly out-of-band is intentional. Per the plan: admins are operators, not a self-serve tier. There is no way to elevate yourself or anyone else through the API.

### 4.3 Middleware

- `auth.Middleware(verifier)` — validates the bearer token and injects `users.AuthenticatedUser` into the request context.
- `auth.RequireRole(role)` — returns a middleware that 403s if the actor's role doesn't match. Applied to the `/reports/*` routes and the GET-by-poll listing.

---

## 5. API surface

All paths under `/api/v1`. Full schema in `api/openapi.yaml`. Public endpoints are explicitly opted out of the bearer requirement.

### 5.1 Auth & identity

| Method | Path | Auth | Behaviour |
| --- | --- | --- | --- |
| `POST` | `/auth/register` | none | Email + password (≥8 chars) → 201 with user + tokens. 409 if email taken. |
| `POST` | `/auth/login` | none | Email + password → 200 with user + tokens. 401 on bad creds. |
| `GET`  | `/users/me` | USER | Current user as `UserSummary`. |

### 5.2 Polls

| Method | Path | Auth | Behaviour |
| --- | --- | --- | --- |
| `GET` | `/polls/` | USER | Paginated list with filters (`status`, `creator_id`, `is_anonymous`, `is_multiple_choice`, `allow_custom_answer`, `available_for_voting`, `page`, `limit`, `sort`). |
| `POST` | `/polls/` | USER | Creates a poll. Domain rejects multi-choice + custom-answer combo, end-before-start, etc. |
| `GET` | `/polls/{id}` | USER | Single poll with options + computed status. |
| `PATCH` | `/polls/{id}` | USER (creator) | Title / description / start_at / end_at — only before `start_at` and before any participants exist. Otherwise 409. |

### 5.3 Voting

| Method | Path | Auth | Behaviour |
| --- | --- | --- | --- |
| `POST` | `/polls/{id}/votes` | USER | One vote per (poll, user). Body is `oneOf` `{option_ids: [...]}` or `{custom_text: "..."}`. 409 if already voted or poll not active. |
| `GET` | `/polls/{id}/results` | USER | Aggregated counts. `?include_voters=true` works only on non-anonymous polls (422 otherwise). |

### 5.4 Reports & moderation

| Method | Path | Auth | Behaviour |
| --- | --- | --- | --- |
| `POST` | `/polls/{id}/reports` | USER | One active report per (user, poll). 409 if duplicate. |
| `GET` | `/polls/{id}/reports` | ADMIN | Reports for a poll. |
| `GET` | `/reports/` | ADMIN | All reports, paginated, filterable by `status`. |
| `GET` | `/reports/{id}` | ADMIN | Single report with all reviews. |
| `POST` | `/reports/{id}/reviews` | ADMIN | Submit `APPROVE` / `REJECT`. Forbidden on own report (403). Triggers the quorum logic. |

### 5.5 Errors

Every error response uses the same envelope:

```json
{
  "error": {
    "code": "poll_already_voted",
    "message": "user has already participated in this poll",
    "details": {}
  }
}
```

Codes: `validation_error`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `poll_not_active`, `poll_hidden`, `poll_already_voted`, `poll_vote_payload_invalid`, `poll_update_not_allowed`, `report_already_exists`, `report_review_self_forbidden`, `report_review_already_exists`, `report_already_resolved`, `quorum_not_reached`, `internal_error`.

---

## 6. Quorum moderation in detail

The quorum logic lives in `backend/internal/moderation/core/quorum.go`. It is a pure function over the reviews list:

```
EvaluateQuorum(reviews, quorum) -> {
    approval_count, rejection_count,
    reached: bool,
    final_decision,
    status: report_status,
    resolution: string
}
```

States and transitions:

```
        OPEN
          │  first review arrives
          ▼
        IN_REVIEW
          │
          ├─ approval_count ≥ quorum  ──► RESOLVED  (poll hidden)
          └─ rejection_count ≥ quorum ──► REJECTED  (poll stays visible)
```

The default quorum is **2** (configurable in `app.Run` via `moderation.DefaultQuorum`). Once a quorum is reached the report is considered closed; further review attempts return `409 report_already_resolved`.

The decision to hide the poll (when an APPROVE quorum is reached) is taken by the moderation service synchronously — it calls `polls.PollVisibilityWriter.Hide(pollID)` and only then persists the review. The whole sequence is driven by application code; the database guards the parts it can (admin role, no self-review, one-review-per-admin) via triggers.

---

## 7. Tech stack

| Layer | Choice | Why |
| --- | --- | --- |
| Backend language | Go 1.25 | Static binary, small footprint, strong stdlib HTTP. |
| HTTP routing | `github.com/go-chi/chi/v5` | Idiomatic with stdlib, easy middleware composition. |
| DB driver | `github.com/jackc/pgx/v5` | Native Postgres protocol, pool, prepared statements. |
| Auth | `github.com/golang-jwt/jwt/v5` | Standard JWT lib. |
| Password hashing | `golang.org/x/crypto/bcrypt` | Adaptive cost factor, well-vetted. |
| Frontend framework | React 19 + TypeScript | Industry default; types help on the API surface. |
| Frontend tooling | Vite 8 | Fast dev server with native ESM, simple proxy config. |
| Routing (FE) | `react-router-dom` v7 | BrowserRouter for clean URLs. |
| Database | PostgreSQL 17 | Strong constraints, triggers, enums. |
| Reverse proxy | Caddy 2 | Single binary, automatic HTTPS, simple Caddyfile. |
| Containers | Docker + docker compose v2 | Single-host deploy story. |
| E2E tests | Playwright | Real-browser testing of the SPA. |

---

## 8. Testing strategy

Five layers of tests, all green:

| Layer | What it covers | Where |
| --- | --- | --- |
| Domain unit | Validators and policies (vote payload rules, poll-create rules, quorum). | `internal/*/core/*_test.go` (in components that have them) |
| HTTP adapter unit | Request/response shape, error mapping, role guard with stub services. | `internal/*/adapters/http/handlers_test.go` |
| Postgres repository | Real Postgres via the `postgrestest` helper (per-test schema). | `internal/*/adapters/postgres/repository_test.go` |
| Backend integration | The full chi router against a real DB: register → vote → report → quorum → poll hidden. | `backend/tests/integration/api_test.go` |
| Frontend E2E | Playwright drives Chromium against the running prod stack. 14 specs cover auth, poll lifecycle, multi-choice / anonymous / free-text variants, moderation flows. | `frontend/tests/e2e/*.spec.ts` |

Run them all:

```bash
# backend (postgrestest needs DATABASE_URL)
cd backend && DATABASE_URL='postgres://pollify:pollify@127.0.0.1:5432/pollify?sslmode=disable' \
  go test ./...

# frontend E2E (full stack must be up)
cd frontend && npm run e2e
```

---

## 9. Local development

### 9.1 Two stacks

| Stack | Compose file | Use case |
| --- | --- | --- |
| Backend dev | `docker-compose.yml` | API + Postgres on host ports 8080 / 5432; pair with `npm run dev` in `frontend/` for Vite HMR. |
| Local prod | `docker-compose.prod.yml` | API + Postgres + Caddy serving the built SPA on port 80. Closest to a real deployment. |

### 9.2 Make targets

```
make help            # list everything
make up              # local prod stack + seed admin1..3@gmail.com / admin123
make down            # stop, keep DB
make reset           # wipe DB volume + reseed admins
make seed-admins     # (re)create the three admin accounts via the live API
make logs            # tail all services
make psql            # psql shell into the DB container
make dev-up / dev-down  # backend dev stack only
```

`make up` creates a `.env` with throwaway development defaults if missing, polls `/healthz` until the API is ready, then registers and promotes three admin accounts so the moderation flow is demoable in seconds.

### 9.3 Seeded admin accounts

| Email | Password | Role |
| --- | --- | --- |
| admin1@gmail.com | admin123 | ADMIN |
| admin2@gmail.com | admin123 | ADMIN |
| admin3@gmail.com | admin123 | ADMIN |

Three are seeded so a single demo session can show: report filed by an unprivileged user, admin1 approves (IN_REVIEW), admin2 approves (RESOLVED + poll hidden), admin3 left over to demonstrate that further reviews on a resolved report are rejected.

---

## 10. Deployment

Single-host deployment is documented in `DEPLOY.md`. Summary:

1. Linux VM with Docker + Docker Compose.
2. `git clone` the repo; `cp .env.example .env`; fill in `POSTGRES_PASSWORD` (any strong value) and `JWT_SECRET` (`openssl rand -hex 32`).
3. `docker compose -f docker-compose.prod.yml up -d --build`.
4. Open `http://<server-ip>`.
5. Promote your first admin via `docker compose ... exec postgres psql -U pollify -d pollify -c "UPDATE users SET role='ADMIN' WHERE email='you@example.com';"`.

Adding TLS is one env-var change (`SITE_ADDRESS=your.domain`) plus exposing port 443 — Caddy fetches and renews a Let's Encrypt cert automatically.

---

## 11. What's intentionally out of scope

For v1, the following are **not** built. Each is a deliberate choice from the plan, not a forgotten item.

- **Admin-promotion API** — admins seeded out-of-band only.
- **Self-service password reset** — no email integration.
- **Refresh-token rotation** — refresh tokens are issued but no refresh endpoint exists.
- **Direct admin actions on polls** — no hide / unhide / delete endpoints. All takedowns flow through reports + quorum.
- **Mixed-mode polls** — `options + custom_text` in the same poll is forbidden by CHECK constraint.
- **Poll deletion** — no DELETE endpoint. A resolved-APPROVE report sets `is_hidden = TRUE` instead.
- **Vote / report / review editing or deletion** — every cast vote and review is final.
- **Real-time push** — clients refetch results; no WebSocket or SSE channel.
- **Per-poll visibility (private polls)** — every poll is visible to every authenticated user.
- **Analytics, time-series, exports** — out of scope.

---

## 12. File map

```
pollify/
├── api/openapi.yaml              # contract source-of-truth
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/{users,polls,voting,moderation,platform,pkg}/...
│   ├── migrations/000001_init.up.sql
│   ├── tests/integration/api_test.go
│   ├── go.mod, go.sum
│   └── Dockerfile
├── frontend/
│   ├── src/{api,auth,pages}, App.tsx, main.tsx, index.css
│   ├── tests/e2e/{auth,poll-flow,moderation}.spec.ts
│   ├── playwright.config.ts
│   ├── Dockerfile, Caddyfile, vite.config.ts
│   └── package.json
├── docs/
│   ├── PROJECT.md                # ← this file
│   ├── Pollify-Database-Talk.pptx
│   ├── system_analytics/         # original requirements docs
│   └── diagrams/                 # ER + use-case diagrams
├── docker-compose.yml            # dev (backend + postgres)
├── docker-compose.prod.yml       # prod (caddy + api + postgres)
├── Makefile                      # local-prod helpers
├── DEPLOY.md                     # single-host deployment guide
├── README.md                     # one-paragraph intro
└── core.md                       # original russian-language principles
```

---

## 13. Glossary

| Term | Meaning |
| --- | --- |
| **Quorum** | The number of admin reviews of the same decision required to close a report. Default: 2. |
| **Anonymous poll** | A poll where `is_anonymous = true`; `votes.user_id` is left NULL by the application; voter details are never exposed by the API. |
| **Participation** | A row in `poll_participants` recording that a user took part. The "one user = one vote" guarantee is enforced here, regardless of how many vote rows the participant created. |
| **Active / scheduled / completed / hidden** | Computed poll status, derived from `start_at`, `end_at`, `is_hidden` and the current time. Not stored. |
| **Self-review** | An admin reviewing a report they themselves filed. Forbidden by both application code and a database trigger. |
