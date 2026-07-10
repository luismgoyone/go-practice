# Phase 1 Manual — HTTP & Go Basics

**Duration:** 1–2 weeks  
**You need:** Go 1.22+, a terminal, `curl`

This document is your manual for Phase 1. Everything you need to understand and build is here — concepts first, then instructions with context. Work top to bottom. Do not skip sections.

---

## 0. What you are building

**Desklog** is a work-log API. In Phase 1 it tracks **projects** and **tasks** in memory (RAM). When the server stops, data disappears. That is intentional.

By the end of this phase you will have:

- A Go program that listens on a port
- HTTP endpoints that accept and return JSON
- CRUD operations for projects and tasks
- Correct status codes and error responses

This is a real backend shape. Phases 2–6 add persistence, users, reporting, and deployment on top of the same structure.

---

## 1. How a backend API works

A **backend API** is a program that waits for **HTTP requests** over the network and sends back **HTTP responses**.

A request has:

| Part | Example | Meaning |
|------|---------|---------|
| Method | `GET`, `POST`, `PATCH`, `DELETE` | What action the client wants |
| Path | `/projects/abc123` | Which resource |
| Headers | `Content-Type: application/json` | Metadata about the request |
| Body | `{"name":"go-practice"}` | Payload (optional; common on POST/PATCH) |

A response has:

| Part | Example | Meaning |
|------|---------|---------|
| Status code | `200`, `201`, `404` | Outcome in one number |
| Headers | `Content-Type: application/json` | Metadata about the response |
| Body | `{"id":"abc123","name":"go-practice"}` | Payload (optional) |

**Your job as a backend developer:** receive the request, do something (read/write data, validate input), return a response. In Go, the function that does this is called a **handler**.

---

## 2. Go modules — your project's identity

Go code is organized into **packages**. Related packages live in a **module**, defined by a `go.mod` file at the project root.

The module path (e.g. `github.com/you/go-practice`) is:

- The name you use to import your own packages later
- What appears in `go.mod` and import statements

**Instruction:** From your project root, run:

```bash
go mod init github.com/<you>/go-practice
```

Replace `<you>` with your GitHub username or any unique path. This creates `go.mod`. Every time you add an import and build, run `go mod tidy` to sync dependencies.

**Folder convention:** Put the runnable program in `cmd/api/`. The `cmd/` folder is a Go community convention meaning "things you execute." `internal/` (Phase 2) means "packages only this module can import."

```bash
mkdir -p cmd/api
```

---

## 3. Go types you will use

### Structs — named bundles of fields

A **struct** groups related data. Desklog's core entities are structs:

```go
type Project struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

`time.Time` is Go's built-in type for timestamps. `string` is text. Each field has a type. Fields starting with a capital letter are **exported** (visible outside the package).

### Slices — ordered lists

```go
projects := []Project{}           // empty slice
projects = append(projects, p)    // add one
```

Use slices for "list all projects."

### Maps — lookup by key

```go
projects := map[string]Project{}  // key = ID string
projects[p.ID] = p                // store
p, ok := projects[id]             // lookup; ok is false if missing
```

Use maps for "get project by ID." Keys must be comparable; strings work well for IDs in Phase 1.

### Pointers — when you need them (brief)

Phase 1 rarely needs pointers. You will see `*http.Request` and `http.ResponseWriter` in handlers — interfaces/types passed by reference so the handler can write the response. You do not need to master pointers yet; know they exist.

### Errors — Go has no exceptions

Functions that can fail return an `error` as the last return value:

```go
result, err := doSomething()
if err != nil {
	// handle failure — return HTTP 500, log, etc.
	return
}
// use result
```

**Idiom:** always check `err != nil` immediately. Ignoring errors is the most common beginner bug.

---

## 4. JSON — the language of APIs

Clients and servers exchange **JSON** (JavaScript Object Notation). It looks like:

```json
{"name": "go-practice", "description": "Learning Go"}
```

Go structs map to JSON with **struct tags** — strings after fields:

```go
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
```

| Tag | Effect |
|-----|--------|
| `json:"name"` | JSON field name is `name` |
| `json:"description,omitempty"` | Omit field if empty in output |
| `json:"created_at"` | Snake_case in JSON is conventional for APIs |

**Encoding** (struct → JSON bytes):

```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(project)
```

**Decoding** (JSON body → struct):

```go
var req struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// invalid JSON → 400 Bad Request
	return
}
defer r.Body.Close()
```

Always set `Content-Type: application/json` before writing JSON. Clients use this to parse correctly.

---

## 5. HTTP methods and status codes

### Methods — what the client wants to do

| Method | Meaning | Idempotent? | Typical use |
|--------|---------|-------------|-------------|
| GET | Read | Yes | Fetch one or list |
| POST | Create | No | Create new resource |
| PATCH | Partial update | No* | Change some fields |
| DELETE | Remove | Yes | Delete resource |

*PATCH is often treated as non-idempotent in practice.

**REST-ish design** maps resources to paths and methods to actions:

- `GET /projects` — list
- `POST /projects` — create
- `GET /projects/{id}` — get one
- `PATCH /projects/{id}` — update
- `DELETE /projects/{id}` — delete

Nested: `GET /projects/{id}/tasks` — tasks belonging to a project.

### Status codes — the outcome in one number

| Code | Name | When to use |
|------|------|-------------|
| 200 | OK | Successful GET, PATCH |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE (no body) |
| 400 | Bad Request | Invalid JSON, missing required field |
| 404 | Not Found | ID does not exist |
| 500 | Internal Server Error | Unexpected bug — log details, return generic message |

**Rule:** never return `200` with `{"error":"..."}` in the body. Use the correct status code. Clients, caches, and monitors depend on it.

---

## 6. `net/http` — Go's HTTP server

Go's standard library includes a production-capable HTTP server. No framework required for Phase 1.

### Minimal server

```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

