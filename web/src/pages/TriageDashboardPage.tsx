import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { AlertTriangle, CheckCircle, RefreshCw, XCircle, ChevronDown, ChevronRight, FileText, GitBranch } from 'lucide-react';
import { schedulerApi } from '../api/schedulerApi';
import { useProject } from '../hooks/useProject';
import type { TriageTask } from '../types/scheduler';

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card, rgba(255,255,255,0.03))',
  border: '1px solid var(--border-color, rgba(255,255,255,0.08))',
  borderRadius: 12,
  padding: '1.2rem',
};

const btnBase: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 6,
  padding: '0.5rem 1rem',
  borderRadius: 8,
  border: 'none',
  cursor: 'pointer',
  fontWeight: 600,
  fontSize: '0.85rem',
  transition: 'opacity 0.15s',
};

const btnApprove: React.CSSProperties = { ...btnBase, background: '#22c55e', color: '#fff' };
const btnRetry: React.CSSProperties = { ...btnBase, background: '#3b82f6', color: '#fff' };
const btnWonFix: React.CSSProperties = { ...btnBase, background: '#6b7280', color: '#fff' };

const stateBadge = (state: string): React.CSSProperties => {
  const colors: Record<string, string> = {
    blocked: '#ef4444',
    triage: '#f59e0b',
    verify_failed: '#f97316',
  };
  return {
    display: 'inline-block',
    padding: '0.2rem 0.6rem',
    borderRadius: 6,
    fontSize: '0.75rem',
    fontWeight: 600,
    background: colors[state] || '#6b7280',
    color: '#fff',
  };
};

