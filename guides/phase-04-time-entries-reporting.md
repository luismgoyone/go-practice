# Phase 4 Manual — Time Entries & Reporting

**Duration:** 2 weeks  
**Prerequisite:** Phase 3 complete (JWT + user-scoped projects/tasks)  
**Audience:** Beginners — Part A concepts, Part B activities (same as Phases 1–3)

**How to use**

1. Read Part A.
2. Do Part B steps in order; finish each Activity before the next.
3. For every new endpoint, fill a [recipe card](./the-endpoint-recipe.md) first.
4. All new routes stay **behind** auth middleware.

Replace `github.com/<you>/go-practice` with your module path.

---

# Part A — Concepts (read first)

## A0. What you add

| New | Purpose |
|-----|---------|
| `time_entries` collection | Log minutes on a task |
| Nested routes | Create/list under a task |
| `?status=` on task list | Filter |
| `GET /reports/summary` | Minutes per project in a date range |

| Recipe step | Phase 4 |
|-------------|---------|
| Contract | See Step 1 activities |
| Model | `TimeEntry` |
| Data | Time-entry repo + aggregation |
| Business | Ownership chain + minutes/date validation |
| Handler | Nested paths + query params |
| Wire | Protected routes in `main` |
| Verify | Full curl session + tests |

---

## A1. Why a separate time-entry entity

Putting one `minutes` field on a task only stores a total. Real logging needs many sessions, notes, and `logged_at` (when work happened) vs `created_at` (when the row was saved).

---

## A2. URL design

**Nested (create/list):**

```
POST /tasks/{task_id}/time-entries
GET  /tasks/{task_id}/time-entries
```

**Flat (delete):**

```
DELETE /time-entries/{id}
```

`task_id` comes from the **path**, never from the body (prevents logging time on someone else’s task).

---

## A3. Authz chain for time entries

```
user from JWT → load task → load project → project.UserID must match → then create/list/delete
```

Otherwise → `404`.

---

## A4. Aggregation (reports)

Do **not** load all entries into Go and sum. Use a MongoDB aggregation pipeline: `$match` → `$lookup` tasks → `$group` by project → optional project name lookup.

Put the pipeline in the **repository**. Handler only parses `from`/`to` and returns JSON.

Timestamps: RFC3339 UTC (`2026-07-10T14:30:00Z`).

---

# Part B — Build steps

---

## Step 1 — Write the contracts (Activity)

### Activity 1.1 — Fill recipe cards for:

1. `POST /tasks/{task_id}/time-entries`  
   Body: `{"minutes":45,"note":"...","logged_at":"..."}` (`logged_at` optional)  
   Success: `201` + entry  
   Errors: `400` validation, `401`, `404` task/project  

2. `GET /tasks/{task_id}/time-entries` → `200` + array  

3. `DELETE /time-entries/{id}` → `204`  

4. `GET /reports/summary?from=&to=` → `200` + summary object  

5. Stretch/filter: `GET /projects/{id}/tasks?status=done`

**Check:** You can explain each without looking at code. Then continue.

---

## Step 2 — Model + indexes

### File: `internal/model/time_entry.go` (full file)

```go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TimeEntry struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID    primitive.ObjectID `bson:"task_id" json:"task_id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Minutes   int                `bson:"minutes" json:"minutes"`
	Note      string             `bson:"note,omitempty" json:"note,omitempty"`
	LoggedAt  time.Time          `bson:"logged_at" json:"logged_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
```

**Why `user_id` on the entry:** faster auth checks and report `$match` without joining every time.

### Activity 2.1 — Indexes

Append to `scripts/indexes.js`:

```javascript
db.time_entries.createIndex({ task_id: 1, logged_at: -1 });
db.time_entries.createIndex({ user_id: 1, logged_at: -1 });
```

```bash
docker exec -i desklog-mongo mongosh < scripts/indexes.js
```

---

## Step 3 — Time entry repository

### File: `internal/repository/time_entry.go`

Implement:

| Method | Behavior |
|--------|----------|
| `Create(ctx, entry)` | InsertOne; set ID + CreatedAt |
| `ListByTask(ctx, taskID, userID)` | Find `{task_id, user_id}` |
| `Delete(ctx, id, userID)` | DeleteOne `{_id, user_id}`; `ErrNotFound` if 0 |
| `SummaryByProject(ctx, userID, from, to)` | Aggregate (Step 6) — stub returning empty for now is OK |

### Activity 3.1

```bash
go build ./internal/repository/
```

---

## Step 4 — Time entry service (ownership chain)

### File: `internal/service/time_entry.go`

Core create flow:

