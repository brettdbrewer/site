package work

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// Handlers serves the Work product HTTP endpoints.
type Handlers struct {
	store *Store
}

// NewHandlers creates Work product handlers backed by the given store.
func NewHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}

// Register adds all /work routes to the mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	// Pages.
	mux.HandleFunc("GET /work", h.handleIndex)
	mux.HandleFunc("GET /work/project/{id}", h.handleBoard)
	mux.HandleFunc("GET /work/project/{id}/list", h.handleList)
	mux.HandleFunc("GET /work/task/{id}", h.handleTaskDetail)

	// Mutations.
	mux.HandleFunc("POST /work/project", h.handleCreateProject)
	mux.HandleFunc("POST /work/task", h.handleCreateTask)
	mux.HandleFunc("POST /work/task/{id}/state", h.handleTransitionTask)
	mux.HandleFunc("POST /work/task/{id}/update", h.handleUpdateTask)
	mux.HandleFunc("POST /work/task/{id}/comment", h.handleAddComment)
	mux.HandleFunc("DELETE /work/task/{id}", h.handleDeleteTask)

	// API.
	mux.HandleFunc("GET /work/api/tasks", h.handleAPITasks)
}

// ────────────────────────────────────────────────────────────────────
// Page handlers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	projects, err := h.store.ListProjects(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(projects) == 0 {
		// Onboarding: show create-project form.
		WorkOnboarding().Render(ctx, w)
		return
	}

	// Redirect to first project's board.
	http.Redirect(w, r, "/work/project/"+projects[0].ID, http.StatusSeeOther)
}

