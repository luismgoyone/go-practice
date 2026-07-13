# Phase 3 Manual — Auth, Validation & Tests

**Duration:** 2–3 weeks  
**Prerequisite:** Phase 2 complete

Desklog until now has no concept of "who" is calling the API. Anyone can read and write all projects. Phase 3 adds **users**, **authentication** (proving identity), **authorization** (proving permission), **input validation**, **structured logging**, and **automated tests**.

---

## 0. What changes in this phase

| Before | After |
|--------|-------|
| Open API | Login required for data endpoints |
| All projects visible | Each user sees only their projects |
| Ad-hoc validation | Consistent validation errors |
| `log.Printf` | Structured JSON logs |
| No tests | `go test ./...` passes |

**Same habit:** [The endpoint recipe](./the-endpoint-recipe.md). Auth adds middleware and scoping; it does not replace the recipe.

| Recipe step | What changes in Phase 3 |
|-------------|-------------------------|
| Contract | New: `POST /auth/register`, `POST /auth/login`; data routes require `Authorization: Bearer …` |
| Model | `User`; projects/tasks gain `user_id` |
| Data | User repository; all project/task queries filter by `user_id` |
| Business | Hash passwords, issue/validate JWT, ownership checks in service |
| Handler | Auth handlers; data handlers read `user_id` from context (never from body) |
| Wire | Auth middleware wraps protected routes in `main` |
| Verify | curl with token **and** `go test ./...` |

For a **new protected endpoint**: contract → model/repo/service as needed → handler → register **behind** auth middleware → curl with Bearer token → add a test if it encodes a security rule.

---

## 1. Authentication vs authorization

### Authentication (authn) — "Who are you?"

The client proves identity, usually by:

- Sending a **token** (JWT in `Authorization` header) obtained from login
- Or sending a **session cookie** from a previous login

Desklog uses **JWT Bearer tokens** — common for APIs, no session store required in Phase 3.

### Authorization (authz) — "Are you allowed?"

After identity is known, every data operation checks:

- Does this **project** belong to the current user?
- Does this **task** belong to a project owned by the current user?

**Critical rule:** `user_id` comes from the **token**, never from the request body. A client sending `{"user_id": "someone-else"}` must be ignored.

### Flow overview

```
1. POST /auth/register  → create user (hashed password)
2. POST /auth/login     → verify password → return JWT
3. Client sends: Authorization: Bearer <jwt>
4. Middleware validates JWT → extracts user_id → puts in context
5. Handler/service uses user_id from context for all queries
```

---

## 2. Password storage

### Never store plaintext passwords

If your database leaks, plaintext passwords compromise users everywhere they reuse passwords. Store only a **hash** — a one-way transformation.

### bcrypt

**bcrypt** is a password hashing algorithm designed to be slow (resistant to brute force).

```bash
go get golang.org/x/crypto/bcrypt
```

**Register — hash before insert:**

```go
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// store string(hash) in user.PasswordHash
```

`DefaultCost` is 10 — higher = slower = more secure but more CPU.

**Login — compare:**

```go
err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
if err != nil {
	// wrong password — same response as user not found
	return ErrInvalidCredentials
}
```

### User model

```go
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}
```

`json:"-"` means this field is **never** included in JSON output — even if you accidentally try to encode the full struct.

### Login error messages

Return the same error for "email not found" and "wrong password":

```json
{"error": "invalid credentials"}
```

Different messages let attackers discover which emails are registered (**user enumeration**).

### Unique email index

```javascript
db.users.createIndex({ email: 1 }, { unique: true });
```

Duplicate registration → MongoDB error 11000 → HTTP **409 Conflict**.

---

## 3. JWT (JSON Web Token)

### What a JWT is

A JWT is a signed string with three parts (header.payload.signature), dot-separated.

The **payload** contains **claims** — key/value pairs:

```json
{
  "sub": "507f1f77bcf86cd799439011",
  "exp": 1720000000
}
```

- `sub` — subject (user ID)
- `exp` — expiration (Unix timestamp)

The server **signs** the token with a secret. Clients cannot forge tokens without the secret.

### Install

```bash
go get github.com/golang-jwt/jwt/v5
```

### Issue token on login

