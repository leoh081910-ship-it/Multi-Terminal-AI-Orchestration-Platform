// API Types
export interface Task {
  id: string
  dispatch_ref: string
  state: TaskState
  retry_count: number
  loop_iteration_count: number
  transport: string
  wave: number
  topo_rank: number
  workspace_path?: string
  artifact_path?: string
  last_error_reason?: string
  created_at: string
  updated_at: string
  terminal_at?: string
  card_json: string
}

export type TaskState =
  | 'queued'
  | 'routed'
  | 'workspace_prepared'
  | 'running'
  | 'patch_ready'
  | 'verified'
  | 'merged'
  | 'done'
  | 'retry_waiting'
  | 'verify_failed'
  | 'apply_failed'
  | 'failed'

export interface TaskCard {
  id?: string
  dispatch_ref?: string
  source?: string
  source_ref?: string
  type?: string
  objective?: string
  context?: Record<string, unknown>
  files_to_read?: string[]
  files_to_modify?: string[]
  acceptance_criteria?: string[]
  relations?: TaskRelation[]
  wave?: number
  priority?: number
}

export interface TaskRelation {
  taskId: string
  type: 'depends_on' | 'conflicts_with'
  reason?: string
}

export interface TaskStats {
  total: number
  byState: Record<string, number>
  recent: Task[]
}

export interface CreateTaskInput {
  id: string
  dispatch_ref: string
  transport: 'cli' | 'api'
  state?: string
  retry_count?: number
  loop_iteration_count?: number
  wave?: number
  topo_rank?: number
  workspace_path?: string
  artifact_path?: string
  last_error_reason?: string
  card_json: string
}

export interface Wave {
  dispatchRef: string
  wave: number
  sealedAt?: string
  createdAt: string
}

export interface Event {
  eventId: string
  taskId: string
  eventType: string
  fromState?: string
  toState?: string
  timestamp: string
  reason?: string
  attempt: number
  transport?: string
  runnerId?: string
  details?: string
}

// UI Types
export type Theme = 'light' | 'dark' | 'system'

export interface Notification {
  id: string
  type: 'info' | 'success' | 'warning' | 'error'
  message: string
  timestamp: number
}

export interface FilterState {
  search: string
  states: TaskState[]
  waves: number[]
  dateRange: { start?: Date; end?: Date }
}

export interface SortState {
  field: keyof Task
  direction: 'asc' | 'desc'
}

// WebSocket Types
export interface WebSocketMessage {
  type: 'task_update' | 'event_log' | 'state_change' | 'ping'
  data: unknown
  timestamp: string
}

export interface TaskUpdateMessage extends WebSocketMessage {
  type: 'task_update'
  data: {
    taskId: string
    oldState: TaskState
    newState: TaskState
    timestamp: string
  }
}

// Chart Types
export interface StateTransitionData {
  from: TaskState
  to: TaskState
  count: number
}

export interface TimeSeriesData {
  timestamp: string
  value: number
  label: string
}

// Form Types
export interface TaskFormData {
  objective: string
  type: string
  filesToModify: string[]
  acceptanceCriteria: string[]
  priority: number
}

// Error Types
export interface ApiError {
  code: string
  message: string
  details?: Record<string, string[]>
}

export interface ValidationError {
  field: string
  message: string
}
