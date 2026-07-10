# Phase 5 Manual — Concurrency & Resilience

**Duration:** 1 week  
**Prerequisite:** Phase 4 complete

Desklog's features are complete. Phase 5 makes the **process** behave well: shut down cleanly, don't hang forever, don't block HTTP while doing slow side work. This is what separates a demo from something you could actually deploy.

---

## 0. What changes in this phase

| Addition | Purpose |
|----------|---------|
| Graceful shutdown | Finish in-flight requests before exit |
| Request timeouts | Cancel slow operations |
| Background worker | Async jobs (webhook on task completed) |
| Bounded job queue | Prevent unbounded goroutines |

---

## 1. Concurrency you already have

### Go's HTTP server is concurrent

`net/http` spawns a **goroutine** per incoming request. You have been writing concurrent code since Phase 1.

A **goroutine** is a lightweight thread managed by the Go runtime. Thousands can run on one machine. They share memory — which is why Phase 1 needed `sync.Mutex` on maps.

### What Phase 5 adds

Your **own** goroutines for work that should not block the HTTP response:

- Sending a webhook after marking a task done
- (Later) email digests, exports

**Rule:** If the client does not need the result to form the HTTP response, consider doing it asynchronously.

---

## 2. Graceful shutdown

### The problem

When you deploy or press Ctrl+C, the OS sends **SIGINT** or **SIGTERM** to your process. If you exit immediately:

- In-flight HTTP requests are cut off mid-response
- Clients see connection errors
- Database writes may be half-done

### The solution

1. Stop accepting new connections
2. Wait for active requests to finish (with a deadline)
3. Stop background workers
4. Close database connection
5. Exit

### Implementation

```go
srv := &http.Server{
	Addr:    ":" + port,
	Handler: mux,
}

go func() {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit  // block until signal

log.Println("shutting down...")
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
	log.Fatal("shutdown:", err)
}

// disconnect mongo, stop workers — see below
log.Println("server stopped")
```

### What Shutdown does

`srv.Shutdown(ctx)`:

- Closes the listener (no new connections)
- Waits for active handlers to return
- Returns when all done OR context deadline exceeded

`http.ErrServerClosed` from `ListenAndServe` is expected — not a real error.

### Shutdown order

```
1. HTTP Shutdown (stop new requests, drain active)
2. Cancel worker context (workers stop accepting new jobs)
3. Wait for workers to finish current job (WaitGroup)
4. mongo.Client.Disconnect
```

---

## 3. Request timeouts

### The problem

A slow MongoDB query or bug can hang a handler forever, tying up a goroutine and possibly a connection.

### context.WithTimeout

Wrap the request context:

```go
func withTimeout(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

Apply globally (e.g. 30 seconds) when registering routes.

### Propagation

Repository methods already take `ctx`. When timeout fires:

- `Find`, `InsertOne`, etc. return `context deadline exceeded`
- Handler maps to 503 Service Unavailable or 500 — document your choice

MongoDB driver respects context cancellation.

---

## 4. Channels and worker pools

### Channels

A **channel** is a typed queue for sending data between goroutines:

```go
jobs := make(chan Job, 100)  // buffer 100 jobs
jobs <- Job{Type: "task.completed"}  // send (blocks if buffer full)
j := <-jobs  // receive
```

Buffered channel (`100`) lets producers enqueue without blocking until buffer fills — **backpressure**.

### Worker struct

```go
type Job struct {
	Type    string
	Payload map[string]string
}

type Worker struct {
	jobs   chan Job
	logger *slog.Logger
	client *http.Client
}

