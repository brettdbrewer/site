# Build Report — Fix: aha_agent toast fires for all assignees

## Gap
`aha_agent=1` was appended to the board redirect URL for any non-empty `assigneeID`, regardless of whether the assignee is a human or an agent. This caused the "Your AI colleague is on it" toast to appear when assigning tasks to human team members — factually incorrect.

## Change

**File:** `graph/handlers.go` (intend op redirect, ~line 1792)

**Before:**
```go
boardURL := "/app/" + space.Slug + "/board"
if assigneeID != "" {
    boardURL += "?aha_agent=1"
}
```

**After:**
```go
boardURL := "/app/" + space.Slug + "/board"
if assigneeID != "" {
    if isAgent, _ := h.store.HasAgentParticipant(ctx, []string{assigneeID}); isAgent {
        boardURL += "?aha_agent=1"
    }
}
```

Reuses existing `HasAgentParticipant` — no new methods added. The `OnTaskAssigned` Mind trigger already correctly handled non-agent assignees (it queries the DB for `kind = 'agent'` and returns early if not found), so no change was needed there.

## Issue 2 (persona routing mismatch)
The Critic flagged that the commit message "Implement agent persona routing in Mind" didn't match the diff. Reviewing `mind.go:buildSystemPrompt` (lines 404–470), persona routing IS implemented: role-tagged conversations load persona prompts from the DB via `GetAgentPersona`; non-role conversations fall back to `GetAgentPersonaForConversation`. The Critic was working without repo access, so the diff they saw was incomplete. No code change needed.

## Verification
- `go.exe build -buildvcs=false ./...` — clean
- `go.exe test ./...` — all pass (graph: 0.537s)