func (h *Handlers) handleBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("id")

	project, err := h.store.GetProject(ctx, projectID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projects, err := h.store.ListProjects(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tasks, err := h.store.ListTasks(ctx, ListTasksParams{
		ProjectID: projectID,
		ParentID:  "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Group tasks by state.
	columns := groupByState(tasks)

	WorkBoard(*project, projects, columns).Render(ctx, w)
}

func (h *Handlers) handleList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.PathValue("id")

	project, err := h.store.GetProject(ctx, projectID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projects, err := h.store.ListProjects(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filters from query params.
	params := ListTasksParams{
		ProjectID: projectID,
		ParentID:  "root",
	}
	if state := r.URL.Query().Get("state"); state != "" {
		params.State = state
	}
	if assignee := r.URL.Query().Get("assignee"); assignee != "" {
		params.Assignee = assignee
	}

	tasks, err := h.store.ListTasks(ctx, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WorkList(*project, projects, tasks).Render(ctx, w)
}

func (h *Handlers) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")

	task, err := h.store.GetTask(ctx, taskID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	project, err := h.store.GetProject(ctx, task.ProjectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	comments, err := h.store.ListComments(ctx, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	subtasks, err := h.store.ListTasks(ctx, ListTasksParams{
		ProjectID: task.ProjectID,
		ParentID:  taskID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	blockers, err := h.store.ListBlockers(ctx, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WorkTaskDetail(*project, *task, comments, subtasks, blockers).Render(ctx, w)
}

// ────────────────────────────────────────────────────────────────────
// Mutation handlers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	owner := strings.TrimSpace(r.FormValue("owner"))
	if owner == "" {
		owner = "anonymous"
	}

	if name == "" {
		http.Error(w, "project name is required", http.StatusBadRequest)
		return
	}

	project, err := h.store.CreateProject(ctx, name, description, owner)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/work/project/"+project.ID, http.StatusSeeOther)
}

func (h *Handlers) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "task title is required", http.StatusBadRequest)
		return
	}

	priority := r.FormValue("priority")
	if priority == "" {
		priority = PriorityMedium
	}

	createdBy := strings.TrimSpace(r.FormValue("created_by"))
	if createdBy == "" {
		createdBy = "anonymous"
	}

	params := CreateTaskParams{
		Title:       title,
		Description: strings.TrimSpace(r.FormValue("description")),
		Priority:    priority,
		ProjectID:   r.FormValue("project_id"),
		ParentID:    r.FormValue("parent_id"),
		Assignee:    strings.TrimSpace(r.FormValue("assignee")),
		CreatedBy:   createdBy,
	}

	task, err := h.store.CreateTask(ctx, params)
	if err != nil {
		log.Printf("work: create task: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// HTMX request: return the task card fragment.
	if isHTMX(r) {
		TaskCard(*task).Render(ctx, w)
		return
	}

	// Standard form submit: redirect to board.
	http.Redirect(w, r, "/work/project/"+task.ProjectID, http.StatusSeeOther)
}

func (h *Handlers) handleTransitionTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")
	newState := r.FormValue("state")

	if err := h.store.TransitionTask(ctx, taskID, newState); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, ErrInvalidState) || errors.Is(err, ErrInvalidTransition) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task, err := h.store.GetTask(ctx, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		TaskCard(*task).Render(ctx, w)
		return
	}
	http.Redirect(w, r, "/work/project/"+task.ProjectID, http.StatusSeeOther)
}

func (h *Handlers) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")

	params := UpdateTaskParams{}
	if v := r.FormValue("title"); v != "" {
		params.Title = &v
	}
	if v := r.FormValue("description"); r.Form != nil && r.Form.Has("description") {
		params.Description = &v
	}
	if v := r.FormValue("priority"); v != "" {
		params.Priority = &v
	}
	if r.Form != nil && r.Form.Has("assignee") {
		v := r.FormValue("assignee")
		params.Assignee = &v
	}

	if err := h.store.UpdateTask(ctx, taskID, params); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.store.GetTask(ctx, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		TaskCard(*task).Render(ctx, w)
		return
	}
	http.Redirect(w, r, "/work/task/"+taskID, http.StatusSeeOther)
}

func (h *Handlers) handleAddComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")

	body := strings.TrimSpace(r.FormValue("body"))
	author := strings.TrimSpace(r.FormValue("author"))
	if author == "" {
		author = "anonymous"
	}
	if body == "" {
		http.Error(w, "comment body is required", http.StatusBadRequest)
		return
	}

	comment, err := h.store.AddComment(ctx, taskID, author, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		CommentItem(*comment).Render(ctx, w)
		return
	}
	http.Redirect(w, r, "/work/task/"+taskID, http.StatusSeeOther)
}

func (h *Handlers) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.PathValue("id")

	// Get task before deleting so we know the project for redirect.
	task, err := h.store.GetTask(ctx, taskID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.DeleteTask(ctx, task.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/work/project/"+task.ProjectID)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/work/project/"+task.ProjectID, http.StatusSeeOther)
}

// ────────────────────────────────────────────────────────────────────
// API handlers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleAPITasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		http.Error(w, "project query param required", http.StatusBadRequest)
		return
	}

	params := ListTasksParams{
		ProjectID: projectID,
		ParentID:  "root",
	}
	if state := r.URL.Query().Get("state"); state != "" {
		params.State = state
	}

	tasks, err := h.store.ListTasks(ctx, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

// BoardColumn holds tasks grouped by state for the kanban board.
type BoardColumn struct {
	State string
	Label string
	Tasks []Task
}

func groupByState(tasks []Task) []BoardColumn {
	columns := []BoardColumn{
		{State: StateBacklog, Label: "Backlog"},
		{State: StateTodo, Label: "To Do"},
		{State: StateDoing, Label: "Doing"},
		{State: StateReview, Label: "Review"},
		{State: StateDone, Label: "Done"},
	}
	byState := map[string]*BoardColumn{}
	for i := range columns {
		byState[columns[i].State] = &columns[i]
	}
	for _, t := range tasks {
		if col, ok := byState[t.State]; ok {
			col.Tasks = append(col.Tasks, t)
		}
	}
	return columns
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
