export const Agent = {
  CLAUDE: 'Claude',
  GEMINI: 'Gemini',
  CODEX: 'Codex',
} as const;

export type Agent = (typeof Agent)[keyof typeof Agent];

export const TaskStatus = {
  BACKLOG: 'backlog',
  READY: 'ready',
  ASSIGNED: 'assigned',
  IN_PROGRESS: 'in_progress',
  REVIEW: 'review',
  VERIFIED: 'verified',
  BLOCKED: 'blocked',
  DONE: 'done',
  TRIAGE: 'triage',
  REVIEW_PENDING: 'review_pending',
} as const;

export type TaskStatus = (typeof TaskStatus)[keyof typeof TaskStatus];

export const DispatchMode = {
  MANUAL: 'manual',
  AUTO: 'auto',
} as const;

export type DispatchMode = (typeof DispatchMode)[keyof typeof DispatchMode];

export const DispatchStatus = {
  PENDING: 'pending',
  QUEUED: 'queued',
  DISPATCHED: 'dispatched',
  RUNNING: 'running',
  FAILED: 'failed',
  COMPLETED: 'completed',
  TRIAGE: 'triage',
  REVIEW_PENDING: 'review_pending',
} as const;

export type DispatchStatus = (typeof DispatchStatus)[keyof typeof DispatchStatus];

export const RuntimeStatus = {
  AVAILABLE: 'available',
  UNAVAILABLE: 'unavailable',
} as const;

export type RuntimeStatus = (typeof RuntimeStatus)[keyof typeof RuntimeStatus];

export const TaskType = {
  COLLECT: 'collect',
  ANALYZE: 'analyze',
  SELECT: 'select',
  LIST: 'list',
  UI: 'ui',
  ALGORITHM: 'algorithm',
  INTEGRATION: 'integration',
  DECISION_LOGIC: 'decision-logic',
  RANKING_RULE: 'ranking-rule',
} as const;

export type TaskType = string;

export interface ScheduledTask {
  project_id?: string;
  id: string;
  title: string;
  owner_agent: Agent;
  status: TaskStatus;
  type: string;
  priority: number;
  description?: string;
  depends_on?: string[];
  input_artifacts?: string[];
  output_artifacts?: string[];
  acceptance_criteria?: string[];
  block_reason?: string;
  result_summary?: string;
  next_action?: string;
  current_focus?: string;
  dispatch_mode: DispatchMode;
  auto_dispatch_enabled: boolean;
  dispatch_status: DispatchStatus;
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

  // PR-4: computed diagnostic fields
  next_automatic_action?: string;
  stop_loss_status?: string;
  recoverable?: boolean;
}

export interface CreateScheduledTaskInput {
  title: string;
  owner_agent: Agent;
  status?: TaskStatus;
  type?: string;
  priority?: number;
  description?: string;
  depends_on?: string[];
  input_artifacts?: string[];
  output_artifacts?: string[];
  acceptance_criteria?: string[];
  block_reason?: string;
  next_action?: string;
  dispatch_mode?: DispatchMode;
  auto_dispatch_enabled?: boolean;
}

export interface UpdateScheduledTaskInput {
  owner_agent?: Agent;
  status?: TaskStatus;
  priority?: number;
  description?: string;
  depends_on?: string[];
  input_artifacts?: string[];
  output_artifacts?: string[];
  acceptance_criteria?: string[];
  block_reason?: string;
  result_summary?: string;
  next_action?: string;
  current_focus?: string;
  dispatch_mode?: DispatchMode;
  auto_dispatch_enabled?: boolean;
  dispatch_status?: DispatchStatus;
  last_dispatch_error?: string;
}

export interface BulkImportTaskDraft {
  line: number;
  raw: string;
  payload?: CreateScheduledTaskInput;
  error?: string;
}

export interface BulkImportResult {
  created: ScheduledTask[];
  failed: BulkImportTaskDraft[];
}

export interface TaskExecutionEvent {
  event: string;
  status: DispatchStatus;
  message: string;
  timestamp: string;
}

export interface TaskExecution {
  execution_id: string;
  task_id: string;
  owner_agent: Agent;
  runtime: string;
  session_id: string;
  status: DispatchStatus;
  command: string;
  workspace?: string;
  artifacts?: string;
  log_path?: string;
  output_tail?: string;
  started_at: string;
  updated_at: string;
  completed_at?: string;
  error?: string;
  events: TaskExecutionEvent[];
}

export interface RuntimeSummary {
  owner_agent: Agent;
  runtime: string;
  status: RuntimeStatus;
  command_configured: boolean;
  active_sessions: number;
  last_heartbeat_at?: string;
  last_error?: string;
}

export interface DashboardStats {
  total_tasks: number;
  agent_task_counts: Record<Agent, number>;
  blocked_count: number;
  recent_done_tasks: ScheduledTask[];
  recent_updates: ScheduledTask[];
  current_focus: string;
  runtime_health: RuntimeSummary[];
  active_sessions: number;
  failed_dispatches: number;
  queued_tasks: number;
}

// PRD-DA-001 types

export interface TaskLineage {
  root_task: ScheduledTask;
  ancestors: ScheduledTask[];
  descendants: ScheduledTask[];
  siblings: ScheduledTask[];
}

export interface FailurePolicyConfig {
  max_auto_repairs: number;
  max_auto_retries: number;
  backoff_intervals: string[];
  failure_codes: Array<{
    code: string;
    description: string;
    retryable: string;
  }>;
}

export interface AgentConfig {
  agent_id: string;
  agent_role: string;
  capabilities: string[];
  specialties: string[];
  enabled: boolean;
}

// PR-OPS-003 system health types

export interface SystemHealth {
  status: string;
  timestamp: string;
  uptime: string;
  version: string;
}

export interface WorkerStatus {
  name: string;
  running: boolean;
  interval?: string;
}

export interface SystemWorkersResponse {
  workers: WorkerStatus[];
}
