// Package org provides the organization management service.
package org

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/agent"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/department"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/organization"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/role"
	"github.com/mCP-DevOS/ai-orchestration-platform/ent/team"
)

// Service provides organization CRUD operations.
type Service struct {
	client *ent.Client
}

// NewService creates a new organization service.
func NewService(client *ent.Client) *Service {
	return &Service{client: client}
}

// --- Organization ---

type CreateOrgInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type OrgView struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) CreateOrg(ctx context.Context, input CreateOrgInput) (*OrgView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	id := uuid.New().String()
	o, err := s.client.Organization.Create().
		SetID(id).
		SetName(input.Name).
		SetNillableDescription(nstr(input.Description)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}
	return orgToView(o), nil
}

func (s *Service) GetOrg(ctx context.Context, id string) (*OrgView, error) {
	o, err := s.client.Organization.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	return orgToView(o), nil
}

func (s *Service) ListOrgs(ctx context.Context) ([]*OrgView, error) {
	orgs, err := s.client.Organization.Query().
		Order(ent.Asc(organization.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orgs: %w", err)
	}
	result := make([]*OrgView, len(orgs))
	for i, o := range orgs {
		result[i] = orgToView(o)
	}
	return result, nil
}

func (s *Service) DeleteOrg(ctx context.Context, id string) error {
	return s.client.Organization.DeleteOneID(id).Exec(ctx)
}

// --- Department ---

type CreateDeptInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	LeadAgentID string `json:"lead_agent_id,omitempty"`
}

type DeptView struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	LeadAgentID string    `json:"lead_agent_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) CreateDept(ctx context.Context, orgID string, input CreateDeptInput) (*DeptView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	id := uuid.New().String()
	d, err := s.client.Department.Create().
		SetID(id).
		SetOrgID(orgID).
		SetName(input.Name).
		SetNillableDescription(nstr(input.Description)).
		SetNillableLeadAgentID(nstr(input.LeadAgentID)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create dept: %w", err)
	}
	return deptToView(d), nil
}

func (s *Service) ListDepts(ctx context.Context, orgID string) ([]*DeptView, error) {
	depts, err := s.client.Department.Query().
		Where(department.OrgID(orgID)).
		Order(ent.Asc(department.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list depts: %w", err)
	}
	result := make([]*DeptView, len(depts))
	for i, d := range depts {
		result[i] = deptToView(d)
	}
	return result, nil
}

func (s *Service) GetDept(ctx context.Context, id string) (*DeptView, error) {
	d, err := s.client.Department.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dept: %w", err)
	}
	return deptToView(d), nil
}

func (s *Service) DeleteDept(ctx context.Context, id string) error {
	return s.client.Department.DeleteOneID(id).Exec(ctx)
}

// --- Team ---

type CreateTeamInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	LeadAgentID string `json:"lead_agent_id,omitempty"`
}

type TeamView struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	DeptID      string    `json:"dept_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	LeadAgentID string    `json:"lead_agent_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) CreateTeam(ctx context.Context, orgID, deptID string, input CreateTeamInput) (*TeamView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	id := uuid.New().String()
	t, err := s.client.Team.Create().
		SetID(id).
		SetOrgID(orgID).
		SetDeptID(deptID).
		SetName(input.Name).
		SetNillableDescription(nstr(input.Description)).
		SetNillableLeadAgentID(nstr(input.LeadAgentID)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}
	return teamToView(t), nil
}

