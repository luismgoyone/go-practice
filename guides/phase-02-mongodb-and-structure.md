# Phase 2 Manual — MongoDB & Structure

**Duration:** 2–3 weeks  
**Prerequisite:** Phase 1 complete (projects + tasks API working in memory)  
**You need:** Go 1.22+, Docker, terminal, `curl`  
**Audience:** Same as Phase 1 — beginners; follow steps in order

**How to use this document**

1. Read **Part A** (concepts) so MongoDB and layers make sense.
2. Follow **Part B** (build steps) in order. Each step gives **full file contents**, explains **why**, then an **Activity** to run before continuing.
3. Keep [The endpoint recipe](./the-endpoint-recipe.md) open. Phase 2 only changes the middle of the recipe (store → repository + service).
4. Migrate **one vertical slice at a time** (e.g. `POST /projects` end-to-end). Do not rewrite every file before testing.

Replace `github.com/<you>/go-practice` with your module path from `go.mod`.

---

# Part A — Concepts (read first)

## A0. What changes

| Before (Phase 1) | After (Phase 2) |
|------------------|-----------------|
| Data in maps in RAM | Data in MongoDB on disk |
| Handler talks to `store` | Handler → **service** → **repository** → MongoDB |
| String IDs (`newID()`) | MongoDB `ObjectID` in DB; hex string in JSON |
| Restart = empty data | Restart = data still there |

Your HTTP routes stay the same. You are swapping **where data lives** and **where business rules live**.

| Recipe step | Phase 1 | Phase 2 |
|-------------|---------|---------|
| Contract | Same paths/statuses | Same; IDs look like `"507f1f77bcf86cd799439011"` |
| Model | JSON tags | JSON **and** BSON tags; `primitive.ObjectID` |
| Data | `internal/store` | `internal/repository` |
| Business rules | Often in handler | `internal/service` |
| Handler | parse → store → respond | parse → **service** → respond (**no** Mongo collection APIs) |
| Wire | `NewMemoryStore()` | Mongo client + repos + services |
| Verify | curl | curl **+ restart** + check mongosh |

**Rule:** never call MongoDB from a handler. Handlers speak HTTP only.

---

## A1. Layered architecture

```
Client (curl)
    ↓ HTTP
handler      — parse request, write JSON + status codes
    ↓
service      — rules: "project must exist before creating a task"
    ↓
repository   — InsertOne / FindOne / UpdateOne / Delete*
    ↓
MongoDB
```

**Why layers**

| If you put Mongo in the handler… | Problem |
|----------------------------------|---------|
| You want to test “title required” | You need a real database |
| You switch to Postgres later | You rewrite every handler |

Dependencies point **down**: handler knows service; service knows repository; repository knows the driver. Never the reverse.

### New folders

```
cmd/api/main.go
internal/
  model/          # updated tags + ObjectID
  repository/     # MongoDB only
  service/        # business rules
  handler/        # HTTP only (edit existing files)
```

You can delete or stop using `internal/store/` after migration is done.

---

## A2. MongoDB in one page

| SQL idea | MongoDB |
|----------|---------|
| Database | Database (`desklog`) |
| Table | Collection (`projects`, `tasks`) |
| Row | Document (BSON object) |
| Primary key | `_id` (usually ObjectId) |

Desklog keeps **two collections**. A task stores `project_id` pointing at a project’s `_id` (reference, not embedding).

**ObjectID**

```go
id := primitive.NewObjectID()                 // create
id, err := primitive.ObjectIDFromHex(hex)     // parse from URL
id.Hex()                                      // string form
```

Invalid hex in the URL → `400` (this guide’s choice). Stay consistent.

**BSON vs JSON**

- JSON = API over HTTP  
- BSON = what MongoDB stores  
- One Go struct can have both tags:

```go
ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
Name string             `bson:"name" json:"name"`
```

---

## A3. context.Context

Every DB call takes `context.Context`:

```go
func (r *ProjectRepo) GetByID(ctx context.Context, id primitive.ObjectID) (model.Project, error)
```

Handlers pass `r.Context()` from the HTTP request. That way, if the client disconnects or a timeout fires, the DB work can stop.

---

## A4. Errors → HTTP

| Domain / repo error | HTTP |
|---------------------|------|
| `repository.ErrNotFound` | 404 |
| validation failure | 400 |
| anything unexpected | 500 (log the real error; return generic message) |

Define one sentinel:

```go
var ErrNotFound = errors.New("not found")
```

Compare with `errors.Is(err, repository.ErrNotFound)`.

---

# Part B — Build steps (do these in order)

Each step: **edit files → read why → complete the Activity → only then continue.**

---

## Step 1 — Run MongoDB and set env vars

**Goal:** A MongoDB process your app can reach.

### Activity 1.1 — Start Mongo with Docker

```bash
docker run -d --name desklog-mongo -p 27017:27017 mongo:7
```

| Flag | Why |
|------|-----|
| `-d` | Run in background |
| `--name desklog-mongo` | Easy to start/stop later |
| `-p 27017:27017` | Your machine talks to Mongo on localhost:27017 |
| `mongo:7` | Official image, version 7 |

Check it is alive:

```bash
docker exec -it desklog-mongo mongosh --eval 'db.runCommand({ ping: 1 })'
```

Expect `{ ok: 1 }`.

### Activity 1.2 — Env vars for the app

In the terminal where you will run the API:

```bash
export MONGODB_URI=mongodb://localhost:27017
export MONGODB_DATABASE=desklog
export PORT=8080
```

| Variable | Why |
|----------|-----|
| `MONGODB_URI` | Where Mongo listens — **never hard-code** in source |
| `MONGODB_DATABASE` | Database name inside Mongo |
| `PORT` | Same idea as Phase 1 |

**Check:** `echo $MONGODB_URI` prints the URI. Do not start the Go app yet.

---

## Step 2 — Drivers and folders

### Activity 2.1 — Install the driver

From the project root:

```bash
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/bson
go get go.mongodb.org/mongo-driver/bson/primitive
go get go.mongodb.org/mongo-driver/mongo/options
go mod tidy
```

### Activity 2.2 — Create packages

```bash
mkdir -p internal/repository internal/service scripts
```

**Why:** `repository` = data layer; `service` = rules; `scripts` = index definitions (later).

**Check:** folders exist; `go.mod` lists `go.mongodb.org/mongo-driver`.

---

## Step 3 — Update models for MongoDB

**Recipe:** model step. Same structs; new ID type and BSON tags.

### File: `internal/model/project.go` (replace full file)

```go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Project is stored in the "projects" collection.
type Project struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}
```

### File: `internal/model/task.go` (replace full file)

```go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Task is stored in the "tasks" collection.
type Task struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProjectID primitive.ObjectID `bson:"project_id" json:"project_id"`
	Title     string             `bson:"title" json:"title"`
	Status    string             `bson:"status" json:"status"` // todo | doing | done
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
```

**Why this block**

| Piece | Why |
|-------|-----|
| `primitive.ObjectID` | Native Mongo `_id` type |
| `bson:"_id,omitempty"` | Mongo primary key field name |
| `json:"id"` | API still uses `"id"` (ObjectID marshals as hex string in JSON) |
| `bson:"project_id"` | Task → project reference |

### Activity 3.1 — Save models

Your Phase 1 handlers/store still use `string` IDs — the full app will not compile until later steps. That is expected. Confirm these two model files look correct and save them.

---

## Step 4 — Repository: errors + project CRUD

**Recipe:** data layer.

### File: `internal/repository/errors.go` (full file)

```go
package repository

import "errors"

// ErrNotFound means no document matched the query.
// Handlers map this to HTTP 404.
var ErrNotFound = errors.New("not found")
```

**Why:** one shared signal instead of leaking `mongo.ErrNoDocuments` into handlers.

### File: `internal/repository/project.go` (full file)

```go
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/<you>/go-practice/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProjectRepo struct {
	col *mongo.Collection
}

func NewProjectRepo(db *mongo.Database) *ProjectRepo {
	return &ProjectRepo{col: db.Collection("projects")}
}

func (r *ProjectRepo) Create(ctx context.Context, p model.Project) (model.Project, error) {
	p.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.col.InsertOne(ctx, p)
	if err != nil {
		return model.Project{}, err
	}
	return p, nil
}

func (r *ProjectRepo) List(ctx context.Context) ([]model.Project, error) {
	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	projects := make([]model.Project, 0)
	if err := cursor.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id primitive.ObjectID) (model.Project, error) {
	var p model.Project
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Project{}, ErrNotFound
		}
		return model.Project{}, err
	}
	return p, nil
}

func (r *ProjectRepo) Update(ctx context.Context, p model.Project) (model.Project, error) {
	p.UpdatedAt = time.Now().UTC()
	res, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": p.ID},
		bson.M{"$set": bson.M{
			"name":        p.Name,
			"description": p.Description,
			"updated_at":  p.UpdatedAt,
		}},
	)
	if err != nil {
		return model.Project{}, err
	}
	if res.MatchedCount == 0 {
		return model.Project{}, ErrNotFound
	}
	return p, nil
}

func (r *ProjectRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
```

