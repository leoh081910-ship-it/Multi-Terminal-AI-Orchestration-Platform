import React, { useState, useCallback, useEffect, useRef } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import {
  AlertTriangle, CheckCircle, RefreshCw, XCircle,
  ChevronDown, ChevronRight, FileText, GitBranch,
  Clock, GitCommit, Wrench, Search as SearchIcon,
} from 'lucide-react';
import { schedulerApi } from '../api/schedulerApi';
import { useProject } from '../hooks/useProject';
import type { TriageTask, TriageBatchResult } from '../types/scheduler';

/* ── Styles ── */

const card: React.CSSProperties = {
  background: 'var(--bg-card, rgba(255,255,255,0.03))',
  border: '1px solid var(--border-color, rgba(255,255,255,0.08))',
  borderRadius: 12,
  padding: '1.2rem',
};

const btn: React.CSSProperties = {
  display: 'inline-flex', alignItems: 'center', gap: 6,
  padding: '0.5rem 1rem', borderRadius: 8, border: 'none',
  cursor: 'pointer', fontWeight: 600, fontSize: '0.85rem',
  transition: 'opacity 0.15s',
};
const btnDisabled: React.CSSProperties = { ...btn, opacity: 0.5, cursor: 'not-allowed' };
const btnGreen = (disabled: boolean): React.CSSProperties => disabled ? { ...btnDisabled, background: '#22c55e', color: '#fff' } : { ...btn, background: '#22c55e', color: '#fff' };
const btnBlue = (disabled: boolean): React.CSSProperties => disabled ? { ...btnDisabled, background: '#3b82f6', color: '#fff' } : { ...btn, background: '#3b82f6', color: '#fff' };
const btnGray = (disabled: boolean): React.CSSProperties => disabled ? { ...btnDisabled, background: '#6b7280', color: '#fff' } : { ...btn, background: '#6b7280', color: '#fff' };
const btnGhost: React.CSSProperties = { ...btn, background: 'rgba(255,255,255,0.08)', color: '#ccc' };

const badge = (state: string): React.CSSProperties => {
  const c: Record<string, string> = { blocked: '#ef4444', triage: '#f59e0b', verify_failed: '#f97316' };
  return {
    display: 'inline-block', padding: '0.2rem 0.6rem', borderRadius: 6,
    fontSize: '0.75rem', fontWeight: 600, background: c[state] || '#6b7280', color: '#fff',
  };
};

/* ── Toast System ── */

interface Toast {
  id: number;
  type: 'success' | 'error' | 'info';
  message: string;
}

let toastIdCounter = 0;

function ToastContainer({ toasts, onDismiss }: { toasts: Toast[]; onDismiss: (id: number) => void }) {
  return (
    <div style={{ position: 'fixed', bottom: 20, right: 20, zIndex: 1000, display: 'flex', flexDirection: 'column', gap: 8 }}>
      {toasts.map(t => (
        <div key={t.id} onClick={() => onDismiss(t.id)} style={{
          padding: '0.7rem 1.2rem', borderRadius: 8, cursor: 'pointer',
          background: t.type === 'success' ? '#22c55e' : t.type === 'error' ? '#ef4444' : '#3b82f6',
          color: '#fff', fontSize: '0.85rem', fontWeight: 600,
          boxShadow: '0 4px 12px rgba(0,0,0,0.3)', maxWidth: 400,
          animation: 'fadeIn 0.2s ease-out',
        }}>
          {t.message}
        </div>
      ))}
    </div>
  );
}

function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timers = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map());

  const addToast = useCallback((type: Toast['type'], message: string) => {
    const id = ++toastIdCounter;
    setToasts(prev => [...prev, { id, type, message }]);
    const timer = setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
      timers.current.delete(id);
    }, 4000);
    timers.current.set(id, timer);
  }, []);

  const dismiss = useCallback((id: number) => {
    setToasts(prev => prev.filter(t => t.id !== id));
    const timer = timers.current.get(id);
    if (timer) { clearTimeout(timer); timers.current.delete(id); }
  }, []);

  return { toasts, addToast, dismiss };
}

