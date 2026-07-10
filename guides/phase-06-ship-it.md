# Phase 6 Manual — Ship It

**Duration:** 1 week  
**Prerequisite:** Phase 5 complete

You built a real backend. Phase 6 is about **delivery** — packaging Desklog so another developer can clone, run, and understand it without your help. This is portfolio and team handoff quality.

---

## 0. What changes in this phase

| Addition | Purpose |
|----------|---------|
| Dockerfile | Reproducible build and run |
| docker-compose | One command: API + MongoDB |
| `/ready` endpoint | Readiness for load balancers |
| CI pipeline | Automated test on every push |
| Portfolio README | Architecture, setup, API examples |
| Release tag | Milestone marker (`v0.1.0`) |

---

## 1. Liveness vs readiness

### Two health checks, two questions

| Endpoint | Question | Should fail when |
|----------|----------|------------------|
| `GET /health` | Is the process running? | Never (if it responds, it's alive) |
| `GET /ready` | Can this instance serve traffic? | MongoDB unreachable |

### Why both exist

**Orchestrators** (Kubernetes, ECS, load balancers) use:

- **Liveness probe** → restart container if deadlocked
- **Readiness probe** → remove from traffic until dependencies ready

Desklog without MongoDB should not receive API traffic even if the Go process is up.

### Implement /ready

```go
func readyHandler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := client.Ping(ctx, nil); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}
```

- MongoDB down → **503 Service Unavailable** (not 500 panic)
- MongoDB up → **200**

`/health` stays simple — no DB check.

---

## 2. Docker fundamentals

### Image vs container

- **Image** — snapshot of filesystem + config (the recipe)
- **Container** — running instance of an image

Your Dockerfile defines how to build the API image.

### Multi-stage build — why

A Go build needs the full compiler (~hundreds of MB). Production only needs the binary + CA certificates (~20 MB).

**Stage 1 (build):** compile  
**Stage 2 (run):** copy binary into minimal image

### Dockerfile

```dockerfile
# --- build ---
FROM golang:1.22-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /desklog ./cmd/api

# --- run ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /desklog .

EXPOSE 8080
ENTRYPOINT ["./desklog"]
```

### Flags explained

| Flag | Meaning |
|------|---------|
| `CGO_ENABLED=0` | Pure Go binary, no C dependencies — runs in minimal Alpine |
| `GOOS=linux` | Target Linux (Docker containers are Linux) |
| `ca-certificates` | HTTPS calls to webhooks need root CAs |

### Build and run

```bash
docker build -t desklog .
docker run --rm -p 8080:8080 \
  -e MONGODB_URI=mongodb://host.docker.internal:27017 \
  -e MONGODB_DATABASE=desklog \
  -e JWT_SECRET=dev-secret-change-me \
  desklog
```

`host.docker.internal` — Docker Desktop alias for your host machine (where MongoDB may run). On Linux Docker, use host network or compose instead.

### Optional: embed version

```dockerfile
RUN CGO_ENABLED=0 go build -ldflags="-X main.version=0.1.0" -o /desklog ./cmd/api
```

In `main.go`:

```go
var version = "dev"
```

Expose in `/health`: `{"status":"ok","version":"0.1.0"}`

---

## 3. docker-compose — full stack

### Why compose

One file defines API + MongoDB + env vars + volumes. `docker compose up` is your demo command.

```yaml
services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      PORT: "8080"
      MONGODB_URI: mongodb://mongodb:27017
      MONGODB_DATABASE: desklog
      JWT_SECRET: change-me-in-production
      WEBHOOK_URL: ""
    depends_on:
      - mongodb

  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - desklog_data:/data/db

volumes:
  desklog_data:
```

### Service names as hostnames

Inside the `api` container, `mongodb://mongodb:27017` works because Docker DNS resolves `mongodb` to the MongoDB container.

### Startup race

`depends_on` waits for container **start**, not MongoDB **ready**. Add retry loop in app:

```go
for i := 0; i < 30; i++ {
	if err := client.Ping(ctx, nil); err == nil {
		break
	}
	time.Sleep(time.Second)
}
```

Or use compose healthcheck on mongodb + `condition: service_healthy`.

### Volume

`desklog_data` persists MongoDB data across `docker compose down` (without `-v`).

---

## 4. Twelve-factor configuration

### Principles for Desklog

| Rule | Implementation |
|------|----------------|
| Config in environment | `PORT`, `MONGODB_URI`, `JWT_SECRET`, `WEBHOOK_URL` |
| No secrets in image | Pass via compose env or `.env` file (gitignored) |
| No secrets in git | Commit `.env.example` only |
| Fail fast | Exit at startup if required var missing |

### .env.example

```
PORT=8080
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=desklog
JWT_SECRET=replace-with-long-random-string
WEBHOOK_URL=
```

### .gitignore

```
.env
```

Developers copy `.env.example` to `.env` locally.

---

## 5. CI pipeline

### Purpose

Run tests automatically on every push. Broken main on a public repo is a red flag for reviewers.

### GitHub Actions — minimum workflow

`.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true

      - name: Vet
        run: go vet ./...

      - name: Test
        run: go test ./...
```

### What each step does

| Step | Catches |
|------|---------|
| `go vet` | Suspicious constructs (unreachable code, printf mistakes) |
| `go test` | Regressions in logic |

### Level up (optional)

- `golangci-lint`
- MongoDB service container for integration tests
- `go build ./...` to verify all packages compile

Push to GitHub and confirm CI passes.

---

## 6. Portfolio README

Your README is part of the deliverable. Structure:

### 1. Title and one-liner

> **Desklog** — Multi-user work-log API in Go and MongoDB.

### 2. Features

- JWT authentication
- Projects, tasks, time entries
- Summary reports with aggregation
- Graceful shutdown, background jobs
- Dockerized

### 3. Architecture diagram

```
┌────────┐     HTTP      ┌───────────┐     ┌─────────┐     ┌──────────┐
│ Client │ ────────────► │ Handlers  │ ──► │ Service │ ──► │ MongoDB  │
└────────┘               └───────────┘     └─────────┘     │ Repos    │
                                 │                │         └──────────┘
                                 │                ▼
                                 │         ┌──────────┐
                                 └────────►│ Worker   │
                                           └──────────┘
```

### 4. Quick start (Docker — primary path)

```bash
git clone <repo>
cd go-practice
docker compose up --build
curl http://localhost:8080/health
```

### 5. Local development (without Docker)

Prerequisites, MongoDB setup, `go run ./cmd/api`

### 6. Environment variables

Table of all vars with required/optional and description.

### 7. API walkthrough

Full curl session: register → login → create project → task → log time → report. Copy-pasteable.

### 8. Project structure

Brief explanation of `cmd/`, `internal/handler`, etc.

### 9. Design decisions

Short honest notes:

- Why JWT
- Why MongoDB references over embedding
- 404 for other users' resources (hide existence)

Interviewers read this section.

### 10. Learning guides

Link to `guides/` for phased curriculum.

---

## 7. Release tag

Mark the milestone:

```bash
git tag -a v0.1.0 -m "Desklog MVP: auth, CRUD, reporting, docker"
git push origin v0.1.0
```

Semantic versioning: `0.1.0` = initial working release, API may still change.

---

## 8. Optional polish

### Makefile

```makefile
.PHONY: run test docker

run:
	go run ./cmd/api

test:
	go test ./...

docker:
	docker compose up --build
```

### .dockerignore

```
.git
.env
*.md
guides/
```

Smaller build context, faster builds.

---

## 9. Fresh-clone test

Simulate a reviewer:

1. Clone to a new directory
2. Follow README only (no asking you questions)
3. `docker compose up --build`
4. Complete API walkthrough with curl
5. Under 10 minutes to working API

If any step fails, fix README or compose — not their problem.

---

## 10. Common mistakes

| Mistake | Impact |
|---------|--------|
| README only shows `go run` | Reviewer without Go/Mongo setup stuck |
| Secrets in compose committed | Security issue |
| Single-stage 800MB image | Looks unprofessional |
| No CI | Broken tests on main |
| Endpoints listed without curl examples | Hard to verify quickly |

---

## 11. Exit checklist — project complete

- [ ] Multi-stage Dockerfile builds
- [ ] `docker compose up` runs API + MongoDB
- [ ] `/health` and `/ready` work
- [ ] `.env.example` committed, `.env` gitignored
- [ ] CI runs vet + test on push
- [ ] README: quick start, env vars, full API walkthrough, architecture
- [ ] Tag `v0.1.0`
- [ ] All phase checklists in PROJECT.md marked done

**Commit:** `feat(phase-6): docker, ci, and production readme`

---

## 12. What comes after

Desklog is a foundation. Natural extensions (not required now):

- **Metrics** — Prometheus `/metrics` endpoint
- **Rate limiting** — protect `/auth/login`
- **Email** — real SendGrid from worker
- **Frontend** — React app consuming your API
- **OpenAPI** — machine-readable spec for clients

Depth on one service teaches more than jumping to microservices. Ship one more feature on Desklog before splitting anything.

---

Congratulations — you built and shipped a Go backend.

**Back to index:** [guides/README.md](./README.md)
