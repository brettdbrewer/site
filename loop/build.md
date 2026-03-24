# Build Report

## Gap
Agent-assigned tasks showed no visual feedback. Users couldn't tell the agent was working — missing the "aha moment" of proof-of-life.

## Changes

### `graph/store.go`
- Added `AssigneeKind string` field to `Node` struct (resolved from users table, "human" or "agent")
- Updated `GetNode` query: `LEFT JOIN users au ON au.id = n.assignee_id`, added `COALESCE(au.kind, '')` to SELECT, added `&n.AssigneeKind` to Scan
- Updated `ListNodes` query: same JOIN and extra column, `&n.AssigneeKind` to Scan

### `graph/views.templ` — `TaskCard` template
- Added compact "Thinking" indicator: shown when `node.AssigneeKind == "agent" && node.State != StateDone`
- Three animated bouncing dots in violet (`bg-violet-400/60 animate-bounce`) matching the chat.templ thinking indicator palette
- Placed in the bottom metadata row alongside blocker badge and state select

## Verification
- `templ generate` ✓ (13 updates)
- `go build -buildvcs=false ./...` ✓
- `go test ./...` ✓ (graph 0.539s, auth cached)
