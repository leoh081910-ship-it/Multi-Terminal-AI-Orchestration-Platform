package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
		r.Post("/{id}/instantiate/batch", s.handleInstantiateBatch)
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
	templateID := chi.URLParam(r, "id")

	var req struct {
		Params     map[string]any `json:"params"`
		DispatchRef string        `json:"dispatch_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	store := s.getTemplateStore()
	tpl, err := store.Get(ctx, templateID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "template not found",
		})
		return
	}

	// Validate required inputs against provided params
	for _, input := range tpl.Inputs {
		if input.Required {
			if val, ok := req.Params[input.Name]; !ok || val == nil || val == "" {
				s.writeJSON(w, http.StatusBadRequest, APIResponse{
					Success: false,
					Error:   fmt.Sprintf("missing required input parameter: %s", input.Name),
				})
				return
			}
		}
	}

	// Build task config from template with param substitution
	taskPayload := s.substituteTemplate(tpl, req.Params)
	taskPayload["project_id"] = tpl.ProjectID

	// Generate dispatch ref if not provided
	dispatchRef := req.DispatchRef
	if dispatchRef == "" {
		if tpl.ProjectID != "" {
			dispatchRef = tpl.ProjectID
		} else {
			dispatchRef = "template-" + templateID[:8]
		}
	}
	taskPayload["dispatch_ref"] = dispatchRef

	// Build the task card using the existing compat scheduler logic
	taskID := "TMPL-" + uuid.NewString()[:8]
	taskPayload["task_id"] = taskID
	taskPayload["id"] = taskID

	card, err := buildCompatTaskCard(taskPayload, nil, taskID, tpl.ProjectID)
	if err != nil {
		s.logger.Error().Err(err).Msg("instantiate: failed to build task card")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to instantiate template: " + err.Error(),
		})
		return
	}

	if _, err := s.repo.CreateTask(ctx, card); err != nil {
		s.logger.Error().Err(err).Str("template_id", templateID).Str("task_id", taskID).Msg("instantiate: failed to create task")
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "failed to create task from template",
		})
		return
	}

	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		s.logger.Error().Err(err).Str("task_id", taskID).Msg("instantiate: failed to reload task")
		s.writeJSON(w, http.StatusCreated, APIResponse{
			Success: true,
			Data:    map[string]string{"id": taskID},
		})
		return
	}

	s.writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    s.mapCompatTask(task),
	})
}

// substituteTemplate renders template fields by replacing {{param}} placeholders with provided values.
func (s *Server) substituteTemplate(tpl *template.TaskTemplateSQL, params map[string]any) map[string]any {
	result := make(map[string]any)

	// String fields that support template substitution
	strFields := map[string]string{
		"name":        tpl.Name,
		"description": tpl.Description,
		"command":     tpl.Command,
		"work_dir":     tpl.WorkDir,
	}
	for key, val := range strFields {
		if val != "" {
			result[key] = s.substituteString(val, params)
		}
	}

	// Direct value fields
	if tpl.TaskType != "" {
		result["type"] = tpl.TaskType
	}
	if tpl.OwnerAgent != "" {
		result["owner_agent"] = tpl.OwnerAgent
	}
	if tpl.Priority > 0 {
		result["priority"] = tpl.Priority
	}

	// Propagate user-provided params (allows overriding template defaults)
	for k, v := range params {
		result[k] = v
	}

	return result
}

// substituteString replaces {{param}} and {{param.default}} placeholders with actual values.
func (s *Server) substituteString(text string, params map[string]any) string {
	// Pattern: {{paramName}} or {{paramName:defaultValue}}
	re := regexp.MustCompile(`\{\{([^}:]+)(?::([^}]*))?\}\}`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		matches := re.FindStringSubmatch(match)
		paramName := matches[1]
		defaultVal := ""
		if len(matches) > 2 {
			defaultVal = matches[2]
		}
		if val, ok := params[paramName]; ok && val != nil {
			return fmt.Sprintf("%v", val)
		}
		return defaultVal
	})
}

// handleInstantiateBatch instantiates a template and creates multiple tasks with sequential params.
// POST /templates/{id}/instantiate/batch
func (s *Server) handleInstantiateBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templateID := chi.URLParam(r, "id")

	var req struct {
		Params       []map[string]any `json:"params"` // array of param sets, one per task
		DispatchRef  string            `json:"dispatch_ref"`
		Wave         int              `json:"wave"`
		TopoRankStart int              `json:"topo_rank_start"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
		return
	}

	if len(req.Params) == 0 {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "params array is required and must not be empty",
		})
		return
	}
	if len(req.Params) > 100 {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "too many tasks (max 100)",
		})
		return
	}

	tpl, err := s.getTemplateStore().Get(ctx, templateID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "template not found",
		})
		return
	}

	dispatchRef := req.DispatchRef
	if dispatchRef == "" {
		if tpl.ProjectID != "" {
			dispatchRef = tpl.ProjectID
		} else {
			dispatchRef = "template-batch-" + templateID[:8]
		}
	}

	wave := req.Wave
	if wave == 0 {
		wave = 1
	}
	topoRankStart := req.TopoRankStart

	created := make([]compatSchedulerTask, 0, len(req.Params))
	errors := make([]map[string]any, 0)

	for i, params := range req.Params {
		taskPayload := s.substituteTemplate(tpl, params)
		taskID := fmt.Sprintf("TMPL-%s-%03d", uuid.NewString()[:8], i+1)
		taskPayload["task_id"] = taskID
		taskPayload["id"] = taskID
		taskPayload["project_id"] = tpl.ProjectID
		taskPayload["dispatch_ref"] = dispatchRef
		taskPayload["wave"] = wave
		taskPayload["topo_rank"] = topoRankStart + i

		card, err := buildCompatTaskCard(taskPayload, nil, taskID, tpl.ProjectID)
		if err != nil {
			errors = append(errors, map[string]any{
				"index":  i,
				"detail": err.Error(),
			})
			continue
		}

		if _, err := s.repo.CreateTask(ctx, card); err != nil {
			s.logger.Error().Err(err).Int("index", i).Msg("batch instantiate: failed to create task")
			errors = append(errors, map[string]any{
				"index":  i,
				"detail": "failed to create task",
			})
			continue
		}

		task, err := s.repo.GetTaskByID(ctx, taskID)
		if err != nil {
			errors = append(errors, map[string]any{
				"index":  i,
				"detail": "failed to reload task",
			})
			continue
		}

		created = append(created, s.mapCompatTask(task))
	}

	status := http.StatusCreated
	if len(errors) > 0 && len(created) == 0 {
		status = http.StatusBadRequest
	}

	s.writeJSON(w, status, APIResponse{
		Success: len(errors) == 0,
		Data: map[string]any{
			"created": created,
			"errors":  errors,
			"total":   len(req.Params),
			"succeeded": len(created),
			"failed":  len(errors),
		},
	})
}
