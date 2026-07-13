# Desklog — Phase Manuals

Self-contained manuals for each phase of [PROJECT.md](../PROJECT.md). Each document explains concepts inline, then tells you **what to build in recipe order**.

**Always keep open:** [The endpoint recipe](./the-endpoint-recipe.md) — the same 7 steps for every new API capability.

Read your current phase manual top to bottom. Complete build steps in order. Pass the exit checklist before moving on.

| Phase | Manual | What it covers |
|-------|--------|----------------|
| — | [The endpoint recipe](./the-endpoint-recipe.md) | Contract → model → data → rules → handler → wire → verify |
| 1 | [HTTP & Go basics](./phase-01-http-and-go-basics.md) | Beginner build steps with full files + why |
| 2 | [MongoDB & structure](./phase-02-mongodb-and-structure.md) | Layers, BSON, driver CRUD, indexes |
| 3 | [Auth, validation & tests](./phase-03-auth-validation-tests.md) | bcrypt, JWT, middleware, scoping, tests |
| 4 | [Time entries & reporting](./phase-04-time-entries-reporting.md) | Nested routes, aggregation, date ranges |
| 5 | [Concurrency & resilience](./phase-05-concurrency-resilience.md) | Shutdown, timeouts, workers |
| 6 | [Ship it](./phase-06-ship-it.md) | Docker, compose, CI, readiness, portfolio README |
| Capstone | [Final quiz — new contract](./final-quiz.md) | Implement project notes using only the recipe |

## How to use a manual

1. Skim section headers once so you know the arc.
2. For Phase 1: do **Part B build steps** in order; run the check at the end of each step.
3. For Phases 2–6: learn the concepts, then follow that phase’s **build order (recipe order)** table.
4. Before any new endpoint, fill the recipe card (see the recipe doc).
5. Commit at the exit checkpoint — one commit per phase keeps history readable.
6. After Phase 6, take the [final quiz](./final-quiz.md).

## Start here

1. [The endpoint recipe](./the-endpoint-recipe.md)  
2. [Phase 1 Manual — HTTP & Go basics](./phase-01-http-and-go-basics.md)
