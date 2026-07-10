# Phase 4 Manual — Time Entries & Reporting

**Duration:** 2 weeks  
**Prerequisite:** Phase 3 complete

Projects and tasks describe *what* you might work on. **Time entries** record *what you actually did and for how long*. This phase also introduces **reporting** — read patterns that aggregate data instead of simple fetch-by-ID.

---

## 0. What changes in this phase

| New | Purpose |
|-----|---------|
| `time_entries` collection | Log minutes worked on a task |
| Nested routes under tasks | REST design for child resources |
| Query param filters | `?status=done` on task lists |
| `GET /reports/summary` | Aggregated minutes per project in a date range |

---

## 1. Time entry domain

### Why a separate entity

You could add `minutes` to a task directly. That only supports one number total. Real work logging needs:

- Multiple entries per task ("morning: 2h", "afternoon: 1h")
- Notes per session
- `logged_at` different from `created_at` (backfilling yesterday's work)

### Document structure

```go
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

### Field decisions explained

| Field | Why |
|-------|-----|
| `task_id` | Which task this time belongs to |
| `user_id` | Denormalized — speeds authorization and reports without extra joins in every check |
| `minutes` | Integer minutes (simpler than fractional hours for v1) |
| `logged_at` | When work happened |
| `created_at` | When record was created in the system |

### Validation rules

| Field | Rule |
|-------|------|
| minutes | integer > 0, max 1440 (24 hours) per entry |
| note | optional, max 500 characters |
| logged_at | valid timestamp; default `time.Now().UTC()` if omitted |

---

## 2. Nested resources — URL design

### Parent-child in URLs

Time entries belong to tasks. Two valid styles:

**Nested (Desklog uses this for create/list):**

```
GET  /tasks/{task_id}/time-entries
POST /tasks/{task_id}/time-entries
```

**Flat (Desklog uses this for delete by entry ID):**

```
DELETE /time-entries/{id}
```

**Why mix styles:** Creating requires knowing the parent task (nested). Deleting by entry ID only needs the entry's global ID — flat is simpler.

### Path vs body

`task_id` comes from the **URL path** on POST, not the body. If you allowed `task_id` in the body, a client could log time against someone else's task while authenticated as themselves.

POST body:

```json
{
  "minutes": 45,
  "note": "Implemented repository",
  "logged_at": "2026-07-10T14:30:00Z"
}
```

---

## 3. Authorization for time entries

### Create entry — check chain

```
1. Get userID from context (JWT)
2. Parse task_id from path
3. Load task by ID
4. Load project for task.ProjectID
5. If project.UserID != userID → 404 (or 403)
6. Insert entry with task_id, user_id, minutes, logged_at
```

### List entries for task

Same ownership check before returning entries. Filter:

```go
bson.M{"task_id": taskID, "user_id": userID}
```

### Delete entry

Load entry → verify `entry.UserID == currentUser` → delete. Or 404 if not found/not owned.

---

## 4. Query parameters — filtering tasks

### Path vs query

- **Path parameter** identifies a resource: `/projects/{id}`
- **Query parameter** filters or modifies the response: `/projects/{id}/tasks?status=done`

### Implementation

Handler reads query:

```go
status := r.URL.Query().Get("status")
```

Service/repository:

```go
filter := bson.M{"project_id": projectID, "user_id": userID}
if status != "" {
	if !isValidStatus(status) {
		return nil, &ValidationError{Field: "status", Message: "invalid"}
	}
	filter["status"] = status
}
```

Empty `status` → return all tasks. Invalid status → 400, not silent ignore.

---

## 5. Time and date ranges

### RFC3339

API timestamps use **RFC3339** format — ISO 8601 subset:

```
2026-07-10T14:30:00Z
```

`Z` means UTC. Always store and compare in UTC internally.

### Parsing in Go

```go
t, err := time.Parse(time.RFC3339, s)
if err != nil {
	return time.Time{}, &ValidationError{Field: "logged_at", Message: "must be RFC3339"}
}
```

### Report query params

```
GET /reports/summary?from=2026-07-01T00:00:00Z&to=2026-07-31T23:59:59Z
```

Validation:

- Both `from` and `to` required
- Both must parse as RFC3339
- `from` must be ≤ `to`
- Else → 400

Document in README: **all API times are UTC**.

---

## 6. MongoDB aggregation — how reporting works

### Why not loop in Go

Loading all time entries into Go and summing works for 100 rows. At scale it wastes memory and network. **Aggregation** runs the computation on the database server.

An **aggregation pipeline** is a sequence of stages. Each stage transforms documents and passes results to the next.

### Pipeline for summary report

**Goal:** Total minutes per project for one user in a date range.

**Stage 1 — `$match`:** Filter entries early (reduces work for later stages)

```javascript
{ $match: {
    user_id: ObjectId("..."),
    logged_at: { $gte: ISODate("2026-07-01"), $lte: ISODate("2026-07-31") }
}}
```

**Stage 2 — `$lookup`:** Join tasks collection (like a left join)

```javascript
{ $lookup: {
    from: "tasks",
    localField: "task_id",
    foreignField: "_id",
    as: "task"
}}
```

**Stage 3 — `$unwind`:** One document per task (lookup returns array)

```javascript
{ $unwind: "$task" }
```

**Stage 4 — `$group`:** Sum minutes by project_id

```javascript
{ $group: {
    _id: "$task.project_id",
    total_minutes: { $sum: "$minutes" }
}}
```

**Stage 5 — `$lookup` (optional):** Join projects for names

```javascript
{ $lookup: {
    from: "projects",
    localField: "_id",
    foreignField: "_id",
    as: "project"
}}
{ $unwind: "$project" }
```

### In Go

```go
pipeline := mongo.Pipeline{
	{{Key: "$match", Value: bson.D{...}}},
	{{Key: "$lookup", Value: bson.D{...}}},
	{{Key: "$unwind", Value: "$task"}},
	{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: "$task.project_id"},
		{Key: "total_minutes", Value: bson.D{{Key: "$sum", Value: "$minutes"}}},
	}}},
}

