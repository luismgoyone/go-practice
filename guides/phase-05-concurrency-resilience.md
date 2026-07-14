# Phase 5 Manual — Concurrency & Resilience

**Duration:** 1 week  
**Prerequisite:** Phase 4 complete  
**Audience:** Beginners — Part A + Part B activities

**How to use**

1. Read Part A (goroutines, shutdown, workers).
2. Do Part B steps; each ends with an Activity.
3. Prefer wiring in `main` over `go func()` inside handlers.
4. Recipe still applies for any HTTP change ([endpoint recipe](./the-endpoint-recipe.md)).

---

# Part A — Concepts (read first)

## A0. What changes

| Addition | Purpose |
|----------|---------|
| Graceful shutdown | Finish in-flight requests on Ctrl+C |
| Request timeouts | Cancel hung DB/handlers |
| Bounded worker pool | Background jobs without unbounded goroutines |
| `task.completed` job | Side effect when status → `done` |

You already had concurrency: each HTTP request runs in its own goroutine. Phase 5 adds **your** background goroutines.

**Rule:** if the client does not need the result in the HTTP response, consider async — after the DB write succeeds.

---

## A1. Graceful shutdown order

```
1. HTTP Shutdown (no new requests; drain active)
2. Cancel worker context
3. WaitGroup.Wait (workers finish current job)
4. Mongo Disconnect
```

---

## A2. Channels and backpressure

```go
jobs := make(chan Job, 100) // buffer
```

When full, `Enqueue` should **not** block forever — return an error or drop + log.

---

## A3. When to enqueue `task.completed`

1. Update task in Mongo (**sync**, must succeed)  
2. Enqueue job  
3. Return `200` to client  
4. Worker posts webhook / logs  

Never enqueue before the DB write succeeds.

---

# Part B — Build steps

---

## Step 1 — Request timeout middleware

### File: `internal/handler/timeout.go` (full file)

```go
package handler

import (
	"context"
	"net/http"
	"time"
)

func WithTimeout(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

### Activity 1.1 — Wire

Use an `http.ServeMux`, wrap it:

```go
mux := http.NewServeMux()
// register routes on mux...
var h http.Handler = mux
h = handler.WithTimeout(30*time.Second, h)
// later: http.Server{Handler: h}
```

**Why:** hung Mongo calls get canceled via `r.Context()` you already pass down.

### Activity 1.2

Restart server; confirm normal curls still work.

---

## Step 2 — Graceful HTTP shutdown

### Activity 2.1 — Replace bare `ListenAndServe` in `main`

Pattern:

```go
srv := &http.Server{Addr: ":" + port, Handler: h}

go func() {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
	log.Println("shutdown:", err)
}
// then stop workers + mongo.Disconnect (Step 4)
```

Imports: `os/signal`, `syscall`.

### Activity 2.2 — Verify

Terminal 1: `go run ./cmd/api`  
Terminal 2:

```bash
while true; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health; sleep 0.2; done
```

Ctrl+C in Terminal 1. Expect clean stop; loop shows codes then connection errors — not a hang.

**Gate:** shutdown completes without force-kill.

---

## Step 3 — Worker package

### File: `internal/worker/worker.go` (full file skeleton)

```go
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

type Job struct {
	Type    string
	Payload map[string]string
}

type Worker struct {
	jobs   chan Job
	logger *slog.Logger
	client *http.Client
}

func New(buffer int, logger *slog.Logger) *Worker {
	return &Worker{
		jobs:   make(chan Job, buffer),
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second}, // mandatory
	}
}

func (w *Worker) Enqueue(job Job) error {
	select {
	case w.jobs <- job:
		return nil
	default:
		return errors.New("job queue full")
	}
}

func (w *Worker) Start(ctx context.Context, n int, wg *sync.WaitGroup) {
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-w.jobs:
					w.handle(job)
				}
			}
		}()
	}
}

func (w *Worker) handle(job Job) {
	switch job.Type {
	case "task.completed":
		w.handleTaskCompleted(job.Payload)
	default:
		w.logger.Warn("unknown job", "type", job.Type)
	}
}

func (w *Worker) handleTaskCompleted(payload map[string]string) {
	url := os.Getenv("WEBHOOK_URL")
	if url == "" {
		w.logger.Info("task.completed", "payload", payload)
		return
	}
	body, _ := json.Marshal(payload)
	resp, err := w.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		w.logger.Error("webhook failed", "err", err)
		return
	}
	defer resp.Body.Close()
	w.logger.Info("webhook sent", "status", resp.StatusCode)
}
```

**Why client timeout:** a dead webhook must not stick a worker forever.

### Activity 3.1

```bash
go build ./internal/worker/
```

---

## Step 4 — Wire worker in `main` + shutdown

### Activity 4.1

```go
workerCtx, workerCancel := context.WithCancel(context.Background())
var workerWg sync.WaitGroup
w := worker.New(100, slog.Default())
w.Start(workerCtx, 3, &workerWg)

// pass w into TaskService constructor

// on shutdown AFTER srv.Shutdown:
workerCancel()
workerWg.Wait()
_ = mongoClient.Disconnect(shutdownCtx)
```

### Activity 4.2 — Inject into task update

When status becomes `"done"` **after** successful Mongo update:

```go
_ = w.Enqueue(worker.Job{
	Type: "task.completed",
	Payload: map[string]string{
		"task_id":    task.ID.Hex(),
		"project_id": task.ProjectID.Hex(),
		"user_id":    userID.Hex(),
	},
})
```

If queue full: log error; still return 200 for the HTTP update (document this).

### Activity 4.3

```bash
# PATCH task to done with auth
# watch server logs for task.completed AFTER the curl returns
```

Optional: set `WEBHOOK_URL` to https://httpbin.org/post and confirm worker logs status 200.

**Gate:** mark done → log line appears; Ctrl+C still drains cleanly.

---

## Step 5 — README + tests

### Activity 5.1

Document: shutdown behavior, `WEBHOOK_URL`, job fired when task → done.

### Activity 5.2

```bash
go test ./...
```

Still green.

### Activity 5.3 — Stretch (optional)

- Persistent jobs collection / outbox  
- In-memory TTL cache for report summaries  

---

## Common mistakes

| Mistake | Fix |
|---------|-----|
| `go sendWebhook()` per request | Bounded worker channel |
| No HTTP client timeout | Set `Timeout` |
| Enqueue before DB write | Update first |
| Exit without `Shutdown` | Use signal + `srv.Shutdown` |

---

## Exit checklist

- [ ] Graceful shutdown on SIGINT/SIGTERM  
- [ ] Request timeout middleware  
- [ ] Worker pool + bounded queue  
- [ ] `task.completed` after status → done  
- [ ] Webhook client has timeout  
- [ ] README updated  
- [ ] `go test ./...` passes  

**Commit:** `feat(phase-5): graceful shutdown and background workers`

---

**Always:** [The endpoint recipe](./the-endpoint-recipe.md)  
**Next:** [Phase 6 Manual — Ship It](./phase-06-ship-it.md)
