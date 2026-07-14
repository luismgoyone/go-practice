# Phase 3 Manual — Auth, Validation & Tests

**Duration:** 2–3 weeks  
**Prerequisite:** Phase 2 complete (Mongo + layers working)  
**You need:** MongoDB running, `curl`, optionally `jq`  
**Audience:** Beginners — same style as Phases 1–2

**How to use this document**

1. Read **Part A** so auth words make sense.
2. Follow **Part B** steps in order. Each step has files + **why** + an **Activity**.
3. Keep [The endpoint recipe](./the-endpoint-recipe.md) open. Auth adds middleware and `user_id` scoping — it does not replace the recipe.
4. Prove register/login **before** protecting every route.

Replace `github.com/<you>/go-practice` with your module path.

---

# Part A — Concepts (read first)

## A0. What changes

| Before | After |
|--------|-------|
| Anyone can call data APIs | JWT required on projects/tasks |
| All users see all projects | Each user sees only theirs |
| `log.Printf` | Structured `slog` JSON logs |
| No automated tests | `go test ./...` passes |

| Recipe step | Phase 3 |
|-------------|---------|
| Contract | `POST /auth/register`, `POST /auth/login`; data routes need `Authorization: Bearer …` |
| Model | `User`; projects (and ownership path for tasks) gain `user_id` |
| Data | User repository; project/task queries filter by owner |
| Business | bcrypt, JWT issue/validate, ownership checks |
| Handler | Auth handlers; read `user_id` from **context**, never body |
| Wire | Auth middleware wraps protected routes |
| Verify | curl with token **and** `go test ./...` |

---

## A1. Authn vs authz

- **Authentication** — who are you? (login → JWT)
- **Authorization** — are you allowed? (does this project belong to you?)

```
register → login → Bearer token → middleware validates → user_id in context → service scopes queries
```

**Critical rule:** `user_id` comes from the token/context only. Ignore any `user_id` in JSON bodies.

---

## A2. Passwords and JWT (one page)

- Store **bcrypt hashes**, never plaintext (`json:"-"` on `PasswordHash`).
- Login: same error for “unknown email” and “wrong password” → `401` `invalid credentials` (stops email enumeration).
- JWT: signed string; claims include `sub` (user id) + `exp`.
- `JWT_SECRET` from env only; refuse to start if missing.

---

## A3. Middleware

Middleware runs **before** the handler: check `Authorization: Bearer <token>`, validate JWT, put `userID` on `context`, call next.

**Public:** `/health`, `/auth/register`, `/auth/login`  
**Protected:** all project and task routes

---

## A4. Scoping = multi-tenant filter

Every project query includes `user_id` of the current user.  
Missing **or** someone else’s ID → same `404` (do not leak that the resource exists).

Tasks: load task → load its project → project must belong to current user.

---

# Part B — Build steps

---

## Step 1 — Dependencies and env

### Activity 1.1 — Install packages

```bash
go get golang.org/x/crypto/bcrypt
go get github.com/golang-jwt/jwt/v5
go get github.com/google/uuid
go mod tidy
```

### Activity 1.2 — Env

```bash
export JWT_SECRET="$(openssl rand -hex 32)"
# keep MONGODB_URI / MONGODB_DATABASE from Phase 2
```

**Check:** `echo $JWT_SECRET` is non-empty. App will also require this at startup later.

---

## Step 2 — User model + unique email index

### File: `internal/model/user.go` (full file)

```go
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}
```

**Why:** `json:"-"` ensures the hash never appears in API responses.

### Activity 2.1 — Index

Add to `scripts/indexes.js` (or run once in mongosh):

```javascript
db = db.getSiblingDB('desklog');
db.users.createIndex({ email: 1 }, { unique: true });
```

```bash
docker exec -i desklog-mongo mongosh < scripts/indexes.js
```

**Why unique:** one account per email; duplicates → Mongo error `11000` → HTTP `409`.

