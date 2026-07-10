# Go Practice: Phased Backend Project

A single project you grow over time. Each phase adds one backend concept and one slice of Go. Finish a phase before moving on — every phase should run, be testable, and be committable.

## The project: **Desklog**

A personal work-log API. You track **projects**, **tasks**, and **time entries** (what you worked on and for how long). It is small enough to finish, rich enough to cover real backend fundamentals, and useful as a portfolio piece.

### Core domain

| Entity      | Purpose                                      |
|-------------|----------------------------------------------|
| Project     | A container (e.g. "go-practice", "client X") |
| Task        | A unit of work inside a project              |
| Time entry  | A logged block of time against a task        |

### API surface (end state)

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

---

## How to use this doc

1. Work phases in order — each builds on the last.
2. **Read the [phase manual](./guides/) for your current phase** — self-contained, concept + instruction in one place.
3. Keep the app runnable after every phase (`go run ./cmd/api`).
4. Commit at the end of each phase with a clear message (e.g. `feat(phase-2): add mongodb repository`).
5. Do not skip tests once Phase 3 introduces them; they are part of the deliverable.
6. Resist adding features from later phases early — scope discipline is part of the training.

### Phase manuals

| Phase | Manual |
|-------|--------|
| 1 | [HTTP & Go basics](./guides/phase-01-http-and-go-basics.md) |
| 2 | [MongoDB & structure](./guides/phase-02-mongodb-and-structure.md) |
| 3 | [Auth, validation & tests](./guides/phase-03-auth-validation-tests.md) |
| 4 | [Time entries & reporting](./guides/phase-04-time-entries-reporting.md) |
| 5 | [Concurrency & resilience](./guides/phase-05-concurrency-resilience.md) |
| 6 | [Ship it](./guides/phase-06-ship-it.md) |

### Suggested weekly rhythm

| Frequency | Activity                                      |
|-----------|-----------------------------------------------|
| 3×/week   | 45–60 min reading (Go + one backend topic)    |
| 3×/week   | 60–90 min coding in this repo                 |
| 1×/week   | Refactor or add tests to the previous phase   |

### Resources (keep handy)