**Why this block**

| Piece | Why |
|-------|-----|
| `*mongo.Collection` | One collection handle; no new client per call |
| `InsertOne` / `Find` / `FindOne` / `UpdateOne` / `DeleteOne` | CRUD primitives |
| `bson.M{"_id": id}` | Filter by primary key |
| `$set` | Update only listed fields |
| Map `ErrNoDocuments` → `ErrNotFound` | Missing doc is not a 500 |

### Activity 4.1 — Packages compile

```bash
go build ./internal/repository/
```

Expect success.

---

## Step 5 — Project service (no cascade yet)

**Recipe:** business rules.

### File: `internal/service/errors.go` (full file)

```go
package service

import (
	"errors"

	"github.com/<you>/go-practice/internal/repository"
)

// ValidationError is a business-rule failure (HTTP 400).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// IsNotFound reports repository.ErrNotFound without handlers importing mongo.
func IsNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}
```

### File: `internal/service/project.go` (full file for this step)

```go
package service

import (
	"context"
	"strings"

	"github.com/<you>/go-practice/internal/model"
	"github.com/<you>/go-practice/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectService struct {
	repo *repository.ProjectRepo
}

func NewProjectService(repo *repository.ProjectRepo) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) Create(ctx context.Context, name, description string) (model.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Project{}, &ValidationError{Message: "name is required"}
	}
	return s.repo.Create(ctx, model.Project{
		Name:        name,
		Description: description,
	})
}

func (s *ProjectService) List(ctx context.Context) ([]model.Project, error) {
	return s.repo.List(ctx)
}

func (s *ProjectService) GetByID(ctx context.Context, id primitive.ObjectID) (model.Project, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProjectService) Update(ctx context.Context, id primitive.ObjectID, name, description *string) (model.Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return model.Project{}, err
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return model.Project{}, &ValidationError{Message: "name is required"}
		}
		p.Name = n
	}
	if description != nil {
		p.Description = *description
	}
	return s.repo.Update(ctx, p)
}

func (s *ProjectService) Delete(ctx context.Context, id primitive.ObjectID) error {
	return s.repo.Delete(ctx, id)
}
```

**Why:** validation lives in the service so handlers stay thin. Cascade delete comes in Step 8 after `TaskRepo` exists.

### Activity 5.1

```bash
go build ./internal/service/
```

---

## Step 6 — First vertical slice: projects on Mongo

**Goal:** App connects to Mongo; create/list/get projects persist. Leave task routes off until Step 9.

Keep Phase 1 `writeError` and `HealthHandler`. Replace project handlers to take `*service.ProjectService`.

### File: `internal/handler/project.go` (full file)

```go
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/<you>/go-practice/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateProjectHandler(svc *service.ProjectService) http.HandlerFunc {
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

		project, err := svc.Create(r.Context(), req.Name, req.Description)
		var ve *service.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, ve.Message)
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(project)
	}
}

func ListProjectsHandler(svc *service.ProjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects, err := svc.List(r.Context())
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(projects)
	}
}

func GetProjectHandler(svc *service.ProjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		project, err := svc.GetByID(r.Context(), id)
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(project)
	}
}

func UpdateProjectHandler(svc *service.ProjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		project, err := svc.Update(r.Context(), id, req.Name, req.Description)
		var ve *service.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, ve.Message)
			return
		}
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(project)
	}
}

func DeleteProjectHandler(svc *service.ProjectService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		err = svc.Delete(r.Context(), id)
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
```

**Why this block**

| Piece | Why |
|-------|-----|
| `svc *service.ProjectService` | Handler depends on service, not Mongo |
| `r.Context()` | Ties HTTP request lifetime to DB calls |
| `ObjectIDFromHex` | URL ids are strings; DB needs ObjectID |
| `errors.As` / `IsNotFound` | Map domain errors to HTTP |
| No `mongo.Collection` in handler | Layer rule |

### Temporary stub for task.go

So `go run` compiles while tasks are not migrated yet, replace `internal/handler/task.go` with:

```go
package handler
```

