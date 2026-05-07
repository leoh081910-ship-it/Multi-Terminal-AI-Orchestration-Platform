import client from './client';

export interface Organization {
  id: string;
  name: string;
  description?: string;
  created_at: string;
}

export interface Department {
  id: string;
  org_id: string;
  name: string;
  description?: string;
  lead_agent_id?: string;
  created_at: string;
}

export interface Team {
  id: string;
  org_id: string;
  dept_id: string;
  name: string;
  description?: string;
  lead_agent_id?: string;
  created_at: string;
}

export interface Role {
  id: string;
  org_id: string;
  team_id: string;
  name: string;
  capabilities: string[];
  permissions: string[];
  created_at: string;
}

export interface Agent {
  id: string;
  org_id: string;
  name: string;
  type: string;
  role_id?: string;
  status: string;
  specialties: string[];
  config: Record<string, string>;
  last_heartbeat_at?: string;
  created_at: string;
  runner_type?: string;
  runner_config?: string;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

function unwrap(resp: { data: ApiResponse<unknown> }) {
  if (!resp.data.success) throw new Error(resp.data.error || 'request failed');
  return resp.data.data;
}

export const orgApi = {
  // Organizations
  listOrgs: async (): Promise<Organization[]> =>
    client.get('/orgs').then(unwrap) as Promise<Organization[]>,

  createOrg: async (input: { name: string; description?: string }): Promise<Organization> =>
    client.post('/orgs', input).then(unwrap) as Promise<Organization>,

  getOrg: async (orgId: string): Promise<Organization> =>
    client.get(`/orgs/${orgId}`).then(unwrap) as Promise<Organization>,

  deleteOrg: async (orgId: string): Promise<void> =>
    client.delete(`/orgs/${orgId}`).then(unwrap) as Promise<void>,

  // Departments
  listDepts: async (orgId: string): Promise<Department[]> =>
    client.get(`/orgs/${orgId}/departments`).then(unwrap) as Promise<Department[]>,

  createDept: async (orgId: string, input: { name: string; description?: string }): Promise<Department> =>
    client.post(`/orgs/${orgId}/departments`, input).then(unwrap) as Promise<Department>,

  deleteDept: async (orgId: string, deptId: string): Promise<void> =>
    client.delete(`/orgs/${orgId}/departments/${deptId}`).then(unwrap) as Promise<void>,

  // Teams
  listTeams: async (orgId: string, deptId: string): Promise<Team[]> =>
    client.get(`/orgs/${orgId}/departments/${deptId}/teams`).then(unwrap) as Promise<Team[]>,

  createTeam: async (orgId: string, deptId: string, input: { name: string; description?: string }): Promise<Team> =>
    client.post(`/orgs/${orgId}/departments/${deptId}/teams`, input).then(unwrap) as Promise<Team>,

  // Agents
  listAgents: async (orgId: string): Promise<Agent[]> =>
    client.get(`/orgs/${orgId}/agents`).then(unwrap) as Promise<Agent[]>,

  createAgent: async (orgId: string, input: { name: string; type: string; role_id?: string; specialties?: string[]; runner_type?: string; runner_config?: string }): Promise<Agent> =>
    client.post(`/orgs/${orgId}/agents`, input).then(unwrap) as Promise<Agent>,

  updateAgent: async (orgId: string, agentId: string, input: { status?: string; role_id?: string; runner_type?: string; runner_config?: string }): Promise<Agent> =>
    client.patch(`/orgs/${orgId}/agents/${agentId}`, input).then(unwrap) as Promise<Agent>,

  deleteAgent: async (orgId: string, agentId: string): Promise<void> =>
    client.delete(`/orgs/${orgId}/agents/${agentId}`).then(unwrap) as Promise<void>,

  agentHeartbeat: async (orgId: string, agentId: string): Promise<void> =>
    client.post(`/orgs/${orgId}/agents/${agentId}/heartbeat`).then(unwrap) as Promise<void>,

  getAgentCapabilities: async (orgId: string, agentId: string): Promise<CapabilityManifest> =>
    client.get(`/orgs/${orgId}/agents/${agentId}/capabilities`).then(unwrap) as Promise<CapabilityManifest>,

  // Routing
  previewRouting: async (orgId: string, input: { task_type: string; capabilities?: string[]; assigned_role_id?: string }): Promise<AgentScore[]> =>
    client.post(`/orgs/${orgId}/routing/preview`, input).then(unwrap) as Promise<AgentScore[]>,
};

export interface AgentScore {
  AgentID: string;
  AgentName: string;
  Score: number;
  Capability: number;
  Load: number;
  Affinity: number;
  Available: boolean;
  ManifestScore?: number;
}

export interface CapabilityManifest {
  name: string;
  task_types: string[];
  model_family: string;
  context_window: number;
  max_input_size: number;
  supports_thinking: boolean;
  supports_multimodal: boolean;
  supports_streaming: boolean;
  latency_p50_ms: number;
  cost_per_1k_input_cents: number;
  cost_per_1k_output_cents: number;
  runtime: string;
  version: string;
  tags: string[];
}
