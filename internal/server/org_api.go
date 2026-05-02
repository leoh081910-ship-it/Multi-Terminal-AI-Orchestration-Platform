package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mCP-DevOS/ai-orchestration-platform/internal/org"
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
	var input org.UpdateAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: "invalid request body"})
		return
	}
	a, err := s.orgSvc.UpdateAgent(r.Context(), chi.URLParam(r, "agentID"), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: a})
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	if err := s.orgSvc.DeleteAgent(r.Context(), chi.URLParam(r, "agentID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if err := s.orgSvc.UpdateAgentHeartbeat(r.Context(), chi.URLParam(r, "agentID")); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, APIResponse{Success: true})
}
