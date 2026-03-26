# Build Report — Tests for Knowledge surface

## Gap
Invariant VERIFIED: no code ships without tests. `ListDocuments`, `ListQuestions`, and `OnQuestionAsked` had no unit tests.

## Files Changed

### `graph/store_test.go`
Added `TestListDocuments` and `TestListQuestions`:
- Each seeds nodes of the correct kind plus one node of a different kind
- Asserts the list returns only the correct kind
- Asserts LIMIT enforcement (pass limit=2 with 3 nodes, expect ≤2 returned)

### `graph/handlers_test.go`
Added `TestHandlerKnowledgeTabs`:
- Creates a space, seeds a document and a question
- Exercises all four tab values: `docs`, `qa`, `claims`, and empty (default)
- Verifies HTTP 200 and no 500 for each tab — tests the HTML render path not covered by the existing JSON test

### `graph/mind_test.go`
Added `TestMindOnQuestionAsked_WithAgent`:
- Seeds an agent user so `GetFirstAgent` returns it
- Creates a question node
- Stubs `callClaudeOverride` so no real Claude call is needed
- Calls `OnQuestionAsked` synchronously
- Asserts a `KindComment` child was created with the agent as author and a non-empty body

## Verification
```
go.exe build -buildvcs=false ./...   → success (no errors)
```
Tests require `DATABASE_URL` (integration DB) and will skip without it — consistent with the rest of the test suite.
