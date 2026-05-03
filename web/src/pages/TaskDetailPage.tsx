import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { schedulerApi } from '../api/schedulerApi';
import { useWebSocket } from '../hooks/useWebSocket';
import LogViewer from '../components/LogViewer';
import { useProject } from '../hooks/useProject';
import type { ScheduledTask, TaskExecution, TaskLineage, WSMessage } from '../types/scheduler';
import { TaskStatus, DispatchStatus, Agent } from '../types/scheduler';

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

  const [activeTab, setActiveTab] = useState<'details' | 'execution' | 'lineage' | 'logs'>('details');

  const { data: task, isLoading, error } = useQuery({
    queryKey: ['task', projectId, taskId],
    queryFn: () => schedulerApi.getTaskExecution(projectId, taskId!).then(() =>
      fetchTask()
    ),
    enabled: !!taskId && !!projectId,
  });

  const [taskData, setTaskData] = useState<ScheduledTask | null>(null);

  async function fetchTask() {
    const tasks = await schedulerApi.getTasks(projectId);
    const found = tasks.find(t => t.id === taskId);
    setTaskData(found || null);
    return found;
  }

  useEffect(() => {
    if (projectId && taskId) fetchTask();
  }, [projectId, taskId]);

  // Real-time updates via WebSocket
  useEffect(() => {
    if (!lastMessage || !taskId) return;
    if (lastMessage.task_id === taskId || lastMessage.type === 'task.state_changed') {
      fetchTask();
      qc.invalidateQueries({ queryKey: ['task', projectId, taskId] });
    }
  }, [lastMessage, taskId, projectId, qc]);

  const { data: execution } = useQuery({
    queryKey: ['execution', projectId, taskId],
    queryFn: () => schedulerApi.getTaskExecution(projectId, taskId!),
    enabled: !!taskId && !!projectId,
  });

  const { data: lineage } = useQuery({
    queryKey: ['lineage', projectId, taskId],
    queryFn: () => schedulerApi.getTaskLineage(projectId, taskId!),
    enabled: !!taskId && !!projectId && activeTab === 'lineage',
  });

  const retryMutation = useMutation({
    mutationFn: () => schedulerApi.retryTask(projectId, taskId!),
    onSuccess: () => { fetchTask(); qc.invalidateQueries({ queryKey: ['task', projectId, taskId] }); },
  });

  const dispatchMutation = useMutation({
    mutationFn: () => schedulerApi.dispatchTask(projectId, taskId!),
    onSuccess: () => { fetchTask(); qc.invalidateQueries({ queryKey: ['task', projectId, taskId] }); },
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
          {(['details', 'execution', 'lineage', 'logs'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`pb-3 text-sm font-medium capitalize transition-colors ${
                activeTab === tab
                  ? 'text-blue-400 border-b-2 border-blue-400'
                  : 'text-gray-400 hover:text-gray-200'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>
      </div>

      {/* Tab Content */}
      {activeTab === 'details' && <DetailsTab task={t} />}
      {activeTab === 'logs' && (
        <LogViewer projectId={projectId} taskId={taskId} maxLines={500} height="500px" />
      )}
      {activeTab === 'lineage' && <LineageTab lineage={lineage} />}
    </div>
  );
}

function DetailsTab({ task }: { task: ScheduledTask }) {
  const fields: Array<{ label: string; value: string | undefined }> = [
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

      {task.depends_on && task.depends_on.length > 0 && (
        <div className="col-span-2 bg-gray-800 rounded-lg p-3">
          <span className="text-xs text-gray-500">Dependencies</span>
          <div className="flex flex-wrap gap-2 mt-1">
            {task.depends_on.map(d => (
              <span key={d} className="px-2 py-0.5 bg-gray-700 rounded text-xs text-gray-300">{d}</span>
            ))}
          </div>
        </div>
      )}

      {task.output_artifacts && task.output_artifacts.length > 0 && (
        <div className="col-span-2 bg-gray-800 rounded-lg p-3">
          <span className="text-xs text-gray-500">Output Artifacts</span>
          <div className="flex flex-wrap gap-2 mt-1">
            {task.output_artifacts.map(a => (
              <span key={a} className="px-2 py-0.5 bg-emerald-900/50 rounded text-xs text-emerald-300">{a}</span>
            ))}
          </div>
        </div>
      )}

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

function ExecutionTab({ execution }: { execution?: TaskExecution }) {
  if (!execution) return <p className="text-gray-500 text-sm">No execution data available.</p>;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <InfoField label="Execution ID" value={execution.execution_id} />
        <InfoField label="Runtime" value={execution.runtime} />
        <InfoField label="Session ID" value={execution.session_id} />
        <InfoField label="Status" value={execution.status} />
        <InfoField label="Started" value={new Date(execution.started_at).toLocaleString()} />
        {execution.completed_at && <InfoField label="Completed" value={new Date(execution.completed_at).toLocaleString()} />}
      </div>

      {execution.command && (
        <div className="bg-gray-800 rounded-lg p-3">
          <span className="text-xs text-gray-500">Command</span>
          <pre className="text-sm text-green-300 mt-1 overflow-x-auto">{execution.command}</pre>
        </div>
      )}

      {execution.output_tail && (
        <div className="bg-gray-900 rounded-lg p-3">
          <span className="text-xs text-gray-500">Output (tail)</span>
          <pre className="text-xs text-gray-300 mt-1 overflow-x-auto max-h-64 whitespace-pre-wrap">{execution.output_tail}</pre>
        </div>
      )}

      {execution.events && execution.events.length > 0 && (
        <div className="bg-gray-800 rounded-lg p-3">
          <span className="text-xs text-gray-500 mb-2 block">Events</span>
          <div className="space-y-2">
            {execution.events.map((ev, i) => (
              <div key={i} className="flex items-center gap-3 text-xs">
                <span className="text-gray-600 w-36 shrink-0">{new Date(ev.timestamp).toLocaleTimeString()}</span>
                <span className="px-1.5 py-0.5 rounded bg-gray-700 text-gray-300">{ev.event}</span>
                <span className="text-gray-400 truncate">{ev.message}</span>
              </div>
            ))}
          </div>
        </div>
      )}
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

function InfoField({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-gray-800 rounded-lg p-3">
      <span className="text-xs text-gray-500">{label}</span>
      <p className="text-sm text-gray-200 mt-1 break-all">{value}</p>
    </div>
  );
}