cursor, err := r.col.Aggregate(ctx, pipeline)
```

Put pipeline in `repository/report.go` — handler calls service, service calls report repository.

### Response shape

```json
{
  "from": "2026-07-01T00:00:00Z",
  "to": "2026-07-31T23:59:59Z",
  "projects": [
    {
      "project_id": "507f1f77bcf86cd799439011",
      "name": "go-practice",
      "total_minutes": 320
    }
  ]
}
```

Empty range with no entries: `200` + `"projects": []`

---

## 7. Indexes for time entries

```javascript
db.time_entries.createIndex({ task_id: 1, logged_at: -1 });
db.time_entries.createIndex({ user_id: 1, logged_at: -1 });
```

| Index | Supports |
|-------|----------|
| `task_id + logged_at` | List entries for a task, sorted recent first |
| `user_id + logged_at` | Report `$match` on user + date range |

`-1` = descending (newest first).

---

## 8. Repository methods to add

| Method | Operation |
|--------|-----------|
| `Create(ctx, entry)` | InsertOne |
| `ListByTask(ctx, taskID, userID)` | Find with filter |
| `Delete(ctx, id, userID)` | DeleteOne with user scope |
| `SummaryByProject(ctx, userID, from, to)` | Aggregate pipeline |

---

## 9. API documentation in README

Add a section listing every endpoint:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /tasks/{id}/time-entries | Yes | Log time |
| GET | /reports/summary | Yes | Minutes per project |

Include full curl example flow:

1. Register
2. Login → save token
3. Create project
4. Create task
5. Log time entry
6. Fetch summary

Someone cloning your repo should complete this flow using only the README.

---

## 10. Stretch: soft delete

Add optional field to tasks:

```go
DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
```

On delete: set `deleted_at` instead of removing document. All queries add:

```go
filter["deleted_at"] = bson.M{"$exists": false}
```

Reports should exclude time on soft-deleted tasks (filter in aggregation or service).

---

## 11. Build order

1. TimeEntry model + indexes
2. Time entry repository
3. Service with authorization chain
4. Handlers for list/create/delete
5. Task status filter on list endpoint
6. Report repository with aggregation
7. Report handler + date validation
8. Tests: invalid date range, wrong user's task, empty report
9. README API docs

---

## 12. Verify

```bash
TOKEN="..."  # from login
TASK_ID="..."

# Log time
curl -s -X POST "http://localhost:8080/tasks/$TASK_ID/time-entries" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"minutes":60,"note":"Phase 4 work"}'

# List entries
curl -s "http://localhost:8080/tasks/$TASK_ID/time-entries" \
  -H "Authorization: Bearer $TOKEN"

# Summary
curl -s "http://localhost:8080/reports/summary?from=2026-07-01T00:00:00Z&to=2026-07-31T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN"

# Invalid range
curl -i "http://localhost:8080/reports/summary?from=2026-08-01T00:00:00Z&to=2026-07-01T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN"
```

Manually verify totals: add entries with known minutes, confirm summary sums match.

---

## 13. Common mistakes

| Mistake | Problem |
|---------|---------|
| Summing in handler with FindAll | Does not scale; wrong layer |
| `$lookup` before `$match` | Processes more data than needed |
| Accepting `task_id` in POST body | Authorization bypass |
| Local time without docs | Bugs across timezones |
| Negative/zero minutes | Corrupts reports |

---

## 14. Exit checklist

- [ ] Time entry CRUD with auth
- [ ] Task list filter by `?status=`
- [ ] Summary report via aggregation
- [ ] Date range validation
- [ ] Indexes on time_entries
- [ ] README API reference with full curl flow
- [ ] Tests for auth and validation edge cases

**Commit:** `feat(phase-4): time entries and summary report`

---

**Next:** [Phase 5 Manual — Concurrency & Resilience](./phase-05-concurrency-resilience.md)