/* ── Main Page ── */

export default function TriageDashboardPage() {
  const { projectId } = useProject();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [timelineTaskId, setTimelineTaskId] = useState<string | null>(null);
  const [batchResults, setBatchResults] = useState<TriageBatchResult[] | null>(null);
  const { toasts, addToast, dismiss } = useToast();

  const { data: tasks, isLoading, error } = useQuery({
    queryKey: ['triage-summary', projectId],
    queryFn: () => schedulerApi.getTriageSummary(projectId!),
    enabled: !!projectId,
    refetchInterval: 10000,
  });

  // Auto-clean selected when tasks change (removed tasks no longer exist)
  useEffect(() => {
    if (!tasks) return;
    const validIds = new Set(tasks.map(t => t.id));
    setSelected(prev => {
      const next = new Set([...prev].filter(id => validIds.has(id)));
      return next.size !== prev.size ? next : prev;
    });
  }, [tasks]);

  const approveMut = useMutation({
    mutationFn: (id: string) => schedulerApi.triageApprove(projectId!, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['triage-summary'] });
      setSelected(new Set());
      addToast('success', 'Task approved');
    },
    onError: (err) => addToast('error', `Failed to approve: ${(err as Error).message}`),
  });
  const retryMut = useMutation({
    mutationFn: (id: string) => schedulerApi.triageRetry(projectId!, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['triage-summary'] });
      setSelected(new Set());
      addToast('success', 'Task queued for retry');
    },
    onError: (err) => addToast('error', `Failed to retry: ${(err as Error).message}`),
  });
  const wontfixMut = useMutation({
    mutationFn: (id: string) => schedulerApi.triageWonFix(projectId!, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['triage-summary'] });
      setSelected(new Set());
      addToast('info', "Marked as won't fix");
    },
    onError: (err) => addToast('error', `Failed: ${(err as Error).message}`),
  });
  const batchMut = useMutation({
    mutationFn: ({ ids, action }: { ids: string[]; action: 'approve' | 'retry' | 'wontfix' }) =>
      schedulerApi.triageBatch(projectId!, ids, action),
    onSuccess: (results, vars) => {
      queryClient.invalidateQueries({ queryKey: ['triage-summary'] });
      setSelected(new Set());
      setBatchResults(results);
      const okCount = results.filter(r => r.status === 'ok').length;
      const errCount = results.length - okCount;
      if (errCount === 0) {
        addToast('success', `Batch ${vars.action}: ${okCount}/${results.length} succeeded`);
      } else {
        addToast('error', `Batch ${vars.action}: ${okCount} ok, ${errCount} failed`);
      }
    },
    onError: (err) => addToast('error', `Batch failed: ${(err as Error).message}`),
  });

  const toggleSelect = useCallback((id: string) => {
    setSelected(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }, []);

  const toggleAll = useCallback(() => {
    if (!tasks) return;
    setSelected(prev =>
      prev.size === tasks.length ? new Set() : new Set(tasks.map(t => t.id))
    );
  }, [tasks]);

  const anyLoading = approveMut.isPending || retryMut.isPending || wontfixMut.isPending || batchMut.isPending;

  if (!projectId) return <div style={{ padding: 40, color: '#888' }}>选择一个项目以查看 Triage 面板</div>;
  if (isLoading) return <div style={{ padding: 40, color: '#888' }}>加载中...</div>;
  if (error) return <div style={{ padding: 40, color: '#ef4444' }}>加载失败: {(error as Error).message}</div>;

  const triageTasks = tasks || [];
  const allSelected = selected.size > 0 && selected.size === triageTasks.length;

  return (
    <div style={{ maxWidth: 1200, margin: '0 auto', padding: '2rem' }}>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />

      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: '1.5rem' }}>
        <AlertTriangle size={28} color="#f59e0b" />
        <h1 style={{ margin: 0, fontSize: '1.5rem', fontWeight: 700 }}>Triage Dashboard</h1>
        <span style={{ ...badge('blocked'), marginLeft: 'auto' }}>{triageTasks.length} 待处理</span>
      </div>

      {/* Batch result detail panel */}
      {batchResults && (
        <div style={{ ...card, marginBottom: 16, background: 'rgba(255,255,255,0.02)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <span style={{ fontWeight: 600, fontSize: '0.9rem' }}>批量操作结果</span>
            <button style={{ ...btnGhost, padding: '0.2rem 0.5rem', fontSize: '0.75rem' }}
              onClick={() => setBatchResults(null)}>✕</button>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {batchResults.map(r => (
              <div key={r.task_id} style={{ display: 'flex', gap: 8, fontSize: '0.8rem', alignItems: 'center' }}>
                {r.status === 'ok'
                  ? <CheckCircle size={14} color="#22c55e" />
                  : <XCircle size={14} color="#ef4444" />}
                <code style={{ color: '#60a5fa' }}>{r.task_id}</code>
                <span style={{ color: r.status === 'ok' ? '#22c55e' : '#ef4444' }}>{r.status}</span>
                {r.detail && <span style={{ color: '#888' }}>— {r.detail}</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Batch action bar */}
      {selected.size > 0 && (
        <div data-testid="triage-batch-bar" style={{
          ...card, marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12,
          background: 'rgba(59,130,246,0.08)', borderColor: 'rgba(59,130,246,0.2)',
        }}>
          <span style={{ fontWeight: 600 }}>已选 {selected.size} 项</span>
          <button style={btnGreen(batchMut.isPending)} disabled={batchMut.isPending}
            onClick={() => batchMut.mutate({ ids: [...selected], action: 'approve' })}>
            <CheckCircle size={16} /> 批量 Approve
          </button>
          <button style={btnBlue(batchMut.isPending)} disabled={batchMut.isPending}
            onClick={() => batchMut.mutate({ ids: [...selected], action: 'retry' })}>
            <RefreshCw size={16} /> 批量 Retry
          </button>
          <button style={btnGray(batchMut.isPending)} disabled={batchMut.isPending}
            onClick={() => batchMut.mutate({ ids: [...selected], action: 'wontfix' })}>
            <XCircle size={16} /> 批量 Won't Fix
          </button>
          <button style={{ ...btnGhost, marginLeft: 'auto' }} onClick={() => setSelected(new Set())}>
            取消选择
          </button>
        </div>
      )}

      {/* Timeline panel */}
      {timelineTaskId && projectId && (
        <TimelinePanel
          projectId={projectId}
          taskId={timelineTaskId}
          onClose={() => setTimelineTaskId(null)}
        />
      )}

      {triageTasks.length === 0 ? (
        <div data-testid="triage-empty-state" style={{ ...card, textAlign: 'center', padding: '3rem', color: '#888' }}>
          <CheckCircle size={48} color="#22c55e" style={{ marginBottom: 12 }} />
          <p style={{ fontSize: '1.1rem', margin: 0 }}>没有需要人工介入的任务</p>
          <p style={{ fontSize: '0.85rem', marginTop: 8 }}>所有任务都在自动处理中</p>
        </div>
      ) : (
        <>
          {/* Select all header */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, padding: '0 4px' }}>
            <input
              data-testid="triage-select-all"
              type="checkbox"
              checked={allSelected}
              onChange={toggleAll}
              style={{ cursor: 'pointer', accentColor: '#3b82f6' }}
            />
            <span style={{ fontSize: '0.8rem', color: '#888' }}>全选</span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {triageTasks.map(task => (
              <TriageCard
                key={task.id}
                task={task}
                selected={selected.has(task.id)}
                expanded={expandedId === task.id}
                onToggleSelect={() => toggleSelect(task.id)}
                onToggleExpand={() => setExpandedId(expandedId === task.id ? null : task.id)}
                onApprove={() => approveMut.mutate(task.id)}
                onRetry={() => retryMut.mutate(task.id)}
                onWonFix={() => wontfixMut.mutate(task.id)}
                onViewDetail={() => navigate(`/tasks/${task.id}`)}
                onViewTimeline={() => setTimelineTaskId(task.id)}
                actionLoading={anyLoading}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

/* ── Triage Card ── */

interface CardProps {
  task: TriageTask;
  selected: boolean;
  expanded: boolean;
  onToggleSelect: () => void;
  onToggleExpand: () => void;
  onApprove: () => void;
  onRetry: () => void;
  onWonFix: () => void;
  onViewDetail: () => void;
  onViewTimeline: () => void;
  actionLoading: boolean;
}

function TriageCard({
  task, selected, expanded, onToggleSelect, onToggleExpand,
  onApprove, onRetry, onWonFix, onViewDetail, onViewTimeline, actionLoading,
}: CardProps) {
  return (
    <div data-testid="triage-task-card" style={{ ...card, borderColor: selected ? 'rgba(59,130,246,0.4)' : undefined }}>
      {/* Header row */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <input
          data-testid="triage-task-checkbox"
          type="checkbox"
          checked={selected}
          onChange={onToggleSelect}
          style={{ cursor: 'pointer', accentColor: '#3b82f6' }}
        />
        <div data-testid="triage-task-expand" style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, flex: 1 }}
          onClick={onToggleExpand}>
          {expanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          <span style={badge(task.status)}>{task.status}</span>
          <span style={{ fontWeight: 600, flex: 1 }}>{task.title}</span>
        </div>
        <span style={{ fontSize: '0.8rem', color: '#888', fontFamily: 'monospace' }}>{task.id}</span>
        {task.rework_count != null && task.rework_count > 0 && (
          <span style={{ fontSize: '0.75rem', color: '#f59e0b' }}>rework #{task.rework_count}</span>
        )}
      </div>

      {/* Expanded details */}
      {expanded && (
        <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid rgba(255,255,255,0.08)' }}>
          {/* Info grid */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
            <InfoField label="Review Decision" value={task.review_decision || 'N/A'}
              color={task.review_decision === 'rejected' ? '#ef4444' : '#22c55e'} />
            <InfoField label="Coordination Stage" value={task.coordination_stage || 'N/A'} />
            <InfoField label="Escalation Reason" value={task.escalation_reason || task.block_reason || 'N/A'} />
            <InfoField label="Last Rejection" value={task.last_rejection_reason || 'N/A'} />
          </div>

          {/* Result summary */}
          {task.result_summary && (
            <div style={{ marginBottom: 16 }}>
              <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 4 }}>
                <FileText size={14} style={{ verticalAlign: 'middle', marginRight: 4 }} />
                Review Summary
              </div>
              <div style={{
                ...card, background: '#05070a', fontSize: '0.85rem', lineHeight: 1.6,
                maxHeight: 200, overflow: 'auto', whiteSpace: 'pre-wrap',
              }}>{task.result_summary}</div>
            </div>
          )}

          {/* Lineage */}
          <div style={{ marginBottom: 16, display: 'flex', gap: 16, fontSize: '0.85rem', flexWrap: 'wrap' }}>
            {task.parent_task_id && (
              <span><GitBranch size={14} style={{ verticalAlign: 'middle', marginRight: 4 }} />
                Parent: <code style={{ color: '#60a5fa' }}>{task.parent_task_id}</code></span>
            )}
            {task.root_task_id && task.root_task_id !== task.id && (
              <span>Root: <code style={{ color: '#60a5fa' }}>{task.root_task_id}</code></span>
            )}
            {task.failure_code && (
              <span>Failure: <code style={{ color: '#f59e0b' }}>{task.failure_code}</code></span>
            )}
          </div>

          {/* Actions */}
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <button style={btnGreen(actionLoading)} onClick={onApprove} disabled={actionLoading}>
              <CheckCircle size={16} /> Approve Manually
            </button>
            <button style={btnBlue(actionLoading)} onClick={onRetry} disabled={actionLoading}>
              <RefreshCw size={16} /> Retry
            </button>
            <button style={btnGray(actionLoading)} onClick={onWonFix} disabled={actionLoading}>
              <XCircle size={16} /> Won't Fix
            </button>
            <button style={btnGhost} onClick={onViewTimeline}>
              <Clock size={16} /> Timeline
            </button>
            <button style={btnGhost} onClick={onViewDetail}>View Detail →</button>
          </div>
        </div>
      )}
    </div>
  );
}

function InfoField({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div>
      <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 4 }}>{label}</div>
      <div style={{ color, fontWeight: color ? 600 : undefined }}>{value}</div>
    </div>
  );
}

/* ── Timeline Panel ── */

function TimelinePanel({ projectId, taskId, onClose }: {
  projectId: string;
  taskId: string;
  onClose: () => void;
}) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['triage-lineage', projectId, taskId],
    queryFn: () => schedulerApi.getTriageLineage(projectId, taskId),
    enabled: !!projectId && !!taskId,
  });

  const iconForType = (t: string) => {
    switch (t) {
      case 'review': return <SearchIcon size={14} color="#3b82f6" />;
      case 'rework': return <Wrench size={14} color="#f59e0b" />;
      case 'triage': return <AlertTriangle size={14} color="#ef4444" />;
      default: return <GitCommit size={14} color="#22c55e" />;
    }
  };

  return (
    <div style={{ ...card, marginBottom: 16, background: 'rgba(255,255,255,0.02)' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
        <h3 style={{ margin: 0, fontSize: '1rem' }}>
          <Clock size={18} style={{ verticalAlign: 'middle', marginRight: 8 }} />
          Rework Timeline — {taskId}
        </h3>
        <button style={{ ...btnGhost, padding: '0.3rem 0.6rem' }} onClick={onClose}>✕</button>
      </div>

      {isLoading ? (
        <div style={{ color: '#888', padding: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
          <RefreshCw size={16} className="animate-spin" />
          Loading timeline...
        </div>
      ) : error ? (
        <div style={{ color: '#ef4444', padding: 12 }}>
          Failed to load timeline: {(error as Error).message}
        </div>
      ) : !data || data.timeline.length === 0 ? (
        <div style={{ color: '#888', padding: 12, textAlign: 'center' }}>
          No timeline data available for this task.
        </div>
      ) : (
        <div style={{ position: 'relative', paddingLeft: 28 }}>
          {/* Vertical line */}
          <div style={{
            position: 'absolute', left: 7, top: 8, bottom: 8, width: 2,
            background: 'rgba(255,255,255,0.1)',
          }} />

          {data.timeline.map((entry, i) => (
            <div key={entry.task_id + i} style={{
              position: 'relative', marginBottom: i < data.timeline.length - 1 ? 16 : 0,
            }}>
              {/* Dot */}
              <div style={{
                position: 'absolute', left: -28, top: 4,
                width: 14, height: 14, borderRadius: '50%',
                background: entry.type === 'original' ? '#22c55e'
                  : entry.type === 'review' ? '#3b82f6'
                  : entry.type === 'rework' ? '#f59e0b' : '#ef4444',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                {iconForType(entry.type)}
              </div>

              {/* Content */}
              <div style={{ fontSize: '0.85rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                  <code style={{ color: '#60a5fa', fontSize: '0.8rem' }}>{entry.task_id}</code>
                  <span style={badge(entry.state)}>{entry.state}</span>
                  {entry.decision && (
                    <span style={{
                      fontSize: '0.75rem', fontWeight: 600,
                      color: entry.decision === 'approved' ? '#22c55e' : '#ef4444',
                    }}>{entry.decision}</span>
                  )}
                </div>
                {entry.summary && (
                  <div style={{ color: '#aaa', fontSize: '0.8rem', lineHeight: 1.4, marginTop: 4 }}>
                    {entry.summary.length > 200 ? entry.summary.slice(0, 200) + '...' : entry.summary}
                  </div>
                )}
                <div style={{ color: '#555', fontSize: '0.7rem', marginTop: 4 }}>
                  {entry.created_at}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
