import client from './client';
import { Agent, DispatchStatus } from '../types/scheduler';
import type {
  AgentConfig,
  BulkImportResult,
  BulkImportTaskDraft,
  CreateScheduledTaskInput,
  DashboardStats,
  FailurePolicyConfig,
  RuntimeSummary,
  ScheduledTask,
  SystemHealth,
  SystemWorkersResponse,
  TaskExecution,
  TaskLineage,
  UpdateScheduledTaskInput,
} from '../types/scheduler';

interface BackendScheduledTask {
  project_id?: string;
  task_id: string;
  title: string;
  owner_agent: ScheduledTask['owner_agent'];
  status: ScheduledTask['status'];
  type: string;
  priority: number;
  description?: string;
  depends_on?: string[];
  input_artifacts?: string[];
  output_artifacts?: string[];
  acceptance_criteria?: string[];
  blocked_reason?: string;
  result_summary?: string;
  next_action?: string;
  current_focus?: string;
  dispatch_mode: ScheduledTask['dispatch_mode'];
  auto_dispatch_enabled: boolean;
  dispatch_status: ScheduledTask['dispatch_status'];
  execution_runtime?: string;
  execution_session_id?: string;
  last_dispatch_at?: string;
  dispatch_attempts: number;
  last_dispatch_error?: string;
  created_at: string;
  updated_at: string;

  // PRD-DA-001 coordination fields
  parent_task_id?: string;
  root_task_id?: string;
  derived_from_failure?: string;
  coordination_stage?: string;
  review_decision?: string;
  failure_code?: string;
  failure_signature?: string;
  auto_repair_count?: number;
  last_review_task_id?: string;

  // PR-OPS-003 execution heartbeat fields
  started_at?: string;
  last_heartbeat_at?: string;
  timeout_at?: string;
  stalled?: boolean;
}

interface BoardSummaryResponse {
  total_tasks: number;
  counts_by_status: Record<string, number>;
  counts_by_agent: Record<string, number>;
  blocked_count: number;
  recent_updates: BackendScheduledTask[];
  recent_done_tasks: BackendScheduledTask[];
  current_focus: BackendScheduledTask[];
}

interface AgentSummaryResponse {
  agents: Array<{
    owner_agent: ScheduledTask['owner_agent'];
    total_tasks: number;
    counts_by_status: Record<string, number>;
  }>;
}

interface RuntimeResponse {
  runtimes: RuntimeSummary[];
}

interface ExecutionResponse {
  executions: TaskExecution[];
}

const mapTask = (task: BackendScheduledTask): ScheduledTask => ({
  project_id: task.project_id,
  id: task.task_id,
  title: task.title,
  owner_agent: task.owner_agent,
  status: task.status,
  type: task.type,
  priority: task.priority,
  description: task.description,
  depends_on: task.depends_on,
  input_artifacts: task.input_artifacts,
  output_artifacts: task.output_artifacts,
  acceptance_criteria: task.acceptance_criteria,
  block_reason: task.blocked_reason,
  result_summary: task.result_summary,
  next_action: task.next_action,
  current_focus: task.current_focus,
  dispatch_mode: task.dispatch_mode,
  auto_dispatch_enabled: task.auto_dispatch_enabled,
  dispatch_status: task.dispatch_status,
  execution_runtime: task.execution_runtime,
  execution_session_id: task.execution_session_id,
  last_dispatch_at: task.last_dispatch_at,
  dispatch_attempts: task.dispatch_attempts,
  last_dispatch_error: task.last_dispatch_error,
  created_at: task.created_at,
  updated_at: task.updated_at,

  // PRD-DA-001 coordination fields
  parent_task_id: task.parent_task_id,
  root_task_id: task.root_task_id,
  derived_from_failure: task.derived_from_failure,
  coordination_stage: task.coordination_stage,
  review_decision: task.review_decision,
  failure_code: task.failure_code,
  failure_signature: task.failure_signature,
  auto_repair_count: task.auto_repair_count,
  last_review_task_id: task.last_review_task_id,

  // PR-OPS-003 heartbeat fields
  started_at: task.started_at,
  last_heartbeat_at: task.last_heartbeat_at,
  timeout_at: task.timeout_at,
  stalled: task.stalled,
});

const buildCurrentFocusLabel = (tasks: ScheduledTask[]): string => {
  if (!tasks.length) return 'No active focus';
  if (tasks.length === 1) return tasks[0].title;
  return `${tasks[0].title} +${tasks.length - 1}`;
};

const projectBase = (projectId: string) => `/projects/${encodeURIComponent(projectId)}`;

