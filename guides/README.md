# Desklog — Phase Manuals

Self-contained manuals for each phase of [PROJECT.md](../PROJECT.md).

**Every phase uses the same beginner format:**

1. **Part A** — short concepts (read first)
2. **Part B** — numbered build steps with files, **why** notes, and **Activities** (do in order; finish each check before the next)

**Always keep open:** [The endpoint recipe](./the-endpoint-recipe.md) — the same 7 steps for every new API capability.

| Phase | Manual | What it covers |
|-------|--------|----------------|
| — | [The endpoint recipe](./the-endpoint-recipe.md) | Contract → model → data → rules → handler → wire → verify |
| 1 | [HTTP & Go basics](./phase-01-http-and-go-basics.md) | In-memory API (full beginner steps) |
| 2 | [MongoDB & structure](./phase-02-mongodb-and-structure.md) | Mongo → repository → service → migrate endpoints |
| 3 | [Auth, validation & tests](./phase-03-auth-validation-tests.md) | Users, bcrypt, JWT, scoping, slog, tests |
| 4 | [Time entries & reporting](./phase-04-time-entries-reporting.md) | Nested routes, aggregation, date ranges |
| 5 | [Concurrency & resilience](./phase-05-concurrency-resilience.md) | Shutdown, timeouts, workers |
| 6 | [Ship it](./phase-06-ship-it.md) | `/ready`, Docker, compose, CI, portfolio README |
| Capstone | [Final quiz — new contract](./final-quiz.md) | Implement project notes using only the recipe |

## How to use a manual

1. Skim Part A headers once.
2. Work **Part B step by step**; complete each **Activity** before moving on.
3. Before any new endpoint, fill the recipe card.
4. Commit at the exit checklist.
5. After Phase 6, take the [final quiz](./final-quiz.md).

## Start here

1. [The endpoint recipe](./the-endpoint-recipe.md)  
2. [Phase 1 Manual — HTTP & Go basics](./phase-01-http-and-go-basics.md)