- [A Tour of Go](https://go.dev/tour/)
- [Go by Example](https://gobyexample.com/)
- [net/http](https://pkg.go.dev/net/http)
- [MongoDB Go Driver](https://www.mongodb.com/docs/drivers/go/current/)
- [Effective Go](https://go.dev/doc/effective_go)

---

## Target layout (grow into this by Phase 2)

```
go-practice/
├── cmd/
│   └── api/
│       └── main.go          # wiring, server start
├── internal/
│   ├── handler/             # HTTP layer
│   ├── service/             # business logic
│   ├── repository/          # data access
│   └── model/               # domain types + BSON tags
├── scripts/
│   └── indexes.js           # index definitions (Phase 2+)
├── docker-compose.yml       # Phase 6
├── Dockerfile               # Phase 6
├── go.mod
├── README.md
└── PROJECT.md               # this file
```

---

## Phase 1 — Go basics + HTTP (week 1–2)

**Goal:** A working HTTP server with in-memory storage. Understand request/response, JSON, and status codes.

### Go topics

- Modules (`go mod init`, `go mod tidy`)
- Structs, methods, slices, maps
- `encoding/json`
- `net/http`, `http.HandlerFunc`, `http.ServeMux` (or `chi` if you prefer a router)
- Error handling (`error`, wrapping with `fmt.Errorf`)

### Backend topics

- HTTP methods and idempotency
- Status codes: `200`, `201`, `204`, `400`, `404`, `500`
- JSON request/response contracts
- Handler responsibilities (parse → call logic → respond)

### Deliverables

- [ ] `GET /health` → `{"status":"ok"}`
- [ ] In-memory store for projects and tasks (maps or slices in `main` or a small package)
- [ ] `GET/POST /projects`, `GET /projects/{id}`
- [ ] `GET/POST /projects/{id}/tasks`, `GET /tasks/{id}`
- [ ] JSON errors for bad input and not-found (`{"error":"..."}`)
- [ ] README: how to run (`go run ./cmd/api`)

### Acceptance criteria

- `curl` against every endpoint succeeds with correct status codes
- Invalid JSON returns `400`
- Unknown IDs return `404`
- Server restarts and data resets (in-memory is fine)

### Stretch

- `PATCH` and `DELETE` for projects and tasks
- Simple middleware that logs method + path + duration

---

## Phase 2 — Structure + MongoDB (week 3–5)

**Goal:** Replace in-memory storage with a real database. Split code into layers.

### Go topics

- Package layout (`internal/`)
- Interfaces for repositories (enables testing later)
- Official MongoDB Go driver (`go.mongodb.org/mongo-driver`)
- `context.Context` on all DB calls
- BSON tags on structs (`bson:"_id"`, `bson:"project_id"`)
- `primitive.ObjectID` for document IDs

### Backend topics

- Document modeling (references vs embedding)
- Collections as domain boundaries
- Repository pattern (handler → service → repository)
- Connection pooling and `MONGODB_URI` config
- Indexes for lookups and uniqueness
- Multi-document operations (e.g. delete project + its tasks)

### Collections (starting point)

**`projects`**

```json
{
  "_id": "ObjectId",
  "name": "go-practice",
  "description": "Learning backend in Go",
  "created_at": "2026-07-10T00:00:00Z",
  "updated_at": "2026-07-10T00:00:00Z"
}
```

**`tasks`**

```json
{
  "_id": "ObjectId",
  "project_id": "ObjectId",
  "title": "Implement health endpoint",
  "status": "todo",
  "created_at": "2026-07-10T00:00:00Z",
  "updated_at": "2026-07-10T00:00:00Z"
}
```

`status` values: `todo` | `doing` | `done`

**Indexes (create in code or `scripts/indexes.js`)**

```javascript
db.projects.createIndex({ name: 1 })
db.tasks.createIndex({ project_id: 1 })
db.tasks.createIndex({ project_id: 1, status: 1 })
```

### Deliverables

- [ ] Folder layout: `cmd/api`, `internal/handler|service|repository|model`
- [ ] MongoDB via Docker locally (`docker run` or compose stub)
- [ ] Repository layer using the official driver (`InsertOne`, `Find`, `UpdateOne`, `DeleteMany`)
- [ ] All Phase 1 endpoints backed by MongoDB
- [ ] `PATCH` / `DELETE` for projects and tasks
- [ ] Env config: `PORT`, `MONGODB_URI`, `MONGODB_DATABASE`

### Acceptance criteria

- Data survives server restart
- Deleting a project removes its tasks (`DeleteMany` on `tasks` where `project_id` matches, or a transaction)
- No driver calls in handlers — only in repository
- IDs are stored as `ObjectID`, exposed in JSON as strings

### Stretch

- Seed script or `make seed` for demo data
- Pagination: `GET /projects?limit=&offset=` using `Find().SetSkip().SetLimit()`
- Unique index on `projects.name` and handle duplicate key errors

---

## Phase 3 — Auth, validation, tests (week 6–8)

**Goal:** Multi-user API with tests. Only owners see their data.

### Go topics

- `httptest` for handler tests
- Table-driven tests
- `testing` package, test helpers
- Password hashing: `golang.org/x/crypto/bcrypt`
- JWT or session cookies (pick one and document it)
- `mongo/integration/mtest` or testcontainers for integration tests (optional)

### Backend topics

- Authentication vs authorization
- Storing password hashes, never plaintext
- Request validation and consistent error shape
- Structuring tests: unit (service) + integration (handler + DB)
- Middleware: auth, request ID, logging
- Scoping queries by `user_id` on every read/write

### Collection additions

**`users`**

```json
{
  "_id": "ObjectId",
  "email": "you@example.com",
  "password_hash": "$2a$...",
  "created_at": "2026-07-10T00:00:00Z"
}
```

**`projects`** — add owner reference:

```json
{
  "_id": "ObjectId",
  "user_id": "ObjectId",
  "name": "go-practice",
  "description": "",
  "created_at": "2026-07-10T00:00:00Z",
  "updated_at": "2026-07-10T00:00:00Z"
}
```

**Indexes**

```javascript
db.users.createIndex({ email: 1 }, { unique: true })
db.projects.createIndex({ user_id: 1 })
```

### Deliverables

- [ ] `POST /auth/register`, `POST /auth/login`
- [ ] Protected routes — require auth for all project/task mutations
- [ ] Users only access their own projects (filter by `user_id` in every query)
- [ ] Input validation (email format, non-empty title, allowed task status)
- [ ] Structured logging (`log/slog` or `zerolog`)
- [ ] Tests: at least service layer + one handler integration test per resource

### Acceptance criteria

- Unauthenticated requests to protected routes return `401`
- User A cannot read or modify User B's project (`404` or `403` — pick one, document it)
- Tests pass with `go test ./...`
- Passwords are hashed in DB; duplicate email returns `409` (unique index)

### Stretch

- Refresh tokens or logout denylist (store in a `sessions` or `token_denylist` collection)
- Rate limit on `/auth/login`

---

## Phase 4 — Time entries + reporting (week 9–10)

**Goal:** Complete the domain. Practice queries, filtering, and aggregation.

### Go topics

- `time` package, RFC3339 timestamps
- MongoDB aggregation pipeline (`$match`, `$group`, `$sum`)
- Query parameters and parsing (`from`, `to`, `status`)
- `options.Find()` filters and sorts

### Backend topics

- Designing nested resources (`/tasks/{id}/time-entries`)
- Reporting endpoints vs CRUD
- Pagination and date-range filters
- Idempotency considerations for creates
- Aggregation pipelines for analytics

### Collection

**`time_entries`**

```json
{
  "_id": "ObjectId",
  "task_id": "ObjectId",
  "user_id": "ObjectId",
  "minutes": 45,
  "note": "Wired up handlers",
  "logged_at": "2026-07-10T14:30:00Z",
  "created_at": "2026-07-10T14:30:00Z"
}
```

**Indexes**

```javascript
db.time_entries.createIndex({ task_id: 1, logged_at: -1 })
db.time_entries.createIndex({ user_id: 1, logged_at: -1 })
```

**Example aggregation** (minutes per project in a date range):

```javascript
db.time_entries.aggregate([
  { $match: { user_id: ObjectId("..."), logged_at: { $gte: ISODate("..."), $lte: ISODate("...") } } },
  { $lookup: { from: "tasks", localField: "task_id", foreignField: "_id", as: "task" } },
  { $unwind: "$task" },
  { $group: { _id: "$task.project_id", total_minutes: { $sum: "$minutes" } } }
])
```

### Deliverables

- [ ] CRUD for time entries (scoped to task + user)
- [ ] `GET /reports/summary?from=&to=` — total minutes per project in range
- [ ] Filter tasks by status: `GET /projects/{id}/tasks?status=done`
- [ ] OpenAPI or markdown API doc in README

### Acceptance criteria

- Summary report matches manual sum of entries in range
- Time entry cannot be created for another user's task
- Invalid date range returns `400`
- Report uses an aggregation pipeline (not in-memory summing in the handler)

### Stretch

- Export report as CSV
- Soft-delete tasks (`deleted_at` field + filter in queries) instead of hard delete

---

## Phase 5 — Concurrency + resilience (week 11)

**Goal:** Background work and production-minded behavior without Kubernetes.

### Go topics

- Goroutines, `sync.WaitGroup`, channels
- Worker pool pattern
- `context` cancellation and timeouts
- Graceful shutdown (`signal.Notify`, `server.Shutdown`)

### Backend topics

- Async jobs (e.g. nightly summary email stub, webhook on task completed)
- Timeouts on HTTP handlers and MongoDB operations (`context.WithTimeout`)
- Retries with backoff (for external calls)
- In-memory or Redis cache for hot reads (e.g. report summary)
- Change streams or polling for job processing (introductory)

### Deliverables

- [ ] Graceful shutdown on `SIGINT`/`SIGTERM`
- [ ] Background worker: process a job queue (channel or `jobs` collection)
- [ ] One async job implemented (e.g. `task.completed` → log or fake webhook POST)
- [ ] Request timeout middleware
- [ ] Optional: cache report summary with TTL

### Acceptance criteria

- In-flight requests complete on shutdown
- Worker processes jobs without blocking HTTP handlers
- Document job flow in README

### Stretch

- Outbox pattern: insert job into `jobs` collection in the same MongoDB transaction as the domain update
- TTL index on `jobs` for automatic cleanup of completed work

---

## Phase 6 — Ship it (week 12)

**Goal:** Runnable by someone else. Portfolio-ready.

### Go topics

- Multi-stage `Dockerfile` (build + minimal runtime image)
- Build flags, `CGO_ENABLED=0` for static binary
- `-ldflags` for version injection (optional)

### Backend topics

- 12-factor config (env only, no secrets in image)
- Health checks: liveness vs readiness (`/health` vs `/ready` with MongoDB ping)
- Basic CI: `go test`, `go vet`, `staticcheck` or `golangci-lint`

### Deliverables

- [ ] `Dockerfile` + `docker-compose.yml` (api + mongodb)
- [ ] `GET /ready` checks MongoDB connectivity (`Ping`)
- [ ] GitHub Actions (or similar): test on push
- [ ] README: architecture diagram, env vars, curl examples, design decisions
- [ ] Tag a release: `v0.1.0`

### Acceptance criteria

- `docker compose up` brings up a working API
- Fresh clone + README steps = running app in &lt; 10 minutes
- CI green on main

---

## What to avoid (until the capstone is done)

| Temptation              | Why wait                                              |
|-------------------------|-------------------------------------------------------|
| Microservices           | One monolith teaches boundaries first                 |
| Heavy frameworks        | `net/http` or `chi` + stdlib teaches HTTP             |
| Heavy ODM abstractions  | Official driver + explicit queries teaches MongoDB    |
| Kubernetes              | Docker Compose is enough for this project             |
| GraphQL / gRPC          | Master REST + JSON first                              |
| Perfect hexagonal arch  | Three layers + tests beats abstract diagrams          |

---

## Phase checklist (track your progress)

| Phase | Focus                            | Done |
|-------|----------------------------------|------|
| 1     | HTTP + in-memory CRUD            | [ ]  |
| 2     | MongoDB + layered architecture   | [ ]  |
| 3     | Auth + validation + tests          | [ ]  |
| 4     | Time entries + reporting         | [ ]  |
| 5     | Workers + graceful shutdown      | [ ]  |
| 6     | Docker + CI + docs               | [ ]  |

---

## Getting started today

```bash
cd go-practice
go mod init github.com/<you>/go-practice   # use your module path
mkdir -p cmd/api internal/{handler,service,repository,model} scripts
```

Implement Phase 1 only. When `/health` and one CRUD flow work, commit and mark Phase 1 done in the table above.
