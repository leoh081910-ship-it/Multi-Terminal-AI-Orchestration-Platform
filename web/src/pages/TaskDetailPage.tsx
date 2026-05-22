import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { schedulerApi } from '../api/schedulerApi';
import { useWebSocket } from '../hooks/useWebSocket';
import { useProject } from '../hooks/useProject';
import type { AgentCallRecord, ScheduledTask, SchedulerEvent, TaskLineage } from '../types/scheduler';
import '../types/scheduler';

const STATUS_COLORS: Record<string, string> = {
  backlog: '#6b7280',
  ready: '#3b82f6',
  assigned: '#8b5cf6',
  in_progress: '#f59e0b',
  review: '#ec4899',
  verified: '#10b981',
  blocked: '#ef4444',
  done: '#22c55e',
  triage: '#f97316',
  review_pending: '#a855f7',
};

const AGENT_COLORS: Record<string, string> = {
  Claude: '#d97706',
  Gemini: '#2563eb',
  Codex: '#7c3aed',
};

export default function TaskDetailPage() {
  const { taskId } = useParams<{ taskId: string }>();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const { projectId } = useProject();
  const { lastMessage } = useWebSocket(projectId);

  const [activeTab, setActiveTab] = useState<'details' | 'events' | 'lineage' | 'logs' | 'agent-calls'>('details');

  const { data: taskData = null, isLoading, error } = useQuery({
    queryKey: ['task', projectId, taskId],
    queryFn: () => schedulerApi.getTask(projectId, taskId!),
    enabled: !!taskId && !!projectId,
  });

  // Real-time updates via WebSocket
  useEffect(() => {
    if (!lastMessage || !taskId) return;
    if (lastMessage.project_id && lastMessage.project_id !== projectId) return;
    const affectsTask = lastMessage.task_id === taskId || lastMessage.type === 'task.state_changed';
    if (!affectsTask) return;
    qc.invalidateQueries({ queryKey: ['task', projectId, taskId] });
    qc.invalidateQueries({ queryKey: ['task-events', projectId, taskId] });
    qc.invalidateQueries({ queryKey: ['task-execution', projectId, taskId] });
    qc.invalidateQueries({ queryKey: ['lineage', projectId, taskId] });
    qc.invalidateQueries({ queryKey: ['agent-calls', taskId] });
  }, [lastMessage, taskId, projectId, qc]);

  const { data: events } = useQuery({
    queryKey: ['task-events', projectId, taskId],
    queryFn: () => schedulerApi.getTaskEvents(projectId, taskId!),
    enabled: !!taskId && !!projectId && activeTab === 'events',
  });

  const { data: execution } = useQuery({
    queryKey: ['task-execution', projectId, taskId],
    queryFn: () => schedulerApi.getTaskExecution(projectId, taskId!),
    enabled: !!taskId && !!projectId && activeTab === 'logs',
  });

  const { data: lineage } = useQuery({
    queryKey: ['lineage', projectId, taskId],
    queryFn: () => schedulerApi.getTaskLineage(projectId, taskId!),
    enabled: !!taskId && !!projectId && activeTab === 'lineage',
  });

  const { data: agentCalls } = useQuery({
    queryKey: ['agent-calls', taskId],
    queryFn: () => schedulerApi.getAgentCalls(taskId!),
    enabled: !!taskId && activeTab === 'agent-calls',
  });

  const retryMutation = useMutation({
    mutationFn: () => schedulerApi.retryTask(projectId, taskId!),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task', projectId, taskId] }); },
  });

  const dispatchMutation = useMutation({
    mutationFn: () => schedulerApi.dispatchTask(projectId, taskId!),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task', projectId, taskId] }); },
  });

  if (isLoading) return <div className="p-8 text-center text-gray-400">Loading...</div>;
  if (error || !taskData) return (
    <div className="p-8 text-center">
      <p className="text-red-400 mb-4">Task not found</p>
      <button onClick={() => navigate('/board')} className="text-blue-400 hover:underline">Back to board</button>
    </div>
  );

  const t = taskData;
  const statusColor = STATUS_COLORS[t.status] || '#6b7280';
  const agentColor = AGENT_COLORS[t.owner_agent] || '#6b7280';
  const canRetry = t.status === 'blocked' || t.dispatch_status === 'failed';
  const canDispatch = t.status === 'ready' || t.status === 'backlog';

  return (
    <div className="max-w-5xl mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="space-y-2">
          <div className="flex items-center gap-3">
            <span
              className="inline-block px-2.5 py-0.5 rounded-full text-xs font-medium text-white"
              style={{ backgroundColor: statusColor }}
            >
              {t.status}
            </span>
            <span className="text-xs text-gray-500">{t.id}</span>
          </div>
          <h1 className="text-2xl font-bold text-gray-100">{t.title}</h1>
          <div className="flex items-center gap-4 text-sm text-gray-400">
            <span style={{ color: agentColor }}>{t.owner_agent}</span>
            <span>Priority: {t.priority}</span>
            <span>Type: {t.type}</span>
            {t.execution_runtime && <span>Runtime: {t.execution_runtime}</span>}
          </div>
        </div>
        <div className="flex gap-2">
          {canDispatch && (
            <button
              onClick={() => dispatchMutation.mutate()}
              disabled={dispatchMutation.isPending}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700 disabled:opacity-50"
            >
              Dispatch
            </button>
          )}
          {canRetry && (
            <button
              onClick={() => retryMutation.mutate()}
              disabled={retryMutation.isPending}
              className="px-4 py-2 bg-amber-600 text-white rounded-lg text-sm hover:bg-amber-700 disabled:opacity-50"
            >
              Retry
            </button>
          )}
          <button
            onClick={() => navigate('/board')}
            className="px-4 py-2 bg-gray-700 text-gray-300 rounded-lg text-sm hover:bg-gray-600"
          >
            Back
          </button>
        </div>
      </div>

      {/* Description */}
      {t.description && (
        <div className="bg-gray-800 rounded-lg p-4">
          <h3 className="text-sm font-medium text-gray-400 mb-2">Description</h3>
          <p className="text-gray-200 whitespace-pre-wrap">{t.description}</p>
        </div>
      )}

      {/* Result Summary */}
      {t.result_summary && (
        <div className="bg-gray-800 rounded-lg p-4 border-l-4 border-green-500">
          <h3 className="text-sm font-medium text-gray-400 mb-2">Result</h3>
          <p className="text-gray-200 whitespace-pre-wrap">{t.result_summary}</p>
        </div>
      )}

      {/* Error */}
      {t.last_dispatch_error && (
        <div className="bg-red-900/30 rounded-lg p-4 border-l-4 border-red-500">
          <h3 className="text-sm font-medium text-red-400 mb-2">Error</h3>
          <p className="text-red-200 whitespace-pre-wrap font-mono text-sm">{t.last_dispatch_error}</p>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-700">
        <div className="flex gap-6">
          {(['details', 'events', 'lineage', 'logs', 'agent-calls'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`pb-3 text-sm font-medium capitalize transition-colors ${
                activeTab === tab
                  ? 'text-blue-400 border-b-2 border-blue-400'
                  : 'text-gray-400 hover:text-gray-200'
              }`}
            >
              {tab === 'agent-calls' ? 'Agent Calls' : tab}
            </button>
          ))}
        </div>
      </div>

      {/* Tab Content */}
      {activeTab === 'details' && <DetailsTab task={t} />}
      {activeTab === 'events' && <EventsTab events={events} />}
      {activeTab === 'logs' && <ExecutionSummary execution={execution} />}
      {activeTab === 'lineage' && <LineageTab lineage={lineage} />}
      {activeTab === 'agent-calls' && <AgentCallsTab calls={agentCalls} />}
    </div>
  );
}

const formatJson = (value: unknown) => {
  if (value === undefined || value === null || value === '') return undefined;
  if (typeof value === 'string') {
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  return JSON.stringify(value, null, 2);
};

function TaskStringList({ label, values, color = 'gray' }: { label: string; values?: string[]; color?: 'gray' | 'blue' | 'emerald' }) {
  if (!values || values.length === 0) return null;
  const colorClasses = {
    gray: 'bg-gray-700 text-gray-300',
    blue: 'bg-blue-900/50 text-blue-300',
    emerald: 'bg-emerald-900/50 text-emerald-300',
  };

  return (
    <div className="col-span-2 bg-gray-800 rounded-lg p-3">
      <span className="text-xs text-gray-500">{label}</span>
      <div className="flex flex-wrap gap-2 mt-1">
        {values.map(value => (
          <span key={value} className={`px-2 py-0.5 rounded text-xs ${colorClasses[color]}`}>{value}</span>
        ))}
      </div>
    </div>
  );
}

function JsonBlock({ label, value }: { label: string; value?: unknown }) {
  const formatted = formatJson(value);
  if (!formatted) return null;

  return (
    <div className="col-span-2 bg-gray-800 rounded-lg p-3">
      <span className="text-xs text-gray-500">{label}</span>
      <pre className="mt-2 text-xs text-gray-300 whitespace-pre-wrap bg-gray-900 rounded p-3 max-h-72 overflow-auto">{formatted}</pre>
    </div>
  );
}

function DetailsTab({ task }: { task: ScheduledTask }) {
  const fields: Array<{ label: string; value: string | undefined }> = [
    { label: 'Dispatch Ref', value: task.dispatch_ref },
    { label: 'Wave', value: task.wave != null ? String(task.wave) : undefined },
    { label: 'Topo Rank', value: task.topo_rank != null ? String(task.topo_rank) : undefined },
    { label: 'Source', value: task.source },
    { label: 'Source Ref', value: task.source_ref },
    { label: 'Dispatch Mode', value: task.dispatch_mode },
    { label: 'Auto Dispatch', value: task.auto_dispatch_enabled ? 'Yes' : 'No' },
    { label: 'Dispatch Status', value: task.dispatch_status },
    { label: 'Dispatch Attempts', value: String(task.dispatch_attempts) },
    { label: 'Current Focus', value: task.current_focus },
    { label: 'Next Action', value: task.next_action },
    { label: 'Block Reason', value: task.block_reason },
    { label: 'Failure Code', value: task.failure_code },
    { label: 'Coordination Stage', value: task.coordination_stage },
    { label: 'Auto Repair Count', value: task.auto_repair_count != null ? String(task.auto_repair_count) : undefined },
    { label: 'Created', value: task.created_at ? new Date(task.created_at).toLocaleString() : undefined },
    { label: 'Updated', value: task.updated_at ? new Date(task.updated_at).toLocaleString() : undefined },
    { label: 'Started', value: task.started_at ? new Date(task.started_at).toLocaleString() : undefined },
    { label: 'Last Heartbeat', value: task.last_heartbeat_at ? new Date(task.last_heartbeat_at).toLocaleString() : undefined },
  ];

  return (
    <div className="grid grid-cols-2 gap-4">
      {fields.filter(f => f.value).map(f => (
        <div key={f.label} className="bg-gray-800 rounded-lg p-3">
          <span className="text-xs text-gray-500">{f.label}</span>
          <p className="text-sm text-gray-200 mt-1">{f.value}</p>
        </div>
      ))}

      <TaskStringList label="Dependencies" values={task.depends_on} />
      <TaskStringList label="Files To Read" values={task.files_to_read} color="blue" />
      <TaskStringList label="Files To Modify" values={task.files_to_modify} color="emerald" />
      <TaskStringList label="Input Artifacts" values={task.input_artifacts} color="blue" />
      <TaskStringList label="Output Artifacts" values={task.output_artifacts} color="emerald" />
      <JsonBlock label="Relations" value={task.relations} />
      <JsonBlock label="Context" value={task.context} />
      <JsonBlock label="Raw Task Card" value={task.card_json} />

      {task.acceptance_criteria && task.acceptance_criteria.length > 0 && (
        <div className="col-span-2 bg-gray-800 rounded-lg p-3">
          <span className="text-xs text-gray-500">Acceptance Criteria</span>
          <ul className="mt-1 space-y-1">
            {task.acceptance_criteria.map((c, i) => (
              <li key={i} className="text-sm text-gray-300 flex items-start gap-2">
                <span className="text-gray-600 mt-0.5">•</span> {c}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

// ExecutionTab intentionally removed (unused)

function EventsTab({ events }: { events?: SchedulerEvent[] }) {
  if (!events) return <p className="text-gray-500 text-sm">Loading events...</p>;
  if (events.length === 0) return <p className="text-gray-500 text-sm">No status history or events for this task.</p>;

  return (
    <div className="space-y-2">
      {events.map(event => (
        <div key={event.event_id} className="bg-gray-800 rounded-lg p-4 space-y-2">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="px-2 py-0.5 rounded text-xs font-medium bg-blue-900/60 text-blue-300">{event.event_type}</span>
              {event.from_state && event.to_state && <span className="text-sm text-gray-300">{event.from_state} → {event.to_state}</span>}
            </div>
            <span className="text-xs text-gray-500">{new Date(event.timestamp).toLocaleString()}</span>
          </div>
          <div className="flex flex-wrap gap-3 text-xs text-gray-500">
            {event.dispatch_ref && <span>dispatch_ref: {event.dispatch_ref}</span>}
            {event.runner_id && <span>runner: {event.runner_id}</span>}
            {event.transport && <span>transport: {event.transport}</span>}
            <span>attempt: {event.attempt}</span>
          </div>
          {event.reason && <p className="text-sm text-gray-300">{event.reason}</p>}
          {event.details && <p className="text-sm text-gray-400 whitespace-pre-wrap">{event.details}</p>}
        </div>
      ))}
    </div>
  );
}

function ExecutionSummary({ execution }: { execution?: Awaited<ReturnType<typeof schedulerApi.getTaskExecution>> }) {
  if (execution === undefined) return <p className="text-gray-500 text-sm">Loading execution...</p>;
  if (execution === null) return <p className="text-gray-500 text-sm">No execution log has been recorded for this task.</p>;

  return (
    <div className="bg-gray-800 rounded-lg p-4 space-y-2">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-gray-200">Execution {execution.execution_id}</span>
        <span className="text-xs text-gray-500">{execution.status}</span>
      </div>
      <div className="grid grid-cols-2 gap-2 text-xs text-gray-400">
        <span>Runtime: {execution.runtime}</span>
        <span>Session: {execution.session_id}</span>
        {execution.workspace && <span>Workspace: {execution.workspace}</span>}
        {execution.artifacts && <span>Artifacts: {execution.artifacts}</span>}
      </div>
      {execution.output_tail && <pre className="text-xs text-gray-300 whitespace-pre-wrap bg-gray-900 rounded p-3 max-h-48 overflow-auto">{execution.output_tail}</pre>}
    </div>
  );
}

function LineageTab({ lineage }: { lineage?: TaskLineage }) {
  if (!lineage) return <p className="text-gray-500 text-sm">Loading lineage...</p>;

  return (
    <div className="space-y-4">
      {lineage.root_task && (
        <LineageSection title="Root Task" tasks={[lineage.root_task]} />
      )}
      {lineage.ancestors.length > 0 && (
        <LineageSection title="Ancestors" tasks={lineage.ancestors} />
      )}
      {lineage.siblings.length > 0 && (
        <LineageSection title="Siblings" tasks={lineage.siblings} />
      )}
      {lineage.descendants.length > 0 && (
        <LineageSection title="Descendants" tasks={lineage.descendants} />
      )}
      {lineage.root_task === undefined && lineage.ancestors.length === 0 && lineage.descendants.length === 0 && (
        <p className="text-gray-500 text-sm">No lineage data for this task.</p>
      )}
    </div>
  );
}

function LineageSection({ title, tasks }: { title: string; tasks: ScheduledTask[] }) {
  return (
    <div>
      <h4 className="text-xs font-medium text-gray-500 mb-2">{title}</h4>
      <div className="space-y-1">
        {tasks.map(t => (
          <div key={t.id} className="flex items-center gap-3 px-3 py-2 bg-gray-800 rounded-lg">
            <span className="text-xs text-gray-500">{t.id}</span>
            <span className="text-sm text-gray-200">{t.title}</span>
            <span
              className="px-1.5 py-0.5 rounded text-xs text-white"
              style={{ backgroundColor: STATUS_COLORS[t.status] || '#6b7280' }}
            >
              {t.status}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function AgentCallsTab({ calls }: { calls?: AgentCallRecord[] }) {
  if (!calls) return <p className="text-gray-500 text-sm">Loading agent calls...</p>;
  if (calls.length === 0) return <p className="text-gray-500 text-sm">No agent call history for this task.</p>;

  const statusBadge = (status: string) => {
    const bg = status === 'success' ? 'bg-green-900/60 text-green-300' : 'bg-red-900/60 text-red-300';
    return <span className={`px-2 py-0.5 rounded text-xs font-medium ${bg}`}>{status}</span>;
  };

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  return (
    <div className="space-y-2">
      {calls.map(call => (
        <div key={call.id} className="bg-gray-800 rounded-lg p-4 space-y-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {statusBadge(call.status)}
              <span className="text-sm font-medium text-gray-200" style={{ color: AGENT_COLORS[call.agent_id] || '#9ca3af' }}>
                {call.agent_id}
              </span>
              <span className="text-xs text-gray-500">{call.runner_type}</span>
              <span className="text-xs text-gray-500">{call.task_type}</span>
            </div>
            <div className="flex items-center gap-3 text-xs text-gray-500">
              <span>{formatDuration(call.duration_ms)}</span>
              <span>exit: {call.exit_code}</span>
            </div>
          </div>
          <div className="flex items-center gap-4 text-xs text-gray-500">
            <span>{new Date(call.started_at).toLocaleString()}</span>
            <span>→</span>
            <span>{new Date(call.finished_at).toLocaleString()}</span>
            {call.trace_id && <span className="text-gray-600">trace: {call.trace_id.slice(0, 8)}...</span>}
          </div>
          {call.output_summary && (
            <p className="text-sm text-gray-400 whitespace-pre-wrap line-clamp-3">{call.output_summary}</p>
          )}
          {call.error_message && (
            <p className="text-sm text-red-300 whitespace-pre-wrap font-mono line-clamp-3">{call.error_message}</p>
          )}
        </div>
      ))}
    </div>
  );
}


