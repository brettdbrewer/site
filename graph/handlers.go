package graph

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/lovyou-ai/site/auth"
)

// ViewUser holds user info for templates.
type ViewUser struct {
	Name    string
	Picture string
}

// Handlers serves the unified product HTTP endpoints.
type Handlers struct {
	store *Store
	wrap  func(http.HandlerFunc) http.Handler
}

// NewHandlers creates handlers with auth middleware.
func NewHandlers(store *Store, wrap func(http.HandlerFunc) http.Handler) *Handlers {
	if wrap == nil {
		wrap = func(hf http.HandlerFunc) http.Handler { return hf }
	}
	return &Handlers{store: store, wrap: wrap}
}

// Register adds all /app routes to the mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	// Space management.
	mux.Handle("GET /app", h.wrap(h.handleSpaceIndex))
	mux.Handle("POST /app/new", h.wrap(h.handleCreateSpace))

	// Space lenses.
	mux.Handle("GET /app/{slug}", h.wrap(h.handleSpaceDefault))
	mux.Handle("GET /app/{slug}/board", h.wrap(h.handleBoard))
	mux.Handle("GET /app/{slug}/feed", h.wrap(h.handleFeed))
	mux.Handle("GET /app/{slug}/threads", h.wrap(h.handleThreads))
	mux.Handle("GET /app/{slug}/people", h.wrap(h.handlePeople))
	mux.Handle("GET /app/{slug}/activity", h.wrap(h.handleActivity))

	// Node detail.
	mux.Handle("GET /app/{slug}/node/{id}", h.wrap(h.handleNodeDetail))

	// Grammar operations.
	mux.Handle("POST /app/{slug}/op", h.wrap(h.handleOp))

	// Node mutations.
	mux.Handle("POST /app/{slug}/node/{id}/state", h.wrap(h.handleNodeState))
	mux.Handle("POST /app/{slug}/node/{id}/update", h.wrap(h.handleNodeUpdate))
	mux.Handle("DELETE /app/{slug}/node/{id}", h.wrap(h.handleNodeDelete))
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) viewUser(r *http.Request) ViewUser {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return ViewUser{Name: "Anonymous"}
	}
	return ViewUser{Name: u.Name, Picture: u.Picture}
}

func (h *Handlers) userID(r *http.Request) string {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return "anonymous"
	}
	return u.ID
}

func (h *Handlers) userName(r *http.Request) string {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return "anonymous"
	}
	return u.Name
}

func (h *Handlers) spaceFromRequest(r *http.Request) (*Space, error) {
	slug := r.PathValue("slug")
	space, err := h.store.GetSpaceBySlug(r.Context(), slug)
	if err != nil {
		return nil, err
	}
	if space.OwnerID != h.userID(r) {
		return nil, ErrNotFound
	}
	return space, nil
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "space"
	}
	return s
}

