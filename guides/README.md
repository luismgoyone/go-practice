# Desklog — Phase Manuals

Self-contained manuals for each phase of [PROJECT.md](../PROJECT.md). Each document explains every concept inline — you should not need to leave the manual to understand what to do and why.

Read your current phase manual top to bottom. Complete sections in order. Pass the exit checklist before moving on.

| Phase | Manual | What it covers |
|-------|--------|----------------|
| 1 | [HTTP & Go basics](./phase-01-http-and-go-basics.md) | HTTP, JSON, structs, handlers, in-memory store |
| 2 | [MongoDB & structure](./phase-02-mongodb-and-structure.md) | Layers, BSON, driver CRUD, indexes, cascade delete |
| 3 | [Auth, validation & tests](./phase-03-auth-validation-tests.md) | bcrypt, JWT, middleware, scoping, slog, testing |
| 4 | [Time entries & reporting](./phase-04-time-entries-reporting.md) | Nested routes, aggregation, date ranges |
| 5 | [Concurrency & resilience](./phase-05-concurrency-resilience.md) | Shutdown, timeouts, workers, job queue |
| 6 | [Ship it](./phase-06-ship-it.md) | Docker, compose, CI, readiness, portfolio README |

## How to use a manual

1. **Read the whole phase once** — skim section headers so you know the arc.
2. **Work section by section** — each section teaches a concept, then tells you what to build.
3. **Verify as you go** — curl commands and checklists are at the end of each phase.
4. **Commit at the exit checkpoint** — one commit per phase keeps history readable.

## Start here

[Phase 1 Manual — HTTP & Go basics](./phase-01-http-and-go-basics.md)