func (s *Service) ListTeams(ctx context.Context, deptID string) ([]*TeamView, error) {
	teams, err := s.client.Team.Query().
		Where(team.DeptID(deptID)).
		Order(ent.Asc(team.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	result := make([]*TeamView, len(teams))
	for i, t := range teams {
		result[i] = teamToView(t)
	}
	return result, nil
}

func (s *Service) GetTeam(ctx context.Context, id string) (*TeamView, error) {
	t, err := s.client.Team.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	return teamToView(t), nil
}

func (s *Service) DeleteTeam(ctx context.Context, id string) error {
	return s.client.Team.DeleteOneID(id).Exec(ctx)
}

// --- Role ---

type CreateRoleInput struct {
	Name        string   `json:"name"`
	Capabilities []string `json:"capabilities,omitempty"`
	Permissions  []string `json:"permissions,omitempty"`
}

type RoleView struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	TeamID      string    `json:"team_id"`
	Name        string    `json:"name"`
	Capabilities []string `json:"capabilities"`
	Permissions  []string `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) CreateRole(ctx context.Context, orgID, teamID string, input CreateRoleInput) (*RoleView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	capsJSON, _ := json.Marshal(input.Capabilities)
	permsJSON, _ := json.Marshal(input.Permissions)
	id := uuid.New().String()
	r, err := s.client.Role.Create().
		SetID(id).
		SetOrgID(orgID).
		SetTeamID(teamID).
		SetName(input.Name).
		SetCapabilities(string(capsJSON)).
		SetLegacyPermissions(string(permsJSON)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}
	return roleToView(r), nil
}

func (s *Service) ListRoles(ctx context.Context, teamID string) ([]*RoleView, error) {
	roles, err := s.client.Role.Query().
		Where(role.TeamID(teamID)).
		Order(ent.Asc(role.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	result := make([]*RoleView, len(roles))
	for i, r := range roles {
		result[i] = roleToView(r)
	}
	return result, nil
}

func (s *Service) GetRole(ctx context.Context, id string) (*RoleView, error) {
	r, err := s.client.Role.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	return roleToView(r), nil
}

func (s *Service) DeleteRole(ctx context.Context, id string) error {
	return s.client.Role.DeleteOneID(id).Exec(ctx)
}

// --- Agent ---

type CreateAgentInput struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	RoleID     string   `json:"role_id,omitempty"`
	Specialties []string `json:"specialties,omitempty"`
	Config     map[string]string `json:"config,omitempty"`
}

type UpdateAgentInput struct {
	Status     *string           `json:"status,omitempty"`
	RoleID     *string           `json:"role_id,omitempty"`
	Specialties []string          `json:"specialties,omitempty"`
	Config     map[string]string `json:"config,omitempty"`
}

type AgentView struct {
	ID              string            `json:"id"`
	OrgID           string            `json:"org_id"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	RoleID          string            `json:"role_id,omitempty"`
	Status          string            `json:"status"`
	Specialties     []string          `json:"specialties"`
	Config          map[string]string `json:"config"`
	LastHeartbeatAt time.Time         `json:"last_heartbeat_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
}

func (s *Service) CreateAgent(ctx context.Context, orgID string, input CreateAgentInput) (*AgentView, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	specJSON, _ := json.Marshal(input.Specialties)
	cfgJSON, _ := json.Marshal(input.Config)
	id := uuid.New().String()
	a, err := s.client.Agent.Create().
		SetID(id).
		SetOrgID(orgID).
		SetName(input.Name).
		SetType(input.Type).
		SetNillableRoleID(nstr(input.RoleID)).
		SetSpecialties(string(specJSON)).
		SetConfig(string(cfgJSON)).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	return agentToView(a), nil
}

func (s *Service) ListAgents(ctx context.Context, orgID string) ([]*AgentView, error) {
	agents, err := s.client.Agent.Query().
		Where(agent.OrgID(orgID)).
		Order(ent.Asc(agent.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	result := make([]*AgentView, len(agents))
	for i, a := range agents {
		result[i] = agentToView(a)
	}
	return result, nil
}

func (s *Service) GetAgent(ctx context.Context, id string) (*AgentView, error) {
	a, err := s.client.Agent.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return agentToView(a), nil
}

func (s *Service) UpdateAgent(ctx context.Context, id string, input UpdateAgentInput) (*AgentView, error) {
	u := s.client.Agent.UpdateOneID(id)
	if input.Status != nil {
		u.SetStatus(*input.Status)
	}
	if input.RoleID != nil {
		u.SetNillableRoleID(input.RoleID)
	}
	if input.Specialties != nil {
		specJSON, _ := json.Marshal(input.Specialties)
		u.SetSpecialties(string(specJSON))
	}
	if input.Config != nil {
		cfgJSON, _ := json.Marshal(input.Config)
		u.SetConfig(string(cfgJSON))
	}
	a, err := u.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}
	return agentToView(a), nil
}

func (s *Service) UpdateAgentHeartbeat(ctx context.Context, id string) error {
	return s.client.Agent.UpdateOneID(id).
		SetLastHeartbeatAt(time.Now()).
		Exec(ctx)
}

func (s *Service) DeleteAgent(ctx context.Context, id string) error {
	return s.client.Agent.DeleteOneID(id).Exec(ctx)
}

// --- helpers ---

func nstr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func orgToView(o *ent.Organization) *OrgView {
	return &OrgView{
		ID:          o.ID,
		Name:        o.Name,
		Description: o.Description,
		CreatedAt:   o.CreatedAt,
	}
}

func deptToView(d *ent.Department) *DeptView {
	return &DeptView{
		ID:          d.ID,
		OrgID:       d.OrgID,
		Name:        d.Name,
		Description: d.Description,
		LeadAgentID: d.LeadAgentID,
		CreatedAt:   d.CreatedAt,
	}
}

func teamToView(t *ent.Team) *TeamView {
	return &TeamView{
		ID:          t.ID,
		OrgID:       t.OrgID,
		DeptID:      t.DeptID,
		Name:        t.Name,
		Description: t.Description,
		LeadAgentID: t.LeadAgentID,
		CreatedAt:   t.CreatedAt,
	}
}

func roleToView(r *ent.Role) *RoleView {
	var caps, perms []string
	json.Unmarshal([]byte(r.Capabilities), &caps)
	json.Unmarshal([]byte(r.LegacyPermissions), &perms)
	return &RoleView{
		ID:           r.ID,
		OrgID:        r.OrgID,
		TeamID:       r.TeamID,
		Name:         r.Name,
		Capabilities: caps,
		Permissions:  perms,
		CreatedAt:    r.CreatedAt,
	}
}

func agentToView(a *ent.Agent) *AgentView {
	var spec []string
	var cfg map[string]string
	json.Unmarshal([]byte(a.Specialties), &spec)
	json.Unmarshal([]byte(a.Config), &cfg)
	return &AgentView{
		ID:              a.ID,
		OrgID:           a.OrgID,
		Name:            a.Name,
		Type:            a.Type,
		RoleID:          a.RoleID,
		Status:          a.Status,
		Specialties:     spec,
		Config:          cfg,
		LastHeartbeatAt: a.LastHeartbeatAt,
		CreatedAt:       a.CreatedAt,
	}
}
