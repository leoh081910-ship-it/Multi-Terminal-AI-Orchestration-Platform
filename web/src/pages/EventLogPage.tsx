import React, { useEffect, useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { Activity, AlertTriangle, Radio, RefreshCw } from 'lucide-react';
import { schedulerApi } from '../api/schedulerApi';
import { useProject } from '../hooks/useProject';
import { useWebSocket, type WSMessage } from '../hooks/useWebSocket';
import type { SchedulerEvent } from '../types/scheduler';

const panelStyle: React.CSSProperties = {
  display: 'grid',
  gap: '0.75rem',
  padding: '1rem',
  background: 'rgba(255,255,255,0.03)',
  border: '1px solid rgba(255,255,255,0.06)',
  borderRadius: 12,
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  background: 'rgba(255,255,255,0.04)',
  border: '1px solid var(--border-color)',
  borderRadius: 8,
  color: 'white',
  padding: '0.65rem 0.75rem',
};

const buttonStyle: React.CSSProperties = {
  borderRadius: 8,
  border: '1px solid var(--border-color)',
  background: 'rgba(255,255,255,0.04)',
  color: 'white',
  padding: '0.55rem 0.75rem',
  cursor: 'pointer',
  fontWeight: 700,
};

const formatTime = (value: string) => new Date(value).toLocaleString();

const EventLogPage: React.FC = () => {
  const { projectId } = useProject();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [taskFilter, setTaskFilter] = useState(searchParams.get('task_id') ?? '');
  const [dispatchFilter, setDispatchFilter] = useState(searchParams.get('dispatch_ref') ?? '');
  const [realtimeEvents, setRealtimeEvents] = useState<SchedulerEvent[]>([]);
  const { connected, lastMessage } = useWebSocket(projectId);

  useEffect(() => {
    setTaskFilter(searchParams.get('task_id') ?? '');
    setDispatchFilter(searchParams.get('dispatch_ref') ?? '');
  }, [searchParams]);

  const filters = useMemo(() => ({
    taskId: taskFilter.trim() || undefined,
    dispatchRef: dispatchFilter.trim() || undefined,
  }), [dispatchFilter, taskFilter]);

  const eventsQuery = useQuery({
    queryKey: ['scheduler-events', projectId, filters.taskId ?? '', filters.dispatchRef ?? ''],
    queryFn: () => schedulerApi.getEvents(projectId, filters),
    enabled: !!projectId,
    refetchInterval: connected ? false : 15_000,
  });

  useEffect(() => {
    setRealtimeEvents([]);
  }, [projectId, filters.taskId, filters.dispatchRef]);

  useEffect(() => {
    const event = websocketMessageToEvent(lastMessage, projectId);
    if (!event) return;
    if (filters.taskId && event.task_id !== filters.taskId) return;
    if (filters.dispatchRef && event.dispatch_ref !== filters.dispatchRef) return;

    queryClient.invalidateQueries({ queryKey: ['scheduler-events', projectId] });
    if (event.task_id) {
      queryClient.invalidateQueries({ queryKey: ['task-events', projectId, event.task_id] });
      queryClient.invalidateQueries({ queryKey: ['task', projectId, event.task_id] });
    }

    setRealtimeEvents((current) => {
      if (current.some((item) => item.event_id === event.event_id)) return current;
      return [event, ...current].slice(0, 200);
    });
  }, [filters.dispatchRef, filters.taskId, lastMessage, projectId, queryClient]);

  const events = useMemo(() => {
    const merged = new Map<string, SchedulerEvent>();
    for (const event of realtimeEvents) merged.set(event.event_id, event);
    for (const event of eventsQuery.data ?? []) merged.set(event.event_id, event);
    return Array.from(merged.values()).sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }, [eventsQuery.data, realtimeEvents]);

  const applyFilters = () => {
    const next = new URLSearchParams();
    if (taskFilter.trim()) next.set('task_id', taskFilter.trim());
    if (dispatchFilter.trim()) next.set('dispatch_ref', dispatchFilter.trim());
    setSearchParams(next);
  };

  const clearFilters = () => {
    setTaskFilter('');
    setDispatchFilter('');
    setSearchParams(new URLSearchParams());
  };

  return (
    <div style={{ display: 'grid', gap: '1.5rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: '1rem' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--accent-cyan)', marginBottom: '0.5rem' }}>
            <Activity size={16} />
            <span style={{ fontSize: '0.75rem', fontWeight: 700, letterSpacing: '2px' }}>EVENT BROWSER</span>
          </div>
          <h2 className="neon-text" style={{ fontSize: '2.2rem' }}>事件日志</h2>
          <p style={{ color: 'var(--text-secondary)' }}>按 task 或 dispatch_ref 查询历史事件，并通过 WebSocket 实时追加最新事件。</p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: connected ? 'var(--accent-green)' : 'var(--text-secondary)' }}>
          <Radio size={16} />
          <span style={{ fontSize: '0.78rem', fontWeight: 700 }}>{connected ? '实时连接中' : '实时未连接'}</span>
        </div>
      </div>

      <div className="glass-card" style={{ display: 'grid', gap: '1rem' }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '0.75rem', alignItems: 'end' }}>
          <label style={{ display: 'grid', gap: '0.35rem', color: 'var(--text-secondary)', fontSize: '0.78rem' }}>
            Task ID
            <input value={taskFilter} onChange={(event) => setTaskFilter(event.target.value)} placeholder="TASK-001" style={inputStyle} />
          </label>
          <label style={{ display: 'grid', gap: '0.35rem', color: 'var(--text-secondary)', fontSize: '0.78rem' }}>
            Dispatch Ref
            <input value={dispatchFilter} onChange={(event) => setDispatchFilter(event.target.value)} placeholder="dispatch-2026..." style={inputStyle} />
          </label>
          <div style={{ display: 'flex', gap: '0.75rem' }}>
            <button type="button" onClick={applyFilters} style={{ ...buttonStyle, background: 'var(--accent-cyan)', color: '#071014' }}>应用过滤</button>
            <button type="button" onClick={clearFilters} style={buttonStyle}>清空</button>
            <button type="button" onClick={() => eventsQuery.refetch()} disabled={eventsQuery.isFetching} style={buttonStyle}>
              <RefreshCw size={14} style={{ verticalAlign: 'middle', marginRight: 6 }} />
              刷新
            </button>
          </div>
        </div>
      </div>

      <div className="glass-card" style={{ display: 'grid', gap: '1rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '1rem' }}>
          <strong>事件链</strong>
          <span style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>{events.length} events · {projectId || '未选择项目'}</span>
        </div>

        {eventsQuery.isLoading && <p style={{ color: 'var(--text-secondary)' }}>加载历史事件中…</p>}
        {eventsQuery.error && <ErrorLine message="事件历史加载失败" />}
        {!eventsQuery.isLoading && !events.length && <p style={{ color: 'var(--text-secondary)' }}>没有匹配当前过滤条件的事件。</p>}

        <div style={{ display: 'grid', gap: '0.75rem' }}>
          {events.map((event) => (
            <div key={event.event_id} style={panelStyle}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: '1rem', alignItems: 'flex-start' }}>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: '0.5rem' }}>
                    <strong>{event.event_type}</strong>
                    {event.from_state || event.to_state ? (
                      <span style={{ color: 'var(--text-secondary)' }}>{event.from_state || '—'} → {event.to_state || '—'}</span>
                    ) : null}
                  </div>
                  <div className="mono" style={{ color: 'var(--text-secondary)', fontSize: '0.78rem', marginTop: '0.3rem' }}>{event.event_id}</div>
                </div>
                <span style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>{formatTime(event.timestamp)}</span>
              </div>

              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', color: 'var(--text-secondary)', fontSize: '0.82rem' }}>
                <Link to={`/tasks/${encodeURIComponent(event.task_id)}`} style={{ color: 'var(--accent-cyan)' }}>{event.task_id}</Link>
                {event.dispatch_ref && <Link to={`/events?dispatch_ref=${encodeURIComponent(event.dispatch_ref)}`} style={{ color: 'var(--accent-cyan)' }}>{event.dispatch_ref}</Link>}
                <span>attempt {event.attempt}</span>
                {event.transport && <span>{event.transport}</span>}
                {event.runner_id && <span>{event.runner_id}</span>}
              </div>

              {event.reason && <p style={{ color: 'var(--text-secondary)' }}>{event.reason}</p>}
              {event.details && <pre style={{ color: 'var(--text-secondary)', whiteSpace: 'pre-wrap', fontFamily: 'var(--font-mono)', fontSize: '0.78rem' }}>{event.details}</pre>}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

function websocketMessageToEvent(message: WSMessage | null, projectId: string): SchedulerEvent | null {
  if (!message || !message.task_id) return null;
  if (message.project_id && message.project_id !== projectId) return null;

  const data = typeof message.data === 'object' && message.data !== null ? message.data as Record<string, unknown> : {};
  const eventType = stringValue(data.event_type) || message.type;
  const timestamp = stringValue(data.timestamp) || message.timestamp;

  return {
    event_id: stringValue(data.event_id) || `${message.type}:${message.task_id}:${timestamp}`,
    project_id: message.project_id || projectId,
    task_id: message.task_id,
    dispatch_ref: stringValue(data.dispatch_ref),
    event_type: eventType,
    from_state: stringValue(data.from_state),
    to_state: stringValue(data.to_state),
    timestamp,
    reason: stringValue(data.reason),
    attempt: numberValue(data.attempt),
    transport: stringValue(data.transport),
    runner_id: stringValue(data.runner_id),
    details: stringValue(data.details),
  };
}

function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function numberValue(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function ErrorLine({ message }: { message: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--accent-magenta)' }}>
      <AlertTriangle size={16} />
      <span>{message}</span>
    </div>
  );
}

export default EventLogPage;
