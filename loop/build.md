# Build: Fix t.Skipf → t.Fatalf in TestBuildSystemPromptInjectsMemories

## Gap
`TestBuildSystemPromptInjectsMemories` used `t.Skipf` after a live DB connection was already established. A schema failure at that point is real breakage, not a missing environment — silently skipping gives false green in CI and violates invariant 12 (VERIFIED).

## Change

**`graph/memory_test.go:92`**
```go
// Before
t.Skipf("agent_personas insert failed (schema may differ): %v", err)

// After
t.Fatalf("agent_personas insert failed: %v", err)
```

## Verification

- `go.exe build -buildvcs=false ./...` — passes clean
- `go.exe test ./graph/... -run TestBuildSystemPromptInjectsMemories -v` — skips at `testDB(t)` when `DATABASE_URL` is absent (correct behavior; `testDB` owns the env check), does not skip at the schema insert
