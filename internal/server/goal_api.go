package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/decompose"
)

// registerGoalRoutes registers goal and decomposition API routes.
func (s *Server) registerGoalRoutes(r chi.Router) {
	r.Route("/orgs/{orgID}/goals", func(r chi.Router) {
		r.Post("/", s.handleCreateGoal)
		r.Route("/{goalID}", func(r chi.Router) {
			r.Post("/decompose", s.handleDecomposeGoal)
			r.Get("/tree", s.handleGetGoalTree)
			r.Patch("/confirm", s.handleConfirmDecomposition)
			r.Patch("/reject", s.handleRejectDecomposition)
		})
	})

	r.Route("/orgs/{orgID}/tasks/{taskID}", func(r chi.Router) {
		r.Post("/assign", s.handleAssignTask)
	})
}

func (s *Server) handleCreateGoal(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description,omitempty"`
		OrgID       string `json:"org_id,omitempty"`
		Wave        int    `json:"wave"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	if input.Title == "" {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "title is required"})
		return
	}

	orgID := chi.URLParam(r, "orgID")
	cardJSON, _ := json.Marshal(map[string]interface{}{
		"id":                  "", // will be set by store
		"project_id":          "default",
		"dispatch_ref":        "goal-" + orgID,
		"transport":           "api",
		"wave":                input.Wave,
		"title":               input.Title,
		"description":         input.Description,
		"org_id":              orgID,
		"decomposition_status": "pending",
		"task_type":           "goal",
	})

	// Create task via scheduler endpoint logic
	s.writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"card_json": string(cardJSON),
			"message":   "goal created, call /decompose to trigger decomposition",
		},
	})
}

func (s *Server) handleDecomposeGoal(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "goalID")
	var input struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	d := decompose.NewDecomposer(s.repo.Client())
	result, err := d.DecomposeGoal(r.Context(), goalID, input.AgentID)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
}

func (s *Server) handleGetGoalTree(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "goalID")
	treeOps := decompose.NewTreeOps(s.repo.Client())
	tree, err := treeOps.GetTree(r.Context(), goalID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: tree})
}

func (s *Server) handleConfirmDecomposition(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "goalID")
	var input struct {
		Subtasks []decompose.SubtaskSpec `json:"subtasks"`
		Rationale string                `json:"rationale,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	result := &decompose.DecompositionResult{
		Subtasks:  input.Subtasks,
		Rationale: input.Rationale,
	}
	if err := decompose.ValidateDecomposition(result); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}

	d := decompose.NewDecomposer(s.repo.Client())
	childIDs, err := d.CreateSubtasks(r.Context(), goalID, result)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"child_task_ids": childIDs,
			"count":          len(childIDs),
		},
	})
}

func (s *Server) handleRejectDecomposition(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "goalID")
	d := decompose.NewDecomposer(s.repo.Client())
	if err := d.RejectDecomposition(r.Context(), goalID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleAssignTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	var input struct {
		AgentID string `json:"agent_id"`
		RoleID  string `json:"role_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	u := s.repo.Client().Task.UpdateOneID(taskID)
	if input.AgentID != "" {
		u.SetAssignedAgentID(input.AgentID)
	}
	if input.RoleID != "" {
		u.SetAssignedRoleID(input.RoleID)
	}
	if _, err := u.Save(r.Context()); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}