```go
func (s *TimeEntryService) Create(ctx context.Context, userID, taskID primitive.ObjectID, minutes int, note string, loggedAt *time.Time) (model.TimeEntry, error) {
	// 1. load task (ErrNotFound → return)
	// 2. load project for task.ProjectID
	// 3. if project.UserID != userID → return repository.ErrNotFound
	// 4. validate minutes: > 0 and <= 1440
	// 5. note max 500 chars
	// 6. loggedAt default Now().UTC()
	// 7. repo.Create with TaskID, UserID set from args (not client trust)
}
```

Same ownership check for ListByTask. Delete uses `{id, userID}`.

### Activity 4.1

```bash
go build ./internal/service/
```

---

## Step 5 — Handlers + wire create/list/delete

### File: `internal/handler/time_entry.go`

- Parse `task_id` / entry `id` with `ObjectIDFromHex` → `400` if bad  
- `UserIDFromContext` → `401` if missing  
- Map validation → `400`, not found → `404`  
- Register with `Protect(authSvc, ...)`

### Activity 5.1 — Wire in `main`

```go
http.HandleFunc("POST /tasks/{id}/time-entries", handler.Protect(authSvc, handler.CreateTimeEntryHandler(timeSvc)))
http.HandleFunc("GET /tasks/{id}/time-entries", handler.Protect(authSvc, handler.ListTimeEntriesHandler(timeSvc)))
http.HandleFunc("DELETE /time-entries/{id}", handler.Protect(authSvc, handler.DeleteTimeEntryHandler(timeSvc)))
```

### Activity 5.2 — Curl (login first)

```bash
TOKEN=...
TASK_ID=...

curl -i -X POST "http://localhost:8080/tasks/$TASK_ID/time-entries" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"minutes":60,"note":"Phase 4"}'

curl -s "http://localhost:8080/tasks/$TASK_ID/time-entries" \
  -H "Authorization: Bearer $TOKEN"
```

**Gate:** create + list work for your user; other user’s task id → 404.

---

## Step 6 — Summary report (aggregation)

### Activity 6.1 — Implement `SummaryByProject` in repository

Pipeline idea:

1. `$match` — `user_id` + `logged_at` between `from` and `to`  
2. `$lookup` — tasks on `task_id`  
3. `$unwind` — `$task`  
4. `$group` — `_id: $task.project_id`, `total_minutes: {$sum: $minutes}`  
5. Optional `$lookup` projects for names  

### Response shape (handler/service)

```json
{
  "from": "2026-07-01T00:00:00Z",
  "to": "2026-07-31T23:59:59Z",
  "projects": [
    {"project_id": "...", "name": "go-practice", "total_minutes": 320}
  ]
}
```

Empty → `200` + `"projects": []`.

### Activity 6.2 — Validate query params in service/handler

- Both `from` and `to` required  
- Parse with `time.Parse(time.RFC3339, ...)`  
- `from <= to` or `400`

### Activity 6.3 — Wire + curl

```bash
curl -i "http://localhost:8080/reports/summary?from=2026-07-01T00:00:00Z&to=2026-07-31T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN"

curl -i "http://localhost:8080/reports/summary?from=2026-08-01T00:00:00Z&to=2026-07-01T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"
# expect 400
```

**Gate:** totals match minutes you inserted manually.

---

## Step 7 — Task list `?status=` filter

### Activity 7.1

In list-tasks handler: `status := r.URL.Query().Get("status")`.  
If non-empty, validate `todo|doing|done`; else `400`.  
Pass into service/repo filter `{project_id, user scoping, status?}`.

```bash
curl -s "http://localhost:8080/projects/$PID/tasks?status=done" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Step 8 — Tests + README

### Activity 8.1 — Tests (minimum)

- Invalid date range → validation error / 400  
- Wrong user’s task → 404 on create entry  
- Empty report range → `projects: []`

```bash
go test ./...
```

### Activity 8.2 — README

Table of new endpoints + copy-paste curl: register → login → project → task → time entry → summary.

### Activity 8.3 — Stretch (optional)

Soft-delete tasks with `deleted_at`; exclude from lists/reports.

---

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Summing in Go with FindAll | Aggregation in repository |
| `task_id` in POST body | Path only |
| Local timezone without docs | UTC + RFC3339 |
| Minutes ≤ 0 | Reject with 400 |

---

## Exit checklist

- [ ] Time entry create/list/delete with auth  
- [ ] `?status=` filter on tasks  
- [ ] Summary via aggregation  
- [ ] Date validation  
- [ ] Indexes  
- [ ] README walkthrough  
- [ ] Tests for auth/validation edges  

**Commit:** `feat(phase-4): time entries and summary report`

---

**Always:** [The endpoint recipe](./the-endpoint-recipe.md)  
**Next:** [Phase 5 Manual — Concurrency & Resilience](./phase-05-concurrency-resilience.md)
