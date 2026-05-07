package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/router"
)

// registerRoutingRoutes adds routing-related endpoints.
func (s *Server) registerRoutingRoutes(r chi.Router) {
	r.Route("/routing", func(r chi.Router) {
		r.Post("/preview", s.handleRoutingPreview)
		r.Get("/config", s.handleGetRoutingConfig)
		r.Put("/config", s.handlePutRoutingConfig)
	})
}

// handleRoutingPreview shows which agents would be selected for a task.
// POST /api/v1/orgs/{orgID}/routing/preview
func (s *Server) handleRoutingPreview(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgID")
	if orgID == "" {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "orgID is required"})
		return
	}

	if s.taskRouter == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "router not configured"})
		return
	}

	var input struct {
		TaskType       string   `json:"task_type"`
		Capabilities   []string `json:"capabilities"`
		AssignedRoleID string   `json:"assigned_role_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	routeInput := router.RouteInput{
		TaskType:       input.TaskType,
		Capabilities:   input.Capabilities,
		OrgID:          orgID,
		AssignedRoleID: input.AssignedRoleID,
	}

	scores, err := s.taskRouter.ScoreAgents(r.Context(), routeInput)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: scores})
}

// handleGetRoutingConfig returns the current routing configuration.
// GET /api/v1/orgs/{orgID}/routing/config
func (s *Server) handleGetRoutingConfig(w http.ResponseWriter, r *http.Request) {
	if s.taskRouter == nil {
		s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: router.DefaultConfig()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: s.taskRouter.Config()})
}

// handlePutRoutingConfig updates the routing configuration at runtime.
// PUT /api/v1/orgs/{orgID}/routing/config
func (s *Server) handlePutRoutingConfig(w http.ResponseWriter, r *http.Request) {
	if s.taskRouter == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "router not configured"})
		return
	}

	var cfg router.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	// Validate strategy
	switch cfg.Strategy {
	case router.StrategyCapabilityMatch, router.StrategyLoadBalance,
		router.StrategyAffinity, router.StrategyWeighted, router.StrategyCapabilityV2, router.StrategyManual:
		// valid
	default:
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid routing strategy"})
		return
	}

	// Validate weights are in valid range
	weights := cfg.Weights
	if weights.Capability < 0 || weights.Capability > 1 || weights.Load < 0 || weights.Load > 1 || weights.Affinity < 0 || weights.Affinity > 1 {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "weights must be between 0.0 and 1.0"})
		return
	}

	s.taskRouter.SetConfig(cfg)
	s.logger.Info().Str("strategy", string(cfg.Strategy)).
		Float64("capability_weight", cfg.Weights.Capability).
		Float64("load_weight", cfg.Weights.Load).
		Float64("affinity_weight", cfg.Weights.Affinity).
		Msg("routing config updated")

	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: cfg})
}