(You will replace this in Step 9.)

### File: `cmd/api/main.go` (projects only)

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/<you>/go-practice/internal/handler"
	"github.com/<you>/go-practice/internal/repository"
	"github.com/<you>/go-practice/internal/service"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	dbName := os.Getenv("MONGODB_DATABASE")
	if uri == "" || dbName == "" {
		log.Fatal("MONGODB_URI and MONGODB_DATABASE must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("connect:", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("ping:", err)
	}

	db := client.Database(dbName)
	projectRepo := repository.NewProjectRepo(db)
	projectSvc := service.NewProjectService(projectRepo)

	http.HandleFunc("GET /health", handler.HealthHandler)
	http.HandleFunc("POST /projects", handler.CreateProjectHandler(projectSvc))
	http.HandleFunc("GET /projects", handler.ListProjectsHandler(projectSvc))
	http.HandleFunc("GET /projects/{id}", handler.GetProjectHandler(projectSvc))
	http.HandleFunc("PATCH /projects/{id}", handler.UpdateProjectHandler(projectSvc))
	http.HandleFunc("DELETE /projects/{id}", handler.DeleteProjectHandler(projectSvc))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

**Why:** one client, env-based config, repo → service → handlers, prove projects before tasks.

### Activity 6.1 — Run and curl projects

```bash
# Terminal 1 (env vars set):
go run ./cmd/api

# Terminal 2:
curl -i -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"persists","description":"phase 2"}'

curl -s http://localhost:8080/projects
```

Expect `201` with a hex `"id"`, then a list containing that project.

### Activity 6.2 — Prove persistence (required gate)

1. Copy the project `id` from the create response.  
2. Ctrl+C the server.  
3. `go run ./cmd/api` again (same env).  
4. `curl -s http://localhost:8080/projects` — project still there.  
5. Optional mongosh:

```bash
docker exec -it desklog-mongo mongosh
```

```javascript
use desklog
db.projects.find().pretty()
```

**Do not continue until Activity 6.2 works.**

---

## Step 7 — Task repository

### File: `internal/repository/task.go` (full file)

```go
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/<you>/go-practice/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TaskRepo struct {
	col *mongo.Collection
}

func NewTaskRepo(db *mongo.Database) *TaskRepo {
	return &TaskRepo{col: db.Collection("tasks")}
}

func (r *TaskRepo) Create(ctx context.Context, t model.Task) (model.Task, error) {
	t.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, t)
	if err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id primitive.ObjectID) (model.Task, error) {
	var t model.Task
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task{}, ErrNotFound
		}
		return model.Task{}, err
	}
	return t, nil
}

func (r *TaskRepo) ListByProject(ctx context.Context, projectID primitive.ObjectID) ([]model.Task, error) {
	cursor, err := r.col.Find(ctx, bson.M{"project_id": projectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	tasks := make([]model.Task, 0)
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepo) Update(ctx context.Context, t model.Task) (model.Task, error) {
	t.UpdatedAt = time.Now().UTC()
	res, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": t.ID},
		bson.M{"$set": bson.M{
			"title":      t.Title,
			"status":     t.Status,
			"updated_at": t.UpdatedAt,
		}},
	)
	if err != nil {
		return model.Task{}, err
	}
	if res.MatchedCount == 0 {
		return model.Task{}, ErrNotFound
	}
	return t, nil
}

func (r *TaskRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TaskRepo) DeleteByProjectID(ctx context.Context, projectID primitive.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"project_id": projectID})
	return err
}
```

### Activity 7.1

```bash
go build ./internal/repository/
```

---

## Step 8 — Task service + cascade project delete

### File: `internal/service/task.go` (full file)

```go
package service

import (
	"context"
	"strings"

	"github.com/<you>/go-practice/internal/model"
	"github.com/<you>/go-practice/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskService struct {
	tasks    *repository.TaskRepo
	projects *repository.ProjectRepo
}

func NewTaskService(tasks *repository.TaskRepo, projects *repository.ProjectRepo) *TaskService {
	return &TaskService{tasks: tasks, projects: projects}
}

func validStatus(s string) bool {
	return s == "todo" || s == "doing" || s == "done"
}

func (s *TaskService) Create(ctx context.Context, projectID primitive.ObjectID, title, status string) (model.Task, error) {
	if _, err := s.projects.GetByID(ctx, projectID); err != nil {
		return model.Task{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Task{}, &ValidationError{Message: "title is required"}
	}
	if status == "" {
		status = "todo"
	}
	if !validStatus(status) {
		return model.Task{}, &ValidationError{Message: "status must be todo, doing, or done"}
	}
	return s.tasks.Create(ctx, model.Task{
		ProjectID: projectID,
		Title:     title,
		Status:    status,
	})
}

func (s *TaskService) ListByProject(ctx context.Context, projectID primitive.ObjectID) ([]model.Task, error) {
	if _, err := s.projects.GetByID(ctx, projectID); err != nil {
		return nil, err
	}
	return s.tasks.ListByProject(ctx, projectID)
}

func (s *TaskService) GetByID(ctx context.Context, id primitive.ObjectID) (model.Task, error) {
	return s.tasks.GetByID(ctx, id)
}

func (s *TaskService) Update(ctx context.Context, id primitive.ObjectID, title, status *string) (model.Task, error) {
	t, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if title != nil {
		n := strings.TrimSpace(*title)
		if n == "" {
			return model.Task{}, &ValidationError{Message: "title is required"}
		}
		t.Title = n
	}
	if status != nil {
		if !validStatus(*status) {
			return model.Task{}, &ValidationError{Message: "status must be todo, doing, or done"}
		}
		t.Status = *status
	}
	return s.tasks.Update(ctx, t)
}

func (s *TaskService) Delete(ctx context.Context, id primitive.ObjectID) error {
	return s.tasks.Delete(ctx, id)
}
```

### Replace `internal/service/project.go` for cascade

```go
package service

import (
	"context"
	"strings"

	"github.com/<you>/go-practice/internal/model"
	"github.com/<you>/go-practice/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProjectService struct {
	projects *repository.ProjectRepo
	tasks    *repository.TaskRepo
}

func NewProjectService(projects *repository.ProjectRepo, tasks *repository.TaskRepo) *ProjectService {
	return &ProjectService{projects: projects, tasks: tasks}
}

func (s *ProjectService) Create(ctx context.Context, name, description string) (model.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Project{}, &ValidationError{Message: "name is required"}
	}
	return s.projects.Create(ctx, model.Project{
		Name:        name,
		Description: description,
	})
}

func (s *ProjectService) List(ctx context.Context) ([]model.Project, error) {
	return s.projects.List(ctx)
}

func (s *ProjectService) GetByID(ctx context.Context, id primitive.ObjectID) (model.Project, error) {
	return s.projects.GetByID(ctx, id)
}

func (s *ProjectService) Update(ctx context.Context, id primitive.ObjectID, name, description *string) (model.Project, error) {
	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return model.Project{}, err
	}
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return model.Project{}, &ValidationError{Message: "name is required"}
		}
		p.Name = n
	}
	if description != nil {
		p.Description = *description
	}
	return s.projects.Update(ctx, p)
}

func (s *ProjectService) Delete(ctx context.Context, id primitive.ObjectID) error {
	if err := s.tasks.DeleteByProjectID(ctx, id); err != nil {
		return err
	}
	return s.projects.Delete(ctx, id)
}
```

**Why cascade in service:** MongoDB does not enforce foreign keys. Orphan tasks are a business bug.

### Activity 8.1

```bash
go build ./internal/service/
```

---

## Step 9 — Task handlers + full wiring

### File: `internal/handler/task.go` (full file)

```go
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/<you>/go-practice/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ListTasksByProjectHandler(svc *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		tasks, err := svc.ListByProject(r.Context(), projectID)
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tasks)
	}
}

func CreateTaskHandler(svc *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		task, err := svc.Create(r.Context(), projectID, req.Title, req.Status)
		var ve *service.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, ve.Message)
			return
		}
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(task)
	}
}

func GetTaskHandler(svc *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		task, err := svc.GetByID(r.Context(), id)
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}
}

func UpdateTaskHandler(svc *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req struct {
			Title  *string `json:"title"`
			Status *string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		task, err := svc.Update(r.Context(), id, req.Title, req.Status)
		var ve *service.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, ve.Message)
			return
		}
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}
}

func DeleteTaskHandler(svc *service.TaskService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := primitive.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		err = svc.Delete(r.Context(), id)
		if service.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
```

### Activity 9.1 — Update `main` wiring

Replace the service construction block with:

```go
projectRepo := repository.NewProjectRepo(db)
taskRepo := repository.NewTaskRepo(db)
projectSvc := service.NewProjectService(projectRepo, taskRepo)
taskSvc := service.NewTaskService(taskRepo, projectRepo)

http.HandleFunc("GET /health", handler.HealthHandler)
http.HandleFunc("POST /projects", handler.CreateProjectHandler(projectSvc))
http.HandleFunc("GET /projects", handler.ListProjectsHandler(projectSvc))
http.HandleFunc("GET /projects/{id}", handler.GetProjectHandler(projectSvc))
http.HandleFunc("PATCH /projects/{id}", handler.UpdateProjectHandler(projectSvc))
http.HandleFunc("DELETE /projects/{id}", handler.DeleteProjectHandler(projectSvc))
http.HandleFunc("GET /projects/{id}/tasks", handler.ListTasksByProjectHandler(taskSvc))
http.HandleFunc("POST /projects/{id}/tasks", handler.CreateTaskHandler(taskSvc))
http.HandleFunc("GET /tasks/{id}", handler.GetTaskHandler(taskSvc))
http.HandleFunc("PATCH /tasks/{id}", handler.UpdateTaskHandler(taskSvc))
http.HandleFunc("DELETE /tasks/{id}", handler.DeleteTaskHandler(taskSvc))
```

### Activity 9.2 — Curl tasks

```bash
# After creating a project, set ID=...
curl -i -X POST http://localhost:8080/projects/$ID/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"first task"}'

curl -s http://localhost:8080/projects/$ID/tasks
```

### Activity 9.3 — Cascade

Create a project + tasks, `DELETE` the project, confirm tasks are gone in mongosh:

```javascript
use desklog
db.tasks.find().pretty()
```

---

## Step 10 — Indexes

### File: `scripts/indexes.js` (full file)

```javascript
db = db.getSiblingDB('desklog');

db.projects.createIndex({ name: 1 });
db.tasks.createIndex({ project_id: 1 });
db.tasks.createIndex({ project_id: 1, status: 1 });
```

### Activity 10.1 — Apply indexes

```bash
docker exec -i desklog-mongo mongosh < scripts/indexes.js
```

**Why:** listing tasks by `project_id` and cascade deletes stay fast as data grows.

---

## Step 11 — Cleanup, README, smoke test

### Activity 11.1 — Remove in-memory store

- Delete `internal/store/` once nothing imports it.
- Confirm handlers do **not** use `mongo.Collection` / `InsertOne` (ObjectID parse only is fine).

### Activity 11.2 — README

Document:

1. Start Mongo (`docker run ...`)  
2. Required env vars  
3. `go run ./cmd/api`  
4. Data survives restart  

### Activity 11.3 — Full smoke test

```bash
curl -i http://localhost:8080/health
# POST project → GET list → POST task → GET task → PATCH → DELETE project
# restart server → GET projects (data still there)
```

---

## Recipe reminder (Phase 2 edition)

```text
1. Contract   — same as Phase 1
2. Model      — BSON + ObjectID
3. Repository — Insert/Find/Update/Delete
4. Service    — validation + “project must exist” + cascade
5. Handler    — parse hex id, call service, map errors
6. Wire       — client → repos → services → HandleFunc
7. Verify     — curl + restart + mongosh
```

---

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Mongo calls in handler | Move to repository; handler calls service only |
| New `mongo.Connect` per request | One client in `main` |
| Ignore `context` | Pass `r.Context()` into service/repo |
| String `_id` in BSON | Use `primitive.ObjectID` |
| `ErrNoDocuments` → 500 | Map to `ErrNotFound` → 404 |
| Delete project, leave tasks | `DeleteByProjectID` in service before deleting project |
| Forget to restart after code change | Ctrl+C, then `go run` again |
| Env vars not set in the server terminal | Export in the same terminal as `go run` |

---

## Exit checklist

- [ ] Completed Steps 1–11 activities  
- [ ] MongoDB running via Docker; env vars documented  
- [ ] Layout: `handler` → `service` → `repository` → Mongo  
- [ ] All Phase 1 project/task endpoints work against Mongo  
- [ ] Data survives server restart  
- [ ] Delete project removes its tasks  
- [ ] Indexes applied  
- [ ] `internal/store` removed or unused  
- [ ] README explains Mongo + env + run  

**Commit:** `feat(phase-2): mongodb persistence and layered architecture`

---

**Always:** [The endpoint recipe](./the-endpoint-recipe.md)

**Next:** [Phase 3 Manual — Auth, Validation & Tests](./phase-03-auth-validation-tests.md)
