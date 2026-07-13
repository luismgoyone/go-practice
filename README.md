# Desklog (go-practice)

A phased learning project: build a real **work-log API** in Go and MongoDB while learning backend fundamentals — HTTP, persistence, auth, testing, concurrency, and deployment.

You grow one codebase across six phases. Each phase adds one layer of skill. Finish a phase before moving on.

## What you're building

**Desklog** tracks:

| Entity | Purpose |
|--------|---------|
| Project | Container for work (e.g. "go-practice", "client X") |
| Task | Unit of work inside a project |
| Time entry | Logged minutes against a task |

**End-state API** (built incrementally):

```
GET    /health
POST   /auth/register
POST   /auth/login

GET    /projects
POST   /projects
GET    /projects/{id}
PATCH  /projects/{id}
DELETE /projects/{id}

GET    /projects/{id}/tasks
POST   /projects/{id}/tasks
GET    /tasks/{id}
PATCH  /tasks/{id}
DELETE /tasks/{id}

GET    /tasks/{id}/time-entries
POST   /tasks/{id}/time-entries
DELETE /time-entries/{id}

GET    /reports/summary?from=&to=
```

## Prerequisites

- Go 1.22+ ([install](https://go.dev/doc/install))
- Terminal and `curl` (or Postman)
- Docker (Phase 2 onward, for MongoDB)
- Git

## How to use this repo

| Document | Purpose |
|----------|---------|
| [PROJECT.md](./PROJECT.md) | Roadmap — phases, deliverables, acceptance criteria |
| [guides/](./guides/) | **Phase manuals** — self-contained lessons (start here) |
| [The endpoint recipe](./guides/the-endpoint-recipe.md) | Same 7 steps for every new endpoint |

**Workflow:**

1. Skim [the endpoint recipe](./guides/the-endpoint-recipe.md).
2. Open the manual for your current phase.
3. Build in recipe order (Phase 1 has full copy-paste steps).
4. Pass the phase exit checklist and commit.
5. After Phase 6, take the [final quiz](./guides/final-quiz.md).

### Phase manuals

| Phase | Manual | Focus |
|-------|--------|-------|
| — | [Endpoint recipe](./guides/the-endpoint-recipe.md) | How to start any new endpoint |
| 1 | [HTTP & Go basics](./guides/phase-01-http-and-go-basics.md) | In-memory API (beginner steps) |
| 2 | [MongoDB & structure](./guides/phase-02-mongodb-and-structure.md) | Persistence + layers |
| 3 | [Auth, validation & tests](./guides/phase-03-auth-validation-tests.md) | Users, JWT, tests |
| 4 | [Time entries & reporting](./guides/phase-04-time-entries-reporting.md) | Aggregations |
| 5 | [Concurrency & resilience](./guides/phase-05-concurrency-resilience.md) | Workers, shutdown |
| 6 | [Ship it](./guides/phase-06-ship-it.md) | Docker, CI, portfolio |
| Capstone | [Final quiz](./guides/final-quiz.md) | Implement a new contract solo |

### Progress

| Phase | Done |
|-------|------|
| 1 — HTTP + in-memory CRUD | [ ] |
| 2 — MongoDB + layered architecture | [ ] |
| 3 — Auth + validation + tests | [ ] |
| 4 — Time entries + reporting | [ ] |
| 5 — Workers + graceful shutdown | [ ] |
| 6 — Docker + CI + docs | [ ] |
| Capstone — final quiz (project notes) | [ ] |

## Getting started (Phase 1)

```bash
git clone <your-repo-url>
cd go-practice

# module already initialized as github.com/luismgoyone/go-practice
mkdir -p cmd/api internal/model internal/store

go run ./cmd/api
```

Then follow [Phase 1 Manual](./guides/phase-01-http-and-go-basics.md) from section 0.

Quick smoke test once `/health` exists:

```bash
curl -i http://localhost:8080/health
```

Expected: `HTTP/1.1 200 OK` and `{"status":"ok"}`.

## Project layout

Grows as you advance. Phase 1 starts minimal; Phase 2+ fills in `internal/`.

```
go-practice/
├── cmd/
│   └── api/
│       └── main.go              # entry point
├── internal/
│   ├── model/                   # domain types
│   ├── store/                   # Phase 1: in-memory
│   ├── handler/                 # Phase 2+: HTTP layer
│   ├── service/                 # Phase 2+: business logic
│   └── repository/              # Phase 2+: MongoDB
├── guides/                      # phase manuals
├── scripts/                     # Phase 2+: MongoDB indexes
├── PROJECT.md                   # phased roadmap
├── go.mod
└── README.md
```

## Running the API (by phase)

| Phase | Command | Notes |
|-------|---------|-------|
| 1 | `go run ./cmd/api` | In-memory; data resets on restart |
| 2+ | `go run ./cmd/api` | Requires MongoDB (`MONGODB_URI`) |
| 6 | `docker compose up --build` | API + MongoDB together |

Environment variables are introduced in Phase 2. Phase 6 adds `.env.example` and Docker Compose.

## Suggested pace

| Frequency | Activity |
|-----------|----------|
| 3×/week | 45–60 min reading the manual |
| 3×/week | 60–90 min coding |
| 1×/week | Refactor or tests from the previous phase |

Rough timeline: 8–12 weeks part-time.

## License

Personal learning project — add a license if you open-source it.