**What happens:**

1. `HandleFunc` registers a route: method + path → handler function
2. `ListenAndServe` binds port 8080 and blocks forever, accepting connections
3. Each request runs the matching handler in its **own goroutine** (lightweight thread)

Go 1.22+ supports `"GET /health"` pattern on the default `ServeMux`. Older Go: use `chi` router or check `r.Method` manually inside the handler.

### Handler signature

```go
func handler(w http.ResponseWriter, r *http.Request)
```

- `w` — write status, headers, body here
- `r` — read method, path, headers, body from here

**Order matters when writing a response:**

1. Set headers (`w.Header().Set(...)`)
2. Call `w.WriteHeader(status)` — only if not 200 (200 is default on first write)
3. Write body (`Encode`, `Fprint`, etc.)

If you call `WriteHeader` twice or write body before headers, behavior is undefined.

### Path parameters

For `/projects/{id}`, extract `id` from the path. With Go 1.22+ `ServeMux`:

```go
id := r.PathValue("id")
```

With nested routes like `/projects/{id}/tasks`, register the pattern explicitly on `HandleFunc`.

---

## 7. The handler pattern — parse, call, respond

Every handler should follow the same steps. This keeps HTTP concerns separate from data logic (and makes Phase 2 easier).

```
1. Parse   — path params, query params, JSON body
2. Validate — required fields, allowed values
3. Execute — call store/service
4. Respond — JSON + correct status code
5. On error — consistent error JSON
```

### Error response helper

Use one shape everywhere:

```json
{"error": "project not found"}
```

```go
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
```

### Example: POST /projects

**Context:** Client sends a name (and optional description). Server creates a project, assigns an ID and timestamps, stores it, returns 201 with the full object.

```go
func createProjectHandler(store *MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}

		now := time.Now().UTC()
		project := Project{
			ID:          uuid.New().String(), // or simple ID generator
			Name:        req.Name,
			Description: req.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		store.CreateProject(project)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(project)
	}
}
```

**Why UTC for timestamps:** servers run in different timezones. Store and return UTC; clients convert for display.

---

## 8. In-memory storage

**Context:** Before MongoDB (Phase 2), data lives in Go data structures inside the running process. This lets you focus on HTTP and domain logic without database setup.

### MemoryStore design

```go
type MemoryStore struct {
	mu       sync.Mutex
	projects map[string]Project
	tasks    map[string]Task
}
```

**Why `sync.Mutex`:** Go's HTTP server handles each request concurrently. Two requests updating the same map at once can cause a **race condition** (crash or corrupt data). Lock before read/write:

```go
func (s *MemoryStore) CreateProject(p Project) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ID] = p
}
```

`defer s.mu.Unlock()` runs when the function returns — even on error paths.

### Methods to implement

| Method | Behavior |
|--------|----------|
| `ListProjects()` | Return all projects (as slice) |
| `GetProject(id)` | Return project + `true`, or zero value + `false` |
| `CreateProject(p)` | Store and return |
| `UpdateProject(p)` | Replace if exists |
| `DeleteProject(id)` | Remove project and all its tasks |
| `ListTasksByProject(projectID)` | Filter tasks by `project_id` |
| `GetTask(id)` | Return task + found bool |
| `CreateTask(t)` | Store and return |
| `UpdateTask(t)` | Replace if exists |
| `DeleteTask(id)` | Remove task |

**Instruction:** Create `internal/model/` for `Project` and `Task` structs. Create the store in `internal/store/memory.go` or temporarily in `main.go` — but separate files are good practice now.

### Task model

