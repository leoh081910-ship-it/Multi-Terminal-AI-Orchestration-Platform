import client from './client';

export interface TreeNode {
  id: string;
  parent_id?: string;
  root_id?: string;
  depth: number;
  state: string;
  decomposition_status: string;
  title: string;
  assigned_agent_id?: string;
  children?: TreeNode[];
}

export interface SubtaskSpec {
  title: string;
  description?: string;
  type: string;
  depends_on?: string[];
  files_to_modify?: string[];
  assigned_agent_id?: string;
  acceptance_criteria?: string[];
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

export const goalApi = {
  createGoal: async (orgId: string, input: { title: string; description?: string; wave?: number }): Promise<{ card_json: string }> =>
    client.post(`/orgs/${orgId}/goals`, input).then(unwrap) as Promise<{ card_json: string }>,

  decomposeGoal: async (orgId: string, goalId: string, agentId: string): Promise<{ subtasks: SubtaskSpec[]; rationale: string }> =>
    client.post(`/orgs/${orgId}/goals/${goalId}/decompose`, { agent_id: agentId }).then(unwrap) as Promise<{ subtasks: SubtaskSpec[]; rationale: string }>,

  getGoalTree: async (orgId: string, goalId: string): Promise<TreeNode> =>
    client.get(`/orgs/${orgId}/goals/${goalId}/tree`).then(unwrap) as Promise<TreeNode>,

  confirmDecomposition: async (orgId: string, goalId: string, subtasks: SubtaskSpec[], rationale?: string): Promise<{ child_task_ids: string[]; count: number }> =>
    client.patch(`/orgs/${orgId}/goals/${goalId}/confirm`, { subtasks, rationale }).then(unwrap) as Promise<{ child_task_ids: string[]; count: number }>,

  rejectDecomposition: async (orgId: string, goalId: string): Promise<void> =>
    client.patch(`/orgs/${orgId}/goals/${goalId}/reject`).then(unwrap) as Promise<void>,

  assignTask: async (orgId: string, taskId: string, agentId: string, roleId?: string): Promise<void> =>
    client.post(`/orgs/${orgId}/tasks/${taskId}/assign`, { agent_id: agentId, role_id: roleId }).then(unwrap) as Promise<void>,
};