---

## Step 3 — User repository

### File: `internal/repository/user.go` (full file)

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

type UserRepo struct {
	col *mongo.Collection
}

func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{col: db.Collection("users")}
}

func (r *UserRepo) Create(ctx context.Context, u model.User) (model.User, error) {
	u.ID = primitive.NewObjectID()
	u.CreatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, err
	}
	return u, nil
}
```

### File: `internal/repository/errors.go` — add

```go
var ErrDuplicate = errors.New("duplicate")
```

(Keep existing `ErrNotFound`.)

### Activity 3.1

```bash
go build ./internal/repository/
```

---

## Step 4 — Auth service (register + login, no middleware yet)

### File: `internal/service/auth.go` (full file)

```go
package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/<you>/go-practice/internal/model"
	"github.com/<you>/go-practice/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	users  *repository.UserRepo
	secret []byte
}

func NewAuthService(users *repository.UserRepo, secret []byte) *AuthService {
	return &AuthService{users: users, secret: secret}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (model.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !strings.Contains(email, "@") || len(email) < 3 || len(email) > 254 {
		return model.User{}, &ValidationError{Message: "invalid email"}
	}
	if len(password) < 8 {
		return model.User{}, &ValidationError{Message: "password must be at least 8 characters"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	return s.users.Create(ctx, model.User{
		Email:        email,
		PasswordHash: string(hash),
	})
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *AuthService) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials // same message whether missing or wrong
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	exp := time.Now().UTC().Add(24 * time.Hour)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID.Hex(),
		"exp": exp.Unix(),
		"iat": time.Now().UTC().Unix(),
	})
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: signed, ExpiresAt: exp}, nil
}

func (s *AuthService) ParseUserID(tokenStr string) (primitive.ObjectID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return primitive.NilObjectID, ErrInvalidCredentials
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return primitive.NilObjectID, ErrInvalidCredentials
	}
	sub, _ := claims["sub"].(string)
	id, err := primitive.ObjectIDFromHex(sub)
	if err != nil {
		return primitive.NilObjectID, ErrInvalidCredentials
	}
	return id, nil
}

// Optional helper used at startup
func MustJWTSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		panic("JWT_SECRET must be set")
	}
	return []byte(s)
}
```

**Why:** hash on register; same login error always; JWT `sub` = user id hex.

### Activity 4.1

```bash
go build ./internal/service/
```

---

## Step 5 — Auth handlers + public routes

### File: `internal/handler/auth.go` (full file)

```go
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/<you>/go-practice/internal/repository"
	"github.com/<you>/go-practice/internal/service"
)

func RegisterHandler(svc *service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		user, err := svc.Register(r.Context(), req.Email, req.Password)
		var ve *service.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, ve.Message)
			return
		}
		if errors.Is(err, repository.ErrDuplicate) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(user)
	}
}

func LoginHandler(svc *service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		res, err := svc.Login(r.Context(), req.Email, req.Password)
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if err != nil {
			log.Println(err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}
}
```

### Activity 5.1 — Wire public auth routes in `main`

Construct `AuthService`, register:

```go
http.HandleFunc("POST /auth/register", handler.RegisterHandler(authSvc))
http.HandleFunc("POST /auth/login", handler.LoginHandler(authSvc))
```

Keep existing project/task routes working for now (still open).

### Activity 5.2 — Curl register + login

```bash
curl -i -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"securepass123"}'

curl -i -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"securepass123"}'
```

Expect `201` then `200` with `token`. **Gate:** do not continue until this works.

---

## Step 6 — Auth middleware

### File: `internal/handler/auth_middleware.go` (full file)

```go
package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/<you>/go-practice/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type contextKey string

const userIDKey contextKey = "userID"