```go
type Task struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // todo | doing | done
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

When creating a task via `POST /projects/{id}/tasks`:

1. Verify project exists → else 404
2. Set `ProjectID` from path (not from request body — client could lie)
3. Default `Status` to `"todo"` if empty
4. Validate status is one of `todo`, `doing`, `done` if provided

---

## 9. Endpoints to build

Build in this order. Each builds on the previous.

### 9.1 `GET /health`

**Purpose:** Load balancers and orchestrators ping this to see if the process is alive. No auth, no DB — always fast.

Response: `200` + `{"status":"ok"}`

### 9.2 `GET /projects`

**Purpose:** List all projects.

Response: `200` + JSON array `[{...}, {...}]`

Empty list: `200` + `[]` (not 404)

### 9.3 `POST /projects`

**Purpose:** Create a project.

Request body: `{"name":"...", "description":"..."}`  
Response: `201` + created project including `id`  
Errors: `400` for bad JSON or missing name

### 9.4 `GET /projects/{id}`

**Purpose:** Get one project.

Response: `200` + project  
Errors: `404` if ID not found

### 9.5 `GET /projects/{id}/tasks`

**Purpose:** List tasks for a project.

First verify project exists → `404` if not.  
Response: `200` + array of tasks

### 9.6 `POST /projects/{id}/tasks`

**Purpose:** Create a task under a project.

Request body: `{"title":"...", "status":"todo"}`  
Response: `201` + task  
Errors: `400` validation, `404` project missing

### 9.7 `GET /tasks/{id}`

**Purpose:** Get one task by ID (without needing project in path).

Response: `200` or `404`

### Stretch (recommended before Phase 2)

| Endpoint | Notes |
|----------|-------|
| `PATCH /projects/{id}` | Update name/description; bump `updated_at` |
| `DELETE /projects/{id}` | Remove project + its tasks; `204` or `200` |
| `PATCH /tasks/{id}` | Update title/status |
| `DELETE /tasks/{id}` | Remove task |

---

## 10. Middleware — wrapping handlers

**Context:** Some behavior applies to many routes: logging, auth (Phase 3), timeouts (Phase 5). **Middleware** wraps a handler to run code before and after.

```go
func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
}

// usage
http.HandleFunc("GET /health", withLogging(healthHandler))
```

The inner handler stays unchanged. Middleware is how you add cross-cutting concerns without duplicating code.

---

## 11. Wiring `main.go`

**Context:** `main` is the composition root — it creates the store, registers routes, starts the server.

```go
func main() {
	store := NewMemoryStore()

	http.HandleFunc("GET /health", healthHandler)
	http.HandleFunc("GET /projects", listProjectsHandler(store))
	http.HandleFunc("POST /projects", createProjectHandler(store))
	// ... register all routes

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

Read `PORT` from environment so deployment (Phase 6) does not require code changes.

---

## 12. Verify with curl

**Context:** `curl` sends HTTP from the terminal. Use it to test every endpoint without writing a client.

```bash
# Start server in another terminal
go run ./cmd/api

# Health
curl -i http://localhost:8080/health

# Create project
curl -i -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"go-practice","description":"Phase 1"}'

# List (copy ID from create response)
curl -s http://localhost:8080/projects

# Get one
curl -s http://localhost:8080/projects/<ID>

# Create task
curl -i -X POST http://localhost:8080/projects/<PROJECT_ID>/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"First task"}'

# Bad JSON → expect 400
curl -i -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d 'not json'

# Unknown ID → expect 404
curl -i http://localhost:8080/projects/00000000-0000-0000-0000-000000000000
```

`-i` shows response headers including status code. `-s` hides progress noise.

### Race detector

Run with Go's race detector to catch missing mutex usage:

```bash
go run -race ./cmd/api
```

Send concurrent requests (shell loop). No race warnings should appear.

---

## 13. README for this phase

Document in `README.md`:

1. What Desklog is (one sentence)
2. Prerequisites (Go version)
3. How to run: `go run ./cmd/api`
4. Default port
5. List of available endpoints
6. One example `curl` for create + list

---

## 14. Common mistakes

| Mistake | Why it is wrong | Fix |
|---------|-----------------|-----|
| `200` on errors | Clients cannot distinguish success | Use 4xx/5xx |
| No `Content-Type` header | Clients may misparse | Always set for JSON |
| Trusting IDs from request body for ownership | Spoofing | Take `project_id` from URL path |
| Map without mutex | Data races under concurrent requests | `sync.Mutex` around map access |
| Giant handlers with everything inline | Unmaintainable; blocks Phase 2 | Extract store, use handler pattern |
| `panic` on bad user input | Crashes whole server | Return 400, log 500 for real bugs |

---

## 15. Exit checklist

- [ ] `go mod init` and `go run ./cmd/api` work
- [ ] All endpoints in section 9 respond correctly
- [ ] Errors return `{"error":"..."}` with correct status
- [ ] Mutex protects in-memory maps
- [ ] README documents how to run and test
- [ ] You can explain: request → handler → store → response

**Commit:** `feat(phase-1): in-memory projects and tasks API`

---

**Next:** [Phase 2 Manual — MongoDB & Structure](./phase-02-mongodb-and-structure.md)
