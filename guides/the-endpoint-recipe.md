# The endpoint recipe

Use this for **every** new API capability in Desklog — Phase 1 through Phase 6, and anything you add later.

You are not inventing a new process each time. You run the same checklist; only the *contents* of each step change.

---

## The 7 steps

| # | Step | Question you answer | Typical place |
|---|------|---------------------|---------------|
| 1 | **Contract** | Method, path, request JSON, response JSON, status codes? | Write it down first (README, comment, or quiz prompt) |
| 2 | **Model** | What fields does the data have? | `internal/model/` |
| 3 | **Data layer** | How do I create / read / update / delete it? | Phase 1: `internal/store/` · Phase 2+: `internal/repository/` |
| 4 | **Business rules** | What must be true? (exists, owned by user, valid status…) | Phase 1: often in handler · Phase 2+: `internal/service/` |
| 5 | **Handler** | Parse → validate → call layer below → write JSON + status | `internal/handler/` |
| 6 | **Wire** | Register the route; pass dependencies | `cmd/api/main.go` |
| 7 | **Verify** | Does curl (or a test) match the contract? | Terminal / `go test` |

**Rule:** do not start at step 5. If you jump straight to the handler, you will guess types and invent storage as you go — that is how beginners get lost.

---

## Naming

| Kind | Example |
|------|---------|
| Handler function | `CreateProjectHandler`, `HealthHandler`, `ListTasksByProjectHandler` |
| Store / repo method | `CreateProject`, `GetProject`, `ListByProject` |
| File by resource | `handler/project.go`, `handler/task.go` — not one file per route |

Export names that `main` must call (`CreateProjectHandler`). Keep helpers unexported (`writeError`).

---

## Where `main` lives

- **Only** `cmd/api/main.go` has `package main` and `func main`.
- Handlers never call `ListenAndServe` or `HandleFunc` for their own registration — `main` does that.
- Create dependencies (`NewMemoryStore`, DB client, services) **before** `ListenAndServe`. Code after `ListenAndServe` never runs until the server stops.

---

## Phase map (same recipe, different layers)

```
Phase 1:  handler → store (maps in RAM)
Phase 2+: handler → service → repository → MongoDB
Phase 3+: same path + auth middleware + tests
Phase 4+: same path + new models (time entries, reports)
Phase 5:  process concerns (shutdown, workers) — still wire in main
Phase 6:  ship (/ready, Docker) — still one new handler for /ready
```

---

## Quick self-check

Before you write code for a new endpoint, fill this in:

```text
Contract:  METHOD /path
Request:   { ... } or empty
Success:   STATUS + { ... }
Errors:    400 / 401 / 404 / ...
Model:     (new fields? which file?)
Data:      (which store/repo method?)
Rules:     (what must be true?)
Handler:   (which file? function name?)
Wire:      HandleFunc("METHOD /path", ...)
Verify:    curl ...
```

If you cannot fill the contract line, you are not ready to code yet.
