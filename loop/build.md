# Build Report — Reputation Scores

## Gap
No reputation system. Profile shows actions but no quality signal. Market cards have no author trust indicator.

## Changes

### graph/store.go
- **migrate()**: Added `ALTER TABLE users ADD COLUMN IF NOT EXISTS reputation_score INT NOT NULL DEFAULT 0`
- **GetUserProfile()**: Added `ReputationScore int` field; SELECT now includes `u.reputation_score`
- **ComputeAndUpdateReputation(ctx, userID)**: New method. Counts assignee-based completed tasks, review verdicts (approve/revise/reject) on those tasks, and endorsements. Formula: `tasks×1 + approvals×2 + revisions×0.5 + endorsements×1.5 - rejections×1`. Stores result in `users.reputation_score`.
- **GetReputationComponents(ctx, userID)**: New method. Returns `(tasksCompleted, reviewApprovals int)` for the "X tasks completed, Y approved" profile display.
- **GetBulkReputationByIDs(ctx, userIDs)**: New method. Bulk-fetches `reputation_score` by user ID for market card display.
- **ListAvailableTasks scan**: Fixed pre-existing bug where `n.verdict` and `n.rating` were selected but not scanned (column count mismatch). Added `&n.Verdict` and `&n.Rating` to Scan.

### graph/handlers.go
- **complete op**: After marking task done, calls `ComputeAndUpdateReputation(ctx, actorID)`.
- **review op**: After applying verdict, calls `ComputeAndUpdateReputation(ctx, node.AssigneeID)`.

### cmd/site/main.go
- **endorse handler**: After endorse/unendorse, calls `ComputeAndUpdateReputation(ctx, targetID)`.
- **profile handler**: Calls `GetReputationComponents` and passes `ReputationScore`, `TasksCompleted`, `ReviewApprovals` to `UserProfile`.
- **market handler**: Collects unique `author_id` values, bulk-fetches reputation via `GetBulkReputationByIDs`, passes `AuthorReputation` to each `MarketTask`.

### views/profile.templ
- Added `ReputationScore`, `TasksCompleted`, `ReviewApprovals` to `UserProfile` struct.
- New reputation display block (between endorse section and spaces): "Rep N" badge + "X tasks completed, Y approved". Hidden when score=0 and no completed tasks.

### views/market.templ
- Added `AuthorReputation int` to `MarketTask` struct.
- `marketCard`: shows `rep N` badge next to author name when `AuthorReputation > 0`.

## Verification
- `templ generate`: ✓ 13 updates
- `go.exe build -buildvcs=false ./...`: ✓ no errors
- `go.exe test ./...`: ✓ all pass (graph: 0.544s, auth: cached)