export default function TriageDashboardPage() {
  const { projectId } = useProject();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const { data: tasks, isLoading, error } = useQuery({
    queryKey: ['triage-summary', projectId],
    queryFn: () => schedulerApi.getTriageSummary(projectId!),
    enabled: !!projectId,
    refetchInterval: 10000,
  });

  const approveMut = useMutation({
    mutationFn: (taskId: string) => schedulerApi.triageApprove(projectId!, taskId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['triage-summary'] }),
  });

  const retryMut = useMutation({
    mutationFn: (taskId: string) => schedulerApi.triageRetry(projectId!, taskId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['triage-summary'] }),
  });

  const wontfixMut = useMutation({
    mutationFn: (taskId: string) => schedulerApi.triageWonFix(projectId!, taskId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['triage-summary'] }),
  });

  if (!projectId) {
    return <div style={{ padding: 40, color: '#888' }}>选择一个项目以查看 Triage 面板</div>;
  }

  if (isLoading) {
    return <div style={{ padding: 40, color: '#888' }}>加载中...</div>;
  }

  if (error) {
    return <div style={{ padding: 40, color: '#ef4444' }}>加载失败: {(error as Error).message}</div>;
  }

  const triageTasks = tasks || [];

  return (
    <div style={{ maxWidth: 1200, margin: '0 auto', padding: '2rem' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: '1.5rem' }}>
        <AlertTriangle size={28} color="#f59e0b" />
        <h1 style={{ margin: 0, fontSize: '1.5rem', fontWeight: 700 }}>Triage Dashboard</h1>
        <span style={{ ...stateBadge('blocked'), marginLeft: 'auto' }}>
          {triageTasks.length} 待处理
        </span>
      </div>

      {triageTasks.length === 0 ? (
        <div style={{ ...cardStyle, textAlign: 'center', padding: '3rem', color: '#888' }}>
          <CheckCircle size={48} color="#22c55e" style={{ marginBottom: 12 }} />
          <p style={{ fontSize: '1.1rem', margin: 0 }}>没有需要人工介入的任务</p>
          <p style={{ fontSize: '0.85rem', marginTop: 8 }}>所有任务都在自动处理中</p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {triageTasks.map((task) => (
            <TriageCard
              key={task.id}
              task={task}
              expanded={expandedId === task.id}
              onToggle={() => setExpandedId(expandedId === task.id ? null : task.id)}
              onApprove={() => approveMut.mutate(task.id)}
              onRetry={() => retryMut.mutate(task.id)}
              onWonFix={() => wontfixMut.mutate(task.id)}
              onViewDetail={() => navigate(`/tasks/${task.id}`)}
              actionLoading={approveMut.isPending || retryMut.isPending || wontfixMut.isPending}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface TriageCardProps {
  task: TriageTask;
  expanded: boolean;
  onToggle: () => void;
  onApprove: () => void;
  onRetry: () => void;
  onWonFix: () => void;
  onViewDetail: () => void;
  actionLoading: boolean;
}

function TriageCard({ task, expanded, onToggle, onApprove, onRetry, onWonFix, onViewDetail, actionLoading }: TriageCardProps) {
  return (
    <div style={cardStyle}>
      {/* Header */}
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}
        onClick={onToggle}
      >
        {expanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
        <span style={stateBadge(task.status)}>{task.status}</span>
        <span style={{ fontWeight: 600, flex: 1 }}>{task.title}</span>
        <span style={{ fontSize: '0.8rem', color: '#888', fontFamily: 'monospace' }}>{task.id}</span>
        {task.rework_count != null && task.rework_count > 0 && (
          <span style={{ fontSize: '0.75rem', color: '#f59e0b' }}>
            rework #{task.rework_count}
          </span>
        )}
      </div>

      {/* Expanded details */}
      {expanded && (
        <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid rgba(255,255,255,0.08)' }}>
          {/* Defect info */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
            <div>
              <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 4 }}>Review Decision</div>
              <div style={{ color: task.review_decision === 'rejected' ? '#ef4444' : '#22c55e', fontWeight: 600 }}>
                {task.review_decision || 'N/A'}
              </div>
            </div>
            <div>
              <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 4 }}>Coordination Stage</div>
              <div>{task.coordination_stage || 'N/A'}</div>
            </div>
            <div>
              <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 4 }}>Escalation Reason</div>
              <div style={{ fontSize: '0.85rem' }}>{task.escalation_reason || task.block_reason || 'N/A'}</div>
            </div>
            <div>
              <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 4 }}>Last Rejection</div>
              <div style={{ fontSize: '0.85rem' }}>{task.last_rejection_reason || 'N/A'}</div>
            </div>
          </div>

          {/* Result summary */}
          {task.result_summary && (
            <div style={{ marginBottom: 16 }}>
              <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 4 }}>
                <FileText size={14} style={{ verticalAlign: 'middle', marginRight: 4 }} />
                Review Summary
              </div>
              <div style={{
                ...cardStyle,
                background: '#05070a',
                fontSize: '0.85rem',
                lineHeight: 1.6,
                maxHeight: 200,
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
              }}>
                {task.result_summary}
              </div>
            </div>
          )}

          {/* Lineage info */}
          <div style={{ marginBottom: 16, display: 'flex', gap: 16, fontSize: '0.85rem' }}>
            {task.parent_task_id && (
              <span>
                <GitBranch size={14} style={{ verticalAlign: 'middle', marginRight: 4 }} />
                Parent: <code style={{ color: '#60a5fa' }}>{task.parent_task_id}</code>
              </span>
            )}
            {task.root_task_id && task.root_task_id !== task.id && (
              <span>
                Root: <code style={{ color: '#60a5fa' }}>{task.root_task_id}</code>
              </span>
            )}
            {task.failure_code && (
              <span>
                Failure: <code style={{ color: '#f59e0b' }}>{task.failure_code}</code>
              </span>
            )}
          </div>

          {/* Actions */}
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <button style={btnApprove} onClick={onApprove} disabled={actionLoading}>
              <CheckCircle size={16} /> Approve Manually
            </button>
            <button style={btnRetry} onClick={onRetry} disabled={actionLoading}>
              <RefreshCw size={16} /> Retry
            </button>
            <button style={btnWonFix} onClick={onWonFix} disabled={actionLoading}>
              <XCircle size={16} /> Won't Fix
            </button>
            <button
              style={{ ...btnBase, background: 'rgba(255,255,255,0.08)', color: '#ccc' }}
              onClick={onViewDetail}
            >
              View Detail →
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
