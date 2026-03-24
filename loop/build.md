# Build Report — Space Member Welcome Experience

## Gap
When users join an existing space via invite, they had no orientation — no welcome, no context about the space, no agent introduction, no member roster, no clear next actions.

## Changes

### `graph/store.go`
- **Schema migration**: `ALTER TABLE space_members ADD COLUMN IF NOT EXISTS welcomed_at TIMESTAMPTZ` — tracks first visit per user per space
- **`MarkWelcomed(ctx, spaceID, userID string) bool`** — sets `welcomed_at = NOW()` where it was NULL; returns true only the first time (idempotent via UPDATE + rowsAffected check)

### `graph/handlers.go`
- **`handleJoinViaInvite`**: Changed redirect from `/app/{slug}` to `/app/{slug}/board` — so the board handler can check and mark first visit immediately on join
- **`handleBoard`**: Added welcome check — if authenticated non-owner member has `welcomed_at IS NULL`, calls `MarkWelcomed` (marks + returns true), loads member list, sets `showWelcome = true`; updated `BoardView` call with two new params

### `graph/views.templ`
- **`spaceWelcomeModal(space Space, members []SpaceMember)`** — new modal component:
  - Space name + description
  - Agent intro card (shown for any member with `kind == "agent"`)
  - Human member roster with green dots
  - Three action buttons: "Create a task" (primary), "Chat", "Settings"
  - JS `dismissSpaceWelcome()` — removes modal from DOM; no localStorage needed (shown once via DB tracking)
- **`BoardView`** — two new params: `showWelcome bool, welcomeMembers []SpaceMember`; renders `spaceWelcomeModal` before other overlays when `showWelcome` is true

## Verification
- `templ generate` — 13 updates, no errors
- `go.exe build -buildvcs=false ./...` — clean
- `go.exe test ./...` — all pass (`graph` 0.515s, `auth` cached)

## Design notes
- Welcome shows exactly once per user per space (DB-backed, not cookie/localStorage)
- Non-owner check prevents space creators from seeing their own welcome
- Modal renders above checklist and celebration ceremony in z-order
