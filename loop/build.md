# Build Report — Fix: invite management UI (test coverage)

## Gap
Critic review of commit 32b3763 flagged three issues:
1. `ListInvites` missing LIMIT (BOUNDED invariant 13)
2. Dead `POST /app/{slug}/invite` endpoint (old singular route)
3. No tests for `ListInvites`, `RevokeInvite`, `handleCreateInviteHTMX`, `handleRevokeInvite`

Issues 1 and 2 were already resolved in commit de4636a. This build addresses issue 3.

## Changes

### `graph/store_test.go`
Added `TestListInvitesAndRevoke` with three sub-tests:
- `empty_list` — verifies `ListInvites` returns empty slice for a fresh space
- `lists_created_invites` — creates two invites, verifies both appear in the list
- `revoke_removes_invite` — creates an invite, revokes it, verifies `GetInviteCode` returns nil

### `graph/handlers_test.go`
Added `TestHandlerCreateInviteHTMX` with two sub-tests:
- `owner_creates_invite_returns_html` — POST creates invite, returns 200 + HTML fragment, verifies store has the invite
- `nonexistent_space_404` — POST to unknown slug returns 404

Added `TestHandlerRevokeInvite` with two sub-tests:
- `revoke_existing_invite` — DELETE removes token, returns 200, verifies store deletion
- `revoke_nonexistent_token_404` — DELETE with unknown token returns 404

## Verification
- `go.exe build -buildvcs=false ./...` — clean
- `go.exe test ./...` — all pass (DB-dependent tests skip cleanly without DATABASE_URL, consistent with existing pattern)