```go
secret := []byte(os.Getenv("JWT_SECRET"))

token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	"sub": user.ID.Hex(),
	"exp": time.Now().Add(24 * time.Hour).Unix(),
	"iat": time.Now().Unix(),
})

signed, err := token.SignedString(secret)
```

Return:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-07-11T12:00:00Z"
}
```

### Validate token in middleware

```go
token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
	if t.Method != jwt.SigningMethodHS256 {
		return nil, fmt.Errorf("unexpected method")
	}
	return secret, nil
})
if err != nil || !token.Valid {
	// 401
}

claims, ok := token.Claims.(jwt.MapClaims)
sub, _ := claims["sub"].(string)
userID, err := primitive.ObjectIDFromHex(sub)
```

### JWT_SECRET

- Long random string (32+ bytes)
- From environment only: `JWT_SECRET=...`
- Never commit to git
- App refuses to start if missing

### JWT limitations (know this)

- **Hard to revoke** before expiry — acceptable for learning
- **Do not put sensitive data** in payload (it's base64, not encrypted)
- Phase 3 stretch: denylist collection for logout

---

## 4. Auth middleware

### What middleware does

Runs **before** your handler on every protected request:

1. Read `Authorization` header
2. Expect format: `Bearer <token>`
3. Validate JWT
4. Put `userID` in request context
5. Call next handler

```go
type contextKey string
const userIDKey contextKey = "userID"

func AuthMiddleware(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		userID, err := validateToken(tokenStr, secret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

Use a custom `contextKey` type (not a string constant alone) to avoid collisions with other packages.

### Helper to read user ID in handlers

```go
func UserIDFromContext(ctx context.Context) (primitive.ObjectID, error) {
	id, ok := ctx.Value(userIDKey).(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("no user in context")
	}
	return id, nil
}
```

### Route groups

**Public** (no middleware):

- `GET /health`
- `POST /auth/register`
- `POST /auth/login`

**Protected** (wrap with `AuthMiddleware`):

- All project and task routes

With Go 1.22 `ServeMux`, you can mount a sub-mux or wrap individual handlers.

---

## 5. Multi-tenant data scoping

### Add user_id to projects

```go
type Project struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	Name        string             `bson:"name" json:"name"`
	// ...
}
```

Index: `db.projects.createIndex({ user_id: 1 })`

### Every query filters by user

**List projects:**

```go
filter := bson.M{"user_id": userID}
cursor, err := col.Find(ctx, filter)
```

**Get project by ID:**

```go
filter := bson.M{"_id": id, "user_id": userID}
err := col.FindOne(ctx, filter).Decode(&p)
```

If not found — return `ErrNotFound` → 404. This covers both "ID does not exist" and "exists but belongs to another user." Hiding existence of other users' data is a deliberate security choice.

**Create project:**

```go
p.UserID = userID  // from context, not request body
```

### Task authorization through project

Before returning or modifying a task:

1. Load task
2. Load its project
3. Verify `project.UserID == currentUserID`

Or query with a join/filter in repository. Never trust `project_id` from body without verifying ownership.

---

## 6. Validation

### Why validate in the service layer

Handlers parse HTTP. Services enforce **business rules**. Same rules apply whether the caller is HTTP, a CLI, or a test.

### Validation rules for Desklog

| Field | Rule |
|-------|------|
| email | contains `@`, length 3–254 |
| password | minimum 8 characters |
| project name | non-empty, max 100 chars |
| task title | non-empty, max 200 chars |
| task status | exactly `todo`, `doing`, or `done` |

### Validation error type

```go
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
```

### HTTP response shape

```json
{
  "error": "validation failed",
  "fields": [
    {"field": "email", "message": "invalid format"},
    {"field": "password", "message": "must be at least 8 characters"}
  ]
}
```

Status: **400 Bad Request**

Collect multiple field errors on register rather than failing on the first one — better UX.

---

## 7. Structured logging with slog

### Why not log.Printf

`log.Printf("user %s did %s", id, action)` is hard to search in production log systems. **Structured logs** are key-value pairs, usually JSON:

```json
{"level":"INFO","msg":"request","method":"GET","path":"/projects","status":200,"duration_ms":12}
```

### Setup

```go
import "log/slog"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
}
```

### Usage

```go
slog.Info("request completed",
	"method", r.Method,
	"path", r.URL.Path,
	"status", statusCode,
	"duration_ms", elapsed.Milliseconds(),
)

slog.Error("database error", "err", err)
```

### Request ID middleware

Generate UUID per request, add to context and all log lines:

```go
requestID := uuid.New().String()
ctx := context.WithValue(r.Context(), requestIDKey, requestID)
slog.Info("request started", "request_id", requestID)
```

### Never log

- Passwords
- JWT tokens
- Password hashes

---

## 8. Testing

### Why tests matter here

Auth and scoping bugs are **IDOR vulnerabilities** (Insecure Direct Object Reference) — User A accesses User B's data by guessing IDs. Tests catch regressions.

### Three levels

| Level | What it tests | Needs real DB? |
|-------|---------------|----------------|
| Unit (service) | Business rules with fake repos | No |
| Handler (`httptest`) | HTTP status, headers, body | No (mock service) |
| Integration | Full stack | Yes (optional) |

### Table-driven tests

Go idiom for multiple cases:

```go
func TestCreateTask_Validation(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"empty title", "", true},
		{"valid", "Write tests", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), projectID, tt.title, "todo")
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

### Fake repository

```go
type fakeProjectRepo struct {
	project model.Project
	err     error
}

func (f *fakeProjectRepo) GetByID(ctx context.Context, id primitive.ObjectID) (model.Project, error) {
	return f.project, f.err
}
```

Service tests use fakes — fast, no MongoDB.

### httptest — test handlers without network

```go
func TestListProjects_Unauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	// no Authorization header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}
}
```

### Minimum tests for exit

- Service: create task when project missing → error
- Service: invalid status → validation error
- Handler: `GET /projects` without token → 401
- Handler: `GET /projects` with valid token → 200
- Handler: access another user's project ID → 404 (or 403)

Run: `go test ./...`

---

## 9. Auth endpoints

Apply the recipe for each of these the same way you did for projects: **contract first**, then model → repo → service → handler → wire → verify.

### POST /auth/register

Request:

```json
{"email": "you@example.com", "password": "securepass123"}
```

Response `201`:

```json
{"id": "...", "email": "you@example.com", "created_at": "..."}
```

Errors: `400` validation, `409` duplicate email

### POST /auth/login

Request: same shape  
Response `200`:

```json
{"token": "eyJ...", "expires_at": "..."}
```

Errors: `401` invalid credentials (always same message)

### Example session

```bash
# Register
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"securepass123"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"securepass123"}' \
  | jq -r .token)

