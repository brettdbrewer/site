package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lovyou-ai/site/auth"
)

// TestKnowledgePublic verifies unauthenticated GET /app/{slug}/knowledge returns 200
// for a public space.
func TestKnowledgePublic(t *testing.T) {
	_, store := testDB(t)
	slug := fmt.Sprintf("test-knowledge-pub-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Knowledge Public Test", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	h := NewHandlers(store, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/app/"+slug+"/knowledge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// TestKnowledgeAuthed verifies authenticated GET /app/{slug}/knowledge returns 200.
func TestKnowledgeAuthed(t *testing.T) {
	_, store := testDB(t)
	slug := fmt.Sprintf("test-knowledge-auth-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Knowledge Authed Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	testUser := &auth.User{ID: "test-user-1", Name: "Tester", Email: "test@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	h := NewHandlers(store, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/app/"+slug+"/knowledge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// TestAssertOpReturnsCauses verifies that op=assert with a causes field stores
// the causes and returns them in the JSON response. This is the core of
// Invariant 2 (CAUSALITY) for knowledge claims.
func TestAssertOpReturnsCauses(t *testing.T) {
	_, store := testDB(t)
	slug := fmt.Sprintf("test-causes-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Causes Test", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Create a document node to serve as the cause.
	cause, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindDocument,
		Title:      "Build Report: iteration 1",
		AuthorID:   "owner-1",
		AuthorKind: "human",
	})
	if err != nil {
		t.Fatalf("create cause node: %v", err)
	}

	testUser := &auth.User{ID: "owner-1", Name: "Owner", Email: "owner@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	h := NewHandlers(store, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	// Assert a claim with a cause.
	payload := fmt.Sprintf(`{"op":"assert","kind":"claim","title":"Absence is invisible","body":"Test body","causes":%q}`, cause.ID)
	req := httptest.NewRequest("POST", "/app/"+slug+"/op", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("assert op: status = %d, want 201; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Node Node `json:"node"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Node.Causes) != 1 || resp.Node.Causes[0] != cause.ID {
		t.Errorf("causes = %v, want [%s]", resp.Node.Causes, cause.ID)
	}

	// Also verify /knowledge?tab=claims returns the causes field.
	req2 := httptest.NewRequest("GET", "/app/"+slug+"/knowledge?tab=claims", nil)
	req2.Header.Set("Accept", "application/json")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("knowledge: status = %d, want 200; body: %s", w2.Code, w2.Body.String())
	}

	var kr struct {
		Claims []Node `json:"claims"`
	}
	if err := json.NewDecoder(w2.Body).Decode(&kr); err != nil {
		t.Fatalf("decode knowledge response: %v", err)
	}
	if len(kr.Claims) == 0 {
		t.Fatal("knowledge returned 0 claims, want at least 1")
	}
	found := false
	for _, c := range kr.Claims {
		if c.Title == "Absence is invisible" {
			found = true
			if len(c.Causes) != 1 || c.Causes[0] != cause.ID {
				t.Errorf("claim causes = %v, want [%s]", c.Causes, cause.ID)
			}
			break
		}
	}
	if !found {
		t.Error("claim not found in /knowledge response")
	}
}

// TestKnowledgeClaimsCausesFieldPresent verifies that the causes key is always
// present in the JSON response for claims, even when no causes are declared.
// This validates Invariant 2 (CAUSALITY): the API must not silently drop the
// causes field via omitempty — consumers must be able to distinguish "no causes
// declared" (empty array) from "field not supported" (missing key).
func TestKnowledgeClaimsCausesFieldPresent(t *testing.T) {
	_, store := testDB(t)
	slug := fmt.Sprintf("test-causes-present-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Causes Present Test", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	testUser := &auth.User{ID: "owner-1", Name: "Owner", Email: "owner@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	h := NewHandlers(store, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	// Assert a claim with no causes.
	payload := `{"op":"assert","kind":"claim","title":"Uncaused claim","body":"No causes declared"}`
	req := httptest.NewRequest("POST", "/app/"+slug+"/op", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("assert op: status = %d, want 201; body: %s", w.Code, w.Body.String())
	}

	// Verify the response JSON has a "causes" key (not omitted).
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	nodeRaw, ok := raw["node"]
	if !ok {
		t.Fatal("response missing 'node' key")
	}
	var nodeMap map[string]json.RawMessage
	if err := json.Unmarshal(nodeRaw, &nodeMap); err != nil {
		t.Fatalf("decode node: %v", err)
	}
	if _, ok := nodeMap["causes"]; !ok {
		t.Error("node JSON missing 'causes' key — Invariant 2 violation: causes must always be present")
	}

	// Also verify /knowledge JSON response includes causes key on all claims.
	req2 := httptest.NewRequest("GET", "/app/"+slug+"/knowledge", nil)
	req2.Header.Set("Accept", "application/json")
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("knowledge: status = %d, want 200; body: %s", w2.Code, w2.Body.String())
	}

	var kr struct {
		Claims []json.RawMessage `json:"claims"`
	}
	if err := json.NewDecoder(w2.Body).Decode(&kr); err != nil {
		t.Fatalf("decode knowledge response: %v", err)
	}
	if len(kr.Claims) == 0 {
		t.Fatal("knowledge returned 0 claims, want at least 1")
	}
	for i, rawClaim := range kr.Claims {
		var claimMap map[string]json.RawMessage
		if err := json.Unmarshal(rawClaim, &claimMap); err != nil {
			t.Fatalf("decode claim %d: %v", i, err)
		}
		if _, ok := claimMap["causes"]; !ok {
			t.Errorf("claim %d JSON missing 'causes' key — Invariant 2 violation", i)
		}
	}
}

// TestKnowledgeMissingSpace verifies GET /app/{slug}/knowledge returns 404 when the
// space does not exist.
func TestKnowledgeMissingSpace(t *testing.T) {
	_, store := testDB(t)

	h := NewHandlers(store, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/app/nonexistent-space-xyz/knowledge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