export const schedulerApi = {
  getTasks: async (projectId: string): Promise<ScheduledTask[]> => {
    const response = await client.get<BackendScheduledTask[]>(`${projectBase(projectId)}/scheduler/tasks`);
    return response.data.map(mapTask);
  },

  createTask: async (projectId: string, payload: CreateScheduledTaskInput): Promise<ScheduledTask> => {
    const response = await client.post<BackendScheduledTask>(`${projectBase(projectId)}/scheduler/tasks`, {
      ...payload,
      blocked_reason: payload.block_reason,
    });
    return mapTask(response.data);
  },

  createTasksBulk: async (projectId: string, drafts: BulkImportTaskDraft[]): Promise<BulkImportResult> => {
    const created: ScheduledTask[] = [];
    const failed: BulkImportTaskDraft[] = [];

    for (const draft of drafts) {
      if (!draft.payload) {
        failed.push(draft);
        continue;
      }

      try {
        const task = await schedulerApi.createTask(projectId, draft.payload);
        created.push(task);
      } catch (error) {
        failed.push({
          ...draft,
          error: error instanceof Error ? error.message : 'Failed to create task',
        });
      }
    }

    return { created, failed };
  },

  updateTask: async (projectId: string, taskId: string, payload: UpdateScheduledTaskInput): Promise<ScheduledTask> => {
    const response = await client.patch<BackendScheduledTask>(`${projectBase(projectId)}/scheduler/tasks/${taskId}`, {
      ...payload,
      blocked_reason: payload.block_reason,
    });
    return mapTask(response.data);
  },

  dispatchTask: async (projectId: string, taskId: string): Promise<ScheduledTask> => {
    const response = await client.post<BackendScheduledTask>(`${projectBase(projectId)}/scheduler/tasks/${taskId}/dispatch`);
    return mapTask(response.data);
  },

  retryTask: async (projectId: string, taskId: string): Promise<ScheduledTask> => {
    const response = await client.post<BackendScheduledTask>(`${projectBase(projectId)}/scheduler/tasks/${taskId}/retry`);
    return mapTask(response.data);
  },

  getTaskExecution: async (projectId: string, taskId: string): Promise<TaskExecution> => {
    const response = await client.get<TaskExecution>(`${projectBase(projectId)}/scheduler/tasks/${taskId}/execution`);
    return response.data;
  },

  getExecutions: async (projectId: string): Promise<TaskExecution[]> => {
    const response = await client.get<ExecutionResponse>(`${projectBase(projectId)}/scheduler/executions`);
    return response.data.executions;
  },

  getRuntimes: async (projectId: string): Promise<RuntimeSummary[]> => {
    const response = await client.get<RuntimeResponse>(`${projectBase(projectId)}/scheduler/runtimes`);
    return response.data.runtimes;
  },

  getStats: async (projectId: string): Promise<DashboardStats> => {
    const [boardResponse, agentsResponse, runtimesResponse, executionsResponse] = await Promise.all([
      client.get<BoardSummaryResponse>(`${projectBase(projectId)}/board/summary`),
      client.get<AgentSummaryResponse>(`${projectBase(projectId)}/agents/summary`),
      client.get<RuntimeResponse>(`${projectBase(projectId)}/scheduler/runtimes`),
      client.get<ExecutionResponse>(`${projectBase(projectId)}/scheduler/executions`),
    ]);

    const board = boardResponse.data;
    const agents = agentsResponse.data;
    const runtimes = runtimesResponse.data.runtimes;
    const executions = executionsResponse.data.executions;
    const currentFocusTasks = board.current_focus.map(mapTask);
    const allRecentUpdates = board.recent_updates.map(mapTask);
    const agentTaskCounts: Record<ScheduledTask['owner_agent'], number> = {
      [Agent.CLAUDE]: 0,
      [Agent.GEMINI]: 0,
      [Agent.CODEX]: 0,
    };

    for (const agent of agents.agents) {
      agentTaskCounts[agent.owner_agent] = agent.total_tasks;
    }

    const queuedStatuses: DispatchStatus[] = [
      DispatchStatus.QUEUED,
      DispatchStatus.DISPATCHED,
      DispatchStatus.RUNNING,
    ];

    return {
      total_tasks: board.total_tasks,
      agent_task_counts: agentTaskCounts,
      blocked_count: board.blocked_count,
      recent_done_tasks: board.recent_done_tasks.map(mapTask),
      recent_updates: allRecentUpdates,
      current_focus: buildCurrentFocusLabel(currentFocusTasks),
      runtime_health: runtimes,
      active_sessions: runtimes.reduce((sum, runtime) => sum + runtime.active_sessions, 0),
      failed_dispatches: allRecentUpdates.filter((task) => task.dispatch_status === DispatchStatus.FAILED).length,
      queued_tasks: executions.filter((execution) => queuedStatuses.includes(execution.status)).length,
    };
  },

  // PRD-DA-001 coordination APIs

  getTaskLineage: async (projectId: string, taskId: string): Promise<TaskLineage> => {
    const response = await client.get<TaskLineage>(`${projectBase(projectId)}/scheduler/tasks/${taskId}/lineage`);
    const data = response.data;
    return {
      root_task: data.root_task ? mapTask(data.root_task as unknown as BackendScheduledTask) : data.root_task,
      ancestors: (data.ancestors || []).map((p: unknown) => mapTask(p as BackendScheduledTask)),
      descendants: (data.descendants || []).map((c: unknown) => mapTask(c as BackendScheduledTask)),
      siblings: (data.siblings || []).map((s: unknown) => mapTask(s as BackendScheduledTask)),
    };
  },

  getFailurePolicies: async (projectId: string): Promise<FailurePolicyConfig> => {
    const response = await client.get<FailurePolicyConfig>(`${projectBase(projectId)}/scheduler/failure-policies`);
    return response.data;
  },

  getAgents: async (projectId: string): Promise<AgentConfig[]> => {
    const response = await client.get<{ agents: AgentConfig[] }>(`${projectBase(projectId)}/scheduler/agents`);
    return response.data.agents;
  },

  triggerTriage: async (projectId: string, taskId: string): Promise<ScheduledTask> => {
    const response = await client.post<BackendScheduledTask>(`${projectBase(projectId)}/scheduler/tasks/${taskId}/triage`);
    return mapTask(response.data);
  },

  // PR-OPS-003 system health APIs

  getSystemHealth: async (): Promise<SystemHealth> => {
    const response = await client.get<{ success: boolean; data: SystemHealth }>('/system/health');
    return response.data.data;
  },

  getSystemWorkers: async (): Promise<SystemWorkersResponse> => {
    const response = await client.get<{ success: boolean; data: SystemWorkersResponse }>('/system/workers');
    return response.data.data;
  },
};