// ────────────────────────────────────────────────────────────────────
// Space handlers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleSpaceIndex(w http.ResponseWriter, r *http.Request) {
	spaces, err := h.store.ListSpaces(r.Context(), h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(spaces) == 0 {
		SpaceOnboarding(h.viewUser(r)).Render(r.Context(), w)
		return
	}

	SpaceIndex(spaces, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleCreateSpace(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	kind := r.FormValue("kind")
	if kind == "" {
		kind = SpaceProject
	}

	slug := slugify(name)
	// Ensure unique slug by appending random suffix on conflict.
	space, err := h.store.CreateSpace(r.Context(), slug, name, description, h.userID(r), kind)
	if err != nil {
		// Try with random suffix.
		slug = slug + "-" + newID()[:6]
		space, err = h.store.CreateSpace(r.Context(), slug, name, description, h.userID(r), kind)
		if err != nil {
			log.Printf("graph: create space: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/app/"+space.Slug, http.StatusSeeOther)
}

func (h *Handlers) handleSpaceDefault(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Default lens: board for projects, feed for communities.
	lens := "board"
	if space.Kind == SpaceCommunity {
		lens = "feed"
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/"+lens, http.StatusSeeOther)
}

// ────────────────────────────────────────────────────────────────────
// Lens handlers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleBoard(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, err := h.store.ListSpaces(r.Context(), h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tasks, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		ParentID: "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	columns := groupByState(tasks)
	BoardView(*space, spaces, columns, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleFeed(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, err := h.store.ListSpaces(r.Context(), h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	posts, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindPost,
		ParentID: "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	FeedView(*space, spaces, posts, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleThreads(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, err := h.store.ListSpaces(r.Context(), h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	threads, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindThread,
		ParentID: "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ThreadsView(*space, spaces, threads, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handlePeople(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, err := h.store.ListSpaces(r.Context(), h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ops, err := h.store.ListOps(r.Context(), space.ID, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Aggregate by actor.
	memberMap := map[string]*Member{}
	for _, o := range ops {
		m, ok := memberMap[o.Actor]
		if !ok {
			m = &Member{Name: o.Actor}
			memberMap[o.Actor] = m
		}
		m.OpCount++
		m.LastSeen = o.CreatedAt.Format("Jan 2")
	}
	var members []Member
	for _, m := range memberMap {
		members = append(members, *m)
	}

	PeopleView(*space, spaces, members, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleActivity(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, err := h.store.ListSpaces(r.Context(), h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ops, err := h.store.ListOps(r.Context(), space.ID, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ActivityView(*space, spaces, ops, h.viewUser(r)).Render(r.Context(), w)
}

// ────────────────────────────────────────────────────────────────────
// Node detail
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleNodeDetail(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	children, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: nodeID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ops, err := h.store.ListNodeOps(r.Context(), nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	NodeDetailView(*space, *node, children, ops, h.viewUser(r)).Render(r.Context(), w)
}

// ────────────────────────────────────────────────────────────────────
// Grammar operation dispatcher
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleOp(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	op := r.FormValue("op")
	ctx := r.Context()
	actor := h.userName(r)

	switch op {
	case "intend":
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:  space.ID,
			Kind:     KindTask,
			Title:    title,
			Body:     strings.TrimSpace(r.FormValue("description")),
			Priority: r.FormValue("priority"),
			Assignee: strings.TrimSpace(r.FormValue("assignee")),
			Author:   actor,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, "intend", nil)

		if isHTMX(r) {
			TaskCard(*node, space.Slug).Render(ctx, w)
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)

	case "decompose":
		title := strings.TrimSpace(r.FormValue("title"))
		parentID := r.FormValue("parent_id")
		if title == "" || parentID == "" {
			http.Error(w, "title and parent_id required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:  space.ID,
			ParentID: parentID,
			Kind:     KindTask,
			Title:    title,
			Author:   actor,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, "decompose", nil)
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, parentID), http.StatusSeeOther)

	case "express":
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		if title == "" && body == "" {
			http.Error(w, "title or body required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID: space.ID,
			Kind:    KindPost,
			Title:   title,
			Body:    body,
			Author:  actor,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, "express", nil)

		if isHTMX(r) {
			FeedCard(*node, space.Slug).Render(ctx, w)
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/feed", http.StatusSeeOther)

	case "discuss":
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID: space.ID,
			Kind:    KindThread,
			Title:   title,
			Body:    body,
			Author:  actor,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, "discuss", nil)
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, node.ID), http.StatusSeeOther)

	case "respond":
		parentID := r.FormValue("parent_id")
		body := strings.TrimSpace(r.FormValue("body"))
		if parentID == "" || body == "" {
			http.Error(w, "parent_id and body required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:  space.ID,
			ParentID: parentID,
			Kind:     KindComment,
			Body:     body,
			Author:   actor,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, "respond", nil)

		if isHTMX(r) {
			CommentItem(*node).Render(ctx, w)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, parentID), http.StatusSeeOther)

	case "complete":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNodeState(ctx, nodeID, StateDone); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, "complete", nil)

		node, _ := h.store.GetNode(ctx, nodeID)
		if isHTMX(r) && node != nil {
			TaskCard(*node, space.Slug).Render(ctx, w)
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)

	case "assign":
		nodeID := r.FormValue("node_id")
		assignee := strings.TrimSpace(r.FormValue("assignee"))
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNode(ctx, nodeID, nil, nil, nil, &assignee); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, "assign", nil)
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "claim":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNode(ctx, nodeID, nil, nil, nil, &actor); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, "claim", nil)
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "prioritize":
		nodeID := r.FormValue("node_id")
		priority := r.FormValue("priority")
		if nodeID == "" || priority == "" {
			http.Error(w, "node_id and priority required", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNode(ctx, nodeID, nil, nil, &priority, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, "prioritize", nil)
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	default:
		http.Error(w, fmt.Sprintf("unknown op: %s", op), http.StatusBadRequest)
	}
}

// ────────────────────────────────────────────────────────────────────
// Node mutation handlers (non-op convenience routes)
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleNodeState(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	newState := r.FormValue("state")

	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.UpdateNodeState(r.Context(), nodeID, newState); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record as appropriate grammar op.
	opName := "progress"
	if newState == StateDone {
		opName = "complete"
	} else if newState == StateReview {
		opName = "review"
	}
	h.store.RecordOp(r.Context(), space.ID, nodeID, h.userName(r), opName, nil)

	node, _ = h.store.GetNode(r.Context(), nodeID)
	if isHTMX(r) && node != nil {
		TaskCard(*node, space.Slug).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)
}

func (h *Handlers) handleNodeUpdate(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")

	var title, body, priority, assignee *string
	if v := r.FormValue("title"); v != "" {
		title = &v
	}
	if r.Form != nil && r.Form.Has("body") {
		v := r.FormValue("body")
		body = &v
	}
	if v := r.FormValue("priority"); v != "" {
		priority = &v
	}
	if r.Form != nil && r.Form.Has("assignee") {
		v := r.FormValue("assignee")
		assignee = &v
	}

	if err := h.store.UpdateNode(r.Context(), nodeID, title, body, priority, assignee); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node, _ := h.store.GetNode(r.Context(), nodeID)
	if isHTMX(r) && node != nil {
		TaskCard(*node, space.Slug).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)
}

func (h *Handlers) handleNodeDelete(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	if err := h.store.DeleteNode(r.Context(), nodeID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/app/"+space.Slug+"/board")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)
}

// ────────────────────────────────────────────────────────────────────
// Board helpers
// ────────────────────────────────────────────────────────────────────

// BoardColumn holds nodes grouped by state for the kanban board.
type BoardColumn struct {
	State string
	Label string
	Nodes []Node
}

func groupByState(nodes []Node) []BoardColumn {
	columns := []BoardColumn{
		{State: StateOpen, Label: "Open"},
		{State: StateActive, Label: "Active"},
		{State: StateReview, Label: "Review"},
		{State: StateDone, Label: "Done"},
	}
	byState := map[string]*BoardColumn{}
	for i := range columns {
		byState[columns[i].State] = &columns[i]
	}
	for _, n := range nodes {
		if col, ok := byState[n.State]; ok {
			col.Nodes = append(col.Nodes, n)
		}
	}
	return columns
}

// Member holds aggregated activity data for the People lens.
type Member struct {
	Name     string
	OpCount  int
	LastSeen string
}
