package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/template"
)

type createTemplateRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	ProjectID   string              `json:"project_id"`
	OwnerAgent  string              `json:"owner_agent"`
	TaskType    string              `json:"task_type"`
	Priority    int                 `json:"priority"`
	Command     string              `json:"command"`
	WorkDir     string              `json:"work_dir"`
	TimeoutSec  int                 `json:"timeout_sec"`
	Tags        []string            `json:"tags"`
	Inputs      []template.Input    `json:"inputs"`
	Outputs     []template.Output   `json:"outputs"`
	CreatedBy   string              `json:"created_by"`
}

type updateTemplateRequest struct {
	Name        *string             `json:"name,omitempty"`
	Description *string             `json:"description,omitempty"`
	ProjectID   *string             `json:"project_id,omitempty"`
	OwnerAgent  *string             `json:"owner_agent,omitempty"`
	TaskType    *string             `json:"task_type,omitempty"`
	Priority    *int                `json:"priority,omitempty"`
	Command     *string             `json:"command,omitempty"`
	WorkDir     *string             `json:"work_dir,omitempty"`
	TimeoutSec  *int                `json:"timeout_sec,omitempty"`
	Tags        *[]string           `json:"tags,omitempty"`
	Inputs      *[]template.Input   `json:"inputs,omitempty"`
	Outputs     *[]template.Output  `json:"outputs,omitempty"`
}

func (s *Server) registerTemplateRoutes() {
	s.router.Route("/templates", func(r chi.Router) {
		r.Get("/", s.handleListTemplates)
		r.Post("/", s.handleCreateTemplate)
		r.Get("/{id}", s.handleGetTemplate)
		r.Put("/{id}", s.handleUpdateTemplate)
		r.Delete("/{id}", s.handleDeleteTemplate)
		r.Post("/{id}/instantiate", s.handleInstantiateTemplate)
	})
}

func (s *Server) getTemplateStore() *template.Store {
	return s.templateStore
}

func (s *Server) initTemplateTable(ctx context.Context) error {
	return s.getTemplateStore().InitTable(ctx)
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := r.URL.Query().Get("project_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	tpls, total, err := s.getTemplateStore().List(ctx, projectID, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list templates")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to list templates",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"items":  tpls,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req createTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}
	if req.Name == "" {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "name is required",
		})
		return
	}
	if req.OwnerAgent == "" {
		req.OwnerAgent = "Claude"
	}
	if req.TaskType == "" {
		req.TaskType = "task"
	}
	if req.Priority == 0 {
		req.Priority = 3
	}
	if req.TimeoutSec == 0 {
		req.TimeoutSec = 1800
	}

	tpl := &template.TaskTemplateSQL{
		Name:        req.Name,
		Description: req.Description,
		ProjectID:   req.ProjectID,
		OwnerAgent:  req.OwnerAgent,
		TaskType:    req.TaskType,
		Priority:    req.Priority,
		Command:     req.Command,
		WorkDir:     req.WorkDir,
		TimeoutSec:  req.TimeoutSec,
		Tags:        req.Tags,
		Inputs:      req.Inputs,
		Outputs:     req.Outputs,
		CreatedBy:   req.CreatedBy,
	}

	if err := s.getTemplateStore().Create(ctx, tpl); err != nil {
		s.logger.Error().Err(err).Msg("failed to create template")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to create template",
		})
		return
	}

	s.writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    tpl,
	})
}

func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	tpl, err := s.getTemplateStore().Get(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("failed to get template")
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "template not found",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    tpl,
	})
}

func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req updateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ProjectID != nil {
		updates["project_id"] = *req.ProjectID
	}
	if req.OwnerAgent != nil {
		updates["owner_agent"] = *req.OwnerAgent
	}
	if req.TaskType != nil {
		updates["task_type"] = *req.TaskType
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Command != nil {
		updates["command"] = *req.Command
	}
	if req.WorkDir != nil {
		updates["work_dir"] = *req.WorkDir
	}
	if req.TimeoutSec != nil {
		updates["timeout_sec"] = *req.TimeoutSec
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.Inputs != nil {
		updates["inputs"] = *req.Inputs
	}
	if req.Outputs != nil {
		updates["outputs"] = *req.Outputs
	}

	if len(updates) == 0 {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "no fields to update",
		})
		return
	}

	if err := s.getTemplateStore().Update(ctx, id, updates); err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("failed to update template")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to update template",
		})
		return
	}

	tpl, err := s.getTemplateStore().Get(ctx, id)
	if err != nil {
		s.writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    map[string]string{"id": id},
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    tpl,
	})
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := s.getTemplateStore().Delete(ctx, id); err != nil {
		s.logger.Error().Err(err).Str("id", id).Msg("failed to delete template")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to delete template",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"id": id},
	})
}

func (s *Server) handleInstantiateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var req struct {
		Params map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Params = make(map[string]any)
	}

	store := s.getTemplateStore()
	tpl, err := store.Get(ctx, id)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "template not found",
		})
		return
	}

	// Build task config from template
	taskConfig := map[string]any{
		"title":        tpl.Name,
		"owner_agent":  tpl.OwnerAgent,
		"type":         tpl.TaskType,
		"priority":     tpl.Priority,
		"command":      tpl.Command,
		"workspace_path": tpl.WorkDir,
		"task_type":    tpl.TaskType,
	}

	// Merge user params
	for k, v := range req.Params {
		taskConfig[k] = v
	}

	s.writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    taskConfig,
	})
}
