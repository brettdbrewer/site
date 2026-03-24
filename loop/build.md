# Build Report

## Gap
Task cards on the Board showed agent assignment but gave no inline feedback on completion status or agent response — users had to click through to details to see what the agent did.

## Changes

### `graph/views.templ` — `TaskCard` component

1. **Done card styling**: Cards in `StateDone` get `border-emerald-500/20 opacity-75` instead of the active hover styles, making completed tasks visually recede.

2. **Done checkmark icon**: When `State == StateDone`, the priority dot is replaced with a green checkmark SVG, making completion immediately scannable.

3. **Title strikethrough**: Done task titles get `line-through text-warm-muted` — the standard completion visual.

4. **Body snippet**: When `node.Body != ""`, a truncated (120 char) body preview appears under the title with `text-[11px] text-warm-faint line-clamp-2`. This is the agent's work summary / response text.

5. **Agent done indicator**: When `AssigneeKind == "agent"` and `State == StateDone`, the "Thinking" dots are replaced with a green "✓ Agent done" label. When still in progress, "Thinking" dots remain.

## Verification
- `templ generate` — 13 updates, no errors
- `go.exe build -buildvcs=false ./...` — clean
- `go.exe test ./...` — all pass (graph: 0.538s)
