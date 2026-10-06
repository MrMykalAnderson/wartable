# Changelog

## M1: `hex` package

Implemented `internal/hex`: offset/cube coordinate conversion, neighbours,
distance, firing arcs, edge-attacked (front/flank/rear), knockback, and
pathfinding (BFS distance field with clockwise tie-break, falling back to
the closest reachable hex per core-rules.md 7.2 when the destination is
unreachable). Table-driven tests cover the worked examples in core-rules.md
section 11 (EX-3, EX-4) and the direction table in section 2.1.

`go vet ./...` and `go test ./...` pass.
