package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/registry"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/runner"
)

// registerOrgRoutes registers organization management API routes.
func (s *Server) registerOrgRoutes(r chi.Router) {
	r.Route("/orgs", func(r chi.Router) {
		r.Get("/", s.handleListOrgs)
		r.Post("/", s.handleCreateOrg)
		r.Route("/{orgID}", func(r chi.Router) {
			r.Get("/", s.handleGetOrg)
			r.Delete("/", s.handleDeleteOrg)

			// Departments
			r.Route("/departments", func(r chi.Router) {
				r.Get("/", s.handleListDepts)
				r.Post("/", s.handleCreateDept)
				r.Route("/{deptID}", func(r chi.Router) {
					r.Get("/", s.handleGetDept)
					r.Delete("/", s.handleDeleteDept)

					// Teams
					r.Route("/teams", func(r chi.Router) {
						r.Get("/", s.handleListTeams)
						r.Post("/", s.handleCreateTeam)
						r.Route("/{teamID}", func(r chi.Router) {
							r.Get("/", s.handleGetTeam)
							r.Delete("/", s.handleDeleteTeam)

							// Roles
							r.Route("/roles", func(r chi.Router) {
								r.Get("/", s.handleListRoles)
								r.Post("/", s.handleCreateRole)
								r.Route("/{roleID}", func(r chi.Router) {
									r.Get("/", s.handleGetRole)
									r.Delete("/", s.handleDeleteRole)
								})
							})
						})
					})
				})
			})

			// Agents
			r.Route("/agents", func(r chi.Router) {
				r.Get("/", s.handleListAgents)
				r.Post("/", s.handleCreateAgent)
				r.Route("/{agentID}", func(r chi.Router) {
					r.Get("/", s.handleGetAgent)
					r.Patch("/", s.handleUpdateAgent)
					r.Delete("/", s.handleDeleteAgent)
					r.Post("/heartbeat", s.handleAgentHeartbeat)
					r.Get("/capabilities", s.handleGetAgentCapabilities)
				})
			})

			// Routing
			s.registerRoutingRoutes(r)
		})
	})
}

// --- Organization handlers ---

func (s *Server) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.orgSvc.ListOrgs(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: orgs})
}

func (s *Server) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	var input org.CreateOrgInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	o, err := s.orgSvc.CreateOrg(r.Context(), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: o})
}

func (s *Server) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	o, err := s.orgSvc.GetOrg(r.Context(), chi.URLParam(r, "orgID"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: o})
}

