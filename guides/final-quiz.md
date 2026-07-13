# Final quiz — ship a new contract

**When:** After Phase 6 (or after Phase 4 if you want an earlier challenge).  
**Rules:** Use [The endpoint recipe](./the-endpoint-recipe.md). Do **not** paste a tutorial solution. Fill the card first, then implement.

This quiz checks whether you know **where to start** and **what to write** when someone hands you a new API requirement.

---

## Your assignment

Add **project notes** to Desklog.

A note is a short text comment attached to a project (not a task). Users leave reminders like “waiting on design review.”

### Contract (given — implement exactly)

| Item | Spec |
|------|------|
| Create | `POST /projects/{id}/notes` |
| List | `GET /projects/{id}/notes` |
| Delete | `DELETE /notes/{id}` |
| Auth | Same as other data routes (JWT required from Phase 3+) |
| Create body | `{"body":"waiting on design review"}` |
| Create success | `201` + note JSON including `id`, `project_id`, `user_id`, `body`, `created_at` |
| List success | `200` + array of notes for that project (empty `[]` OK) |
| Delete success | `204` (or `200`) |
| Errors | `400` empty body / too long; `401` missing/invalid token; `404` project or note not found **or** not owned by current user |

**Validation:** `body` required, trimmed length 1–500 characters.

**Ownership:** Notes belong to the authenticated user. Listing/deleting another user’s note must not leak existence (prefer `404`). Creating a note requires the project to exist and belong to the current user.

---

## Step 0 — Fill the recipe card (required)

Copy and complete **before** coding:

```text
Contract:  (already above — restate methods/paths in your own words)

Model:     fields? BSON/JSON tags? file?
Data:      repository methods? indexes?
Business:  service checks? (project exists, owner, body length)
Handler:   function names? which file?
Wire:      HandleFunc lines? behind which middleware?
Verify:    curl commands + at least one automated test idea
```

If the card is blank, you are not allowed to open the editor yet.

---

## What “done” looks like

- [ ] Recipe card filled (keep it in a comment or `NOTES.md` if you want)  
- [ ] Model + repository + service + handlers + `main` wiring  
- [ ] curl: register/login → create project → create note → list → delete  
- [ ] curl: note on someone else’s project ID → `404`  
- [ ] curl: empty `body` → `400`  
- [ ] One test (handler or service) covering a security or validation rule  
- [ ] README row for the three note endpoints  

---

## Hints (only if stuck for >20 minutes)

- Follow the **same folder path** you used for time entries (Phase 4): nested create/list under the parent, flat delete by note id.  
- Phase 1-only learners: skip JWT; still do model → store → handler → wire → curl.  
- Do not put Mongo (or map) access in the handler if you are on Phase 2+.  

---

## Stretch contracts (pick one after notes work)

Fill a new recipe card for **one** of these, then implement:

1. `GET /projects/{id}/notes/count` → `{"count": N}` for the current user’s notes on that project  
2. `PATCH /notes/{id}` → update `body` only; bump no other ownership rules  
3. `GET /notes?from=&to=` → list **my** notes across projects in a created_at range (query params + validation)

---

## Self-grade

| Score | Meaning |
|-------|---------|
| You needed the full solution spelled out | Re-do Phase 1 Steps 5–8 and the recipe doc |
| You needed a few nudges but finished | Pass — normal |
| Card first, then clean PR-sized change | Pass with confidence — you can take new API tickets |

**Commit when finished:** `feat: add project notes API` (or similar Conventional Commit)
