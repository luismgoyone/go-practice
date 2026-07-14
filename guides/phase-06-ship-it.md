# Phase 6 Manual — Ship It

**Duration:** 1 week  
**Prerequisite:** Phase 5 complete  
**Audience:** Beginners — Part A + Part B activities (same framing as Phases 1–5)

**How to use**

1. Read Part A (why `/ready`, Docker, CI, README).
2. Do Part B Activities in order.
3. `/ready` still follows the [endpoint recipe](./the-endpoint-recipe.md).
4. End with the [final quiz](./final-quiz.md).

---

# Part A — Concepts (read first)

## A0. What you ship

| Addition | Purpose |
|----------|---------|
| `GET /ready` | Ready for traffic only if Mongo is up |
| Dockerfile | Reproducible Linux binary image |
| docker-compose | One command: API + Mongo |
| `.env.example` | Config without secrets in git |
| CI | `go vet` + `go test` on push |
| Portfolio README | Clone → run without asking you |
| Tag `v0.1.0` | Milestone |

| Endpoint | Question |
|----------|----------|
| `GET /health` | Is the process alive? (always cheap) |
| `GET /ready` | Can we serve? (ping Mongo) → `503` if not |

---

## A1. Multi-stage Docker

**Build stage:** compile with `golang` image.  
**Run stage:** copy binary into tiny `alpine` (+ CA certs for HTTPS webhooks).

`CGO_ENABLED=0` → static-ish pure Go binary for Alpine.

---

## A2. Compose networking

Inside Compose, hostname `mongodb` resolves to the Mongo service.  
API env: `MONGODB_URI=mongodb://mongodb:27017`.

`depends_on` ≠ “Mongo ready” — retry ping at startup.

---

# Part B — Build steps

---

## Step 1 — `GET /ready` (recipe practice)

### Contract

- `GET /ready`  
- Mongo OK → `200` `{"status":"ready"}`  
- Mongo down → `503` `{"error":"database unavailable"}`  

### File: add to `internal/handler/health.go` (or `ready.go`)

```go
func ReadyHandler(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := client.Ping(ctx, nil); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}
```

**Note:** handlers usually avoid `mongo` imports — for this learning phase, either pass a small `Pinger` interface or accept the client here. Prefer:

```go
type Pinger interface{ Ping(ctx context.Context) error }
```

### Activity 1.1 — Wire public route (no auth)

```go
http.HandleFunc("GET /ready", handler.ReadyHandler(...))
```

### Activity 1.2

```bash
curl -i http://localhost:8080/ready   # 200
docker stop desklog-mongo             # or stop compose mongo
curl -i http://localhost:8080/ready   # 503
docker start desklog-mongo
```

**Gate:** 200 then 503 then 200 again.

---

## Step 2 — Dockerfile

### File: `Dockerfile` (full file)

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /desklog ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /desklog .
EXPOSE 8080
ENTRYPOINT ["./desklog"]
```

### File: `.dockerignore`

```
.git
.env
guides/
*.md
```

### Activity 2.1

```bash
docker build -t desklog .
```

Expect build success.

---

## Step 3 — docker-compose

### File: `docker-compose.yml` (full file)

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

### Activity 3.1 — Startup retry in `main` (if needed)

Loop Ping up to ~30s so API waits for Mongo on first boot.

### Activity 3.2

```bash
docker compose up --build
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

**Gate:** both return 200.

---

## Step 4 — Twelve-factor config files

### File: `.env.example`

```
PORT=8080
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=desklog
JWT_SECRET=replace-with-long-random-string
WEBHOOK_URL=
```

### Activity 4.1

Ensure `.env` is in `.gitignore`. Commit `.env.example` only.  
App already fatals on missing required secrets — keep that.

---

## Step 5 — CI

### File: `.github/workflows/ci.yml` (full file)

```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:

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

### Activity 5.1

Push to GitHub; confirm the workflow is green.

---

## Step 6 — Portfolio README

### Activity 6.1 — Rewrite/expand root `README.md` to include:

1. One-liner what Desklog is  
2. Features list  
3. Architecture (handler → service → repo → Mongo + worker)  
4. Quick start: `docker compose up --build`  
5. Env var table  
6. Full curl walkthrough (register → … → report)  
7. Link to `guides/`  
8. Short design decisions (JWT, 404 for others’ resources, etc.)

### Activity 6.2 — Fresh-clone test

1. Clone into a **new** folder  
2. Follow README only  
3. Under ~10 minutes: health OK + one authenticated create  

Fix anything that fails.

---

## Step 7 — Release tag

### Activity 7.1

```bash
git tag -a v0.1.0 -m "Desklog MVP: auth, CRUD, reporting, docker"
git push origin v0.1.0
```

(Only when you are ready and code is committed.)

---

## Step 8 — Capstone quiz

### Activity 8.1

Complete **[Final quiz — ship a new contract](./final-quiz.md)** (project notes).  
Fill the recipe card **before** coding.

---

## Common mistakes

| Mistake | Fix |
|---------|-----|
| README only shows `go run` | Lead with Compose |
| Secrets in git | `.env` gitignored |
| Single huge image | Multi-stage build |
| No CI | Add workflow |

---

## Exit checklist — project complete

- [ ] `/ready` works (503 when Mongo down)  
- [ ] Multi-stage Dockerfile  
- [ ] `docker compose up --build` runs stack  
- [ ] `.env.example` committed  
- [ ] CI green  
- [ ] README: quick start + curl walkthrough  
- [ ] Tag `v0.1.0`  
- [ ] Final quiz attempted  

**Commit:** `feat(phase-6): docker, ci, and production readme`

---

Congratulations — you built and shipped a Go backend.

**Always:** [The endpoint recipe](./the-endpoint-recipe.md)  
**Quiz:** [final-quiz.md](./final-quiz.md)  
**Index:** [guides/README.md](./README.md)