func (s *Server) handleDeleteOrg(w http.ResponseWriter, r *http.Request) {
	if err := s.orgSvc.DeleteOrg(r.Context(), chi.URLParam(r, "orgID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// --- Department handlers ---

func (s *Server) handleListDepts(w http.ResponseWriter, r *http.Request) {
	depts, err := s.orgSvc.ListDepts(r.Context(), chi.URLParam(r, "orgID"))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: depts})
}

func (s *Server) handleCreateDept(w http.ResponseWriter, r *http.Request) {
	var input org.CreateDeptInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	d, err := s.orgSvc.CreateDept(r.Context(), chi.URLParam(r, "orgID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: d})
}

func (s *Server) handleGetDept(w http.ResponseWriter, r *http.Request) {
	d, err := s.orgSvc.GetDept(r.Context(), chi.URLParam(r, "deptID"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: d})
}

func (s *Server) handleDeleteDept(w http.ResponseWriter, r *http.Request) {
	if err := s.orgSvc.DeleteDept(r.Context(), chi.URLParam(r, "deptID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// --- Team handlers ---

func (s *Server) handleListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := s.orgSvc.ListTeams(r.Context(), chi.URLParam(r, "deptID"))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: teams})
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var input org.CreateTeamInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	t, err := s.orgSvc.CreateTeam(r.Context(), chi.URLParam(r, "orgID"), chi.URLParam(r, "deptID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: t})
}

func (s *Server) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	t, err := s.orgSvc.GetTeam(r.Context(), chi.URLParam(r, "teamID"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: t})
}

func (s *Server) handleDeleteTeam(w http.ResponseWriter, r *http.Request) {
	if err := s.orgSvc.DeleteTeam(r.Context(), chi.URLParam(r, "teamID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// --- Role handlers ---

func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.orgSvc.ListRoles(r.Context(), chi.URLParam(r, "teamID"))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: roles})
}

func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var input org.CreateRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	role, err := s.orgSvc.CreateRole(r.Context(), chi.URLParam(r, "orgID"), chi.URLParam(r, "teamID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: role})
}

func (s *Server) handleGetRole(w http.ResponseWriter, r *http.Request) {
	role, err := s.orgSvc.GetRole(r.Context(), chi.URLParam(r, "roleID"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: role})
}

func (s *Server) handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	if err := s.orgSvc.DeleteRole(r.Context(), chi.URLParam(r, "roleID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

// --- Agent handlers ---

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.orgSvc.ListAgents(r.Context(), chi.URLParam(r, "orgID"))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: agents})
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var input org.CreateAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	a, err := s.orgSvc.CreateAgent(r.Context(), chi.URLParam(r, "orgID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Register the Runner in the registry after DB creation.
	// This enables immediate execution of tasks by the newly created agent.
	if s.runnerRegistry != nil {
		if run := s.buildRunnerFromAgentView(a); run != nil {
			s.runnerRegistry.Register(a.ID, run)
			s.logger.Info().Str("agent_id", a.ID).Str("runner_type", a.RunnerType).Msg("agent runner registered via API")
		}
	}

	s.writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: a})
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	a, err := s.orgSvc.GetAgent(r.Context(), chi.URLParam(r, "agentID"))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: a})
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	var input org.UpdateAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	a, err := s.orgSvc.UpdateAgent(r.Context(), agentID, input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}

	// If runner config changed, rebuild and re-register the Runner.
	if s.runnerRegistry != nil && (input.RunnerType != nil || input.RunnerConfig != nil) {
		if run := s.buildRunnerFromAgentView(a); run != nil {
			s.runnerRegistry.Register(agentID, run)
			s.logger.Info().Str("agent_id", agentID).Msg("agent runner re-registered via API")
		}
	}

	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: a})
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	if err := s.orgSvc.DeleteAgent(r.Context(), agentID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Unregister the Runner from the registry after DB deletion.
	if s.runnerRegistry != nil {
		if entry := s.runnerRegistry.Unregister(agentID); entry != nil {
			s.logger.Info().Str("agent_id", agentID).Msg("agent runner unregistered via API")
		}
	}

	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	if err := s.orgSvc.UpdateAgentHeartbeat(r.Context(), agentID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Update registry heartbeat status.
	if s.runnerRegistry != nil {
		s.runnerRegistry.SetStatus(agentID, registry.StatusOnline)
	}

	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleGetAgentCapabilities(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")

	if s.runnerRegistry == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, APIResponse{Success: false, Error: "registry not configured"})
		return
	}

	manifest, err := s.runnerRegistry.Manifest(r.Context(), agentID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, APIResponse{Success: false, Error: "agent not registered or no capabilities"})
		return
	}

	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: manifest})
}

type runnerConfigExtract struct {
	BasePath     string `json:"base_path"`
	MainRepo     string `json:"main_repo"`
	ArtifactBase string `json:"artifact_base"`
}

func (s *Server) buildRunnerFromAgentView(a *org.AgentView) runner.Runner {
	rt := registry.RunnerTypeFromString(a.RunnerType)
	cfg := registry.RegistryConfig{}
	var runnerCfg json.RawMessage

	if a.RunnerConfig != "" && a.RunnerConfig != "{}" {
		runnerCfg = json.RawMessage(a.RunnerConfig)
		var ext runnerConfigExtract
		if err := json.Unmarshal(runnerCfg, &ext); err == nil {
			cfg.BasePath = ext.BasePath
			cfg.MainRepo = ext.MainRepo
			cfg.ArtifactBase = ext.ArtifactBase
		}
	}

	built, err := registry.BuildRunner(registry.BuilderInput{
		AgentID:        a.ID,
		AgentName:      a.Name,
		RunnerType:     rt,
		Config:         runnerCfg,
		RegistryConfig: cfg,
	})
	if err != nil {
		s.logger.Warn().Err(err).Str("agent_id", a.ID).Str("runner_type", a.RunnerType).Msg("failed to build runner")
		return nil
	}
	return built
}