func NewWorker(bufferSize int, logger *slog.Logger) *Worker {
	return &Worker{
		jobs:   make(chan Job, bufferSize),
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}
```

### Enqueue

```go
func (w *Worker) Enqueue(job Job) error {
	select {
	case w.jobs <- job:
		return nil
	default:
		return errors.New("job queue full")
	}
}
```

If queue full: log and return error to service (or drop — document behavior). Never block HTTP indefinitely.

### Start workers

```go
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
```

Start 2–3 workers in `main`. Pass a `sync.WaitGroup` to wait on shutdown.

### sync.WaitGroup

Counts active goroutines:

- `wg.Add(1)` before starting goroutine
- `wg.Done()` when goroutine exits
- `wg.Wait()` blocks until count is zero

Use on shutdown after canceling worker context.

---

## 5. The task.completed job

### Trigger

When a task is patched to `status: "done"`:

1. Service updates task in MongoDB (**synchronous** — must succeed before response)
2. Service enqueues job (**after** successful DB write)
3. Handler returns 200 with updated task
4. Worker processes job in background

**Never enqueue before DB commit** — webhook fires, then DB fails = inconsistent state.

### Job payload

```go
worker.Enqueue(Job{
	Type: "task.completed",
	Payload: map[string]string{
		"task_id":    task.ID.Hex(),
		"project_id": task.ProjectID.Hex(),
		"user_id":    userID.Hex(),
	},
})
```

### Worker handle

```go
func (w *Worker) handle(job Job) {
	switch job.Type {
	case "task.completed":
		w.handleTaskCompleted(job.Payload)
	default:
		w.logger.Warn("unknown job type", "type", job.Type)
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

### HTTP client timeout

**Mandatory.** Without `Timeout`, a dead webhook server hangs the worker goroutine forever.

### Retries (stretch)

On failure: retry 3 times with 1s, 2s, 4s sleep. Log final failure. Do not block queue on infinite retry.

---

## 6. Persistent job queue (stretch)

In-memory channel loses jobs on restart. Production systems use a **jobs collection**:

```json
{
  "_id": "ObjectId",
  "type": "task.completed",
  "payload": { "task_id": "..." },
  "status": "pending",
  "created_at": "..."
}
```

Worker polls `{ status: "pending" }` or uses change streams. Jobs survive restart.

### Outbox pattern

In one MongoDB **transaction**:

1. Update task status to done
2. Insert job document with status pending

Guarantees: no "done" without a recorded side effect. Requires replica set for transactions.

---

## 7. Caching report summary (optional)

### When to cache

`GET /reports/summary` runs an aggregation — expensive on large datasets. Cache result keyed by `userID + from + to`.

### In-memory cache

```go
type cacheEntry struct {
	data      ReportSummary
	expiresAt time.Time
}

type ReportCache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
	ttl   time.Duration
}
```

- `RLock` for reads (multiple readers OK)
- `Lock` for writes
- TTL 60 seconds — document that data may be stale

Invalidate on new time entry (stretch) or accept staleness for simplicity.

---

## 8. Wire in main

```go
workerCtx, workerCancel := context.WithCancel(context.Background())
var workerWg sync.WaitGroup

worker := NewWorker(100, logger)
worker.Start(workerCtx, 3, &workerWg)

taskSvc := service.NewTaskService(..., worker)  // service can enqueue

// on shutdown:
workerCancel()
workerWg.Wait()
_ = mongoClient.Disconnect(shutdownCtx)
_ = srv.Shutdown(shutdownCtx)
```

Pass worker to service via constructor — same dependency injection as repositories.

---

## 9. Verify shutdown

Terminal 1:

```bash
go run ./cmd/api
```

Terminal 2 — loop requests:

```bash
while true; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health; done
```

Terminal 1: Ctrl+C. Loop should see clean responses until server stops — no hung connections.

Mark task done → check logs for worker processing webhook/log **after** response returned.

---

## 10. Common mistakes

| Mistake | Consequence |
|---------|-------------|
| `go webhook()` per request unbounded | Memory exhaustion under load |
| No HTTP client timeout | Worker stuck forever |
| Enqueue before DB write | False events |
| Exit without Shutdown | Broken client responses |
| Ignoring full job queue | Silent job loss — log it |

---

## 11. Exit checklist

- [ ] SIGINT/SIGTERM triggers graceful HTTP shutdown
- [ ] Request timeout middleware
- [ ] Worker pool with bounded channel
- [ ] `task.completed` job on status → done
- [ ] Webhook client has timeout
- [ ] README documents shutdown and job flow
- [ ] `go test ./...` still passes

**Commit:** `feat(phase-5): graceful shutdown and background workers`

---

**Next:** [Phase 6 Manual — Ship It](./phase-06-ship-it.md)
