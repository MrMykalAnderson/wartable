# Wartable

A deterministic hex wargame system (no dice; secret simultaneous orders executed alternately), plus a Go engine and web tools for exploring its rules.

- Local path: `~/Projects/wartable`. Run `git pull` before starting work.
- Read `docs/dev-plan.md` first: it sets the architecture, milestones and working rules.
- Game rules: `docs/core-rules.md`. MVP scenario: `docs/starter-battle.md`. These are the source of truth; never invent rules silently (see dev plan section 1).
- Every number lives in `data/` and must match the rules docs.
- Every worked example (`EX-n`) in the core rules is a test.
- Before committing: `go vet ./...` and `go test ./...`.