func AuthMiddleware(auth *service.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		userID, err := auth.ParseUserID(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (primitive.ObjectID, bool) {
	id, ok := ctx.Value(userIDKey).(primitive.ObjectID)
	return id, ok
}

// Protect wraps a HandlerFunc with auth middleware.
func Protect(auth *service.AuthService, h http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(auth, h).ServeHTTP
}
```

### Activity 6.1 — Protect one route first

```go
http.HandleFunc("GET /projects", handler.Protect(authSvc, handler.ListProjectsHandler(projectSvc)))
```

### Activity 6.2

```bash
curl -i http://localhost:8080/projects
# expect 401

TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"securepass123"}' | jq -r .token)

curl -i http://localhost:8080/projects -H "Authorization: Bearer $TOKEN"
# expect 200 (may be empty list)
```

**Gate:** 401 without token, 200 with token.

---

## Step 7 — Add `user_id` to projects and scope queries

### Activity 7.1 — Model

Add to `Project`:

```go
UserID primitive.ObjectID `bson:"user_id" json:"user_id"`
```

Wipe old project docs if needed (they have no `user_id`):

```javascript
use desklog
db.projects.deleteMany({})
db.tasks.deleteMany({})
```

Index: `db.projects.createIndex({ user_id: 1 })`

### Activity 7.2 — Repository filters

Every project list/get/update/delete filter must include `user_id`. Example get:

```go
err := r.col.FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&p)
```

Create sets `p.UserID = userID` in the **service** from context (handler passes userID into service methods).

### Activity 7.3 — Handlers/services

- Handler: `userID, ok := UserIDFromContext(r.Context())` → if !ok → 401  
- Service: `Create(ctx, userID, name, description)`, `List(ctx, userID)`, etc.  
- Tasks: before mutate/read, verify task’s project belongs to `userID`

### Activity 7.4 — Protect **all** project/task routes with `Protect`

Leave `/health`, `/auth/*` public.

### Activity 7.5 — Two-user test

Register user A and B. Create project as A. As B, `GET /projects/{A's id}` → **404**.

---

## Step 8 — Structured logging (`slog`)

### Activity 8.1 — In `main`

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
slog.SetDefault(logger)
```

### Activity 8.2 — Optional request-id middleware

Generate `uuid.NewString()`, put on context, log `"request_id"`.

**Never log:** passwords, JWTs, password hashes.

### Activity 8.3

Hit an endpoint; confirm JSON log lines in the server terminal.

---

## Step 9 — Tests

### Activity 9.1 — Service unit test (fake repo or real logic)

Minimum cases:

- create task with missing project → error  
- invalid task status → `ValidationError`

### Activity 9.2 — Handler test with `httptest`

```go
req := httptest.NewRequest(http.MethodGet, "/projects", nil)
rec := httptest.NewRecorder()
// call protected handler without Authorization
// expect 401
```

Also: valid token → 200; other user’s project id → 404.

### Activity 9.3

```bash
go test ./...
```

**Gate:** all tests green.

---

## Step 10 — README auth section

### Activity 10.1

Document register → login → `Authorization: Bearer` curl flow and that data is per-user.

---

## Common mistakes

| Mistake | Fix |
|---------|-----|
| `user_id` from body | Ignore body; use context |
| Different login errors | Always `invalid credentials` |
| Protect `/auth/login` | Keep auth routes public |
| Plaintext passwords | bcrypt only |
| 403 for other user’s resource | Prefer 404 |

---

## Exit checklist

- [ ] Register + login work  
- [ ] Data routes return 401 without token  
- [ ] Projects scoped by `user_id`  
- [ ] Other user’s IDs → 404  
- [ ] `slog` in use; no secrets in logs  
- [ ] `go test ./...` passes  
- [ ] README auth flow  

**Commit:** `feat(phase-3): user auth, validation, and tests`

---

**Always:** [The endpoint recipe](./the-endpoint-recipe.md)  
**Next:** [Phase 4 Manual — Time Entries & Reporting](./phase-04-time-entries-reporting.md)