# Authenticated request
curl -s http://localhost:8080/projects \
  -H "Authorization: Bearer $TOKEN"
```

---

## 10. Build order (recipe order)

| Step | Recipe | What to build |
|------|--------|---------------|
| 1 | Model + data | User model + repository |
| 2 | Business + handler + wire | Register + login (no middleware yet); curl both |
| 3 | Business | JWT issue + validate helpers |
| 4 | Wire | Auth middleware; protect one route; curl 401 then 200 |
| 5 | Model + data | Add `user_id` to projects (wipe dev data if needed) |
| 6 | Data + business | Scope all project/task queries by user from context |
| 7 | Business | Consistent validation errors in services |
| 8 | Cross-cutting | Replace logging with `slog` |
| 9 | Verify | Tests listed above; `go test ./...` |
| 10 | Docs | README auth flow |

Do not “add auth everywhere” before register/login work in isolation.

---

## 11. Common mistakes

| Mistake | Risk |
|---------|------|
| Plaintext passwords | Catastrophic on DB leak |
| `user_id` from request body | Full auth bypass |
| Different login errors per case | User enumeration |
| Logging JWTs | Token theft from logs |
| No tests on auth | IDOR ships to production |
| 403 for missing + wrong owner | Leaks that resource exists — 404 is safer for reads |

---

## 12. Exit checklist

- [ ] Register and login work
- [ ] JWT on all data endpoints
- [ ] Projects scoped by `user_id`
- [ ] Validation on all writes
- [ ] Structured logging
- [ ] `go test ./...` passes
- [ ] README documents auth flow

**Commit:** `feat(phase-3): user auth, validation, and tests`

---

**Always:** [The endpoint recipe](./the-endpoint-recipe.md)

**Next:** [Phase 4 Manual — Time Entries & Reporting](./phase-04-time-entries-reporting.md)
