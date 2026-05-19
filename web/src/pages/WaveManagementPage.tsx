import React, { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { AlertTriangle, KanbanSquare, Layers3, RefreshCw } from 'lucide-react';
import { schedulerApi } from '../api/schedulerApi';
import { useProject } from '../hooks/useProject';
import type { SchedulerWave } from '../types/scheduler';

const cardStyle: React.CSSProperties = {
  display: 'grid',
  gap: '0.75rem',
  padding: '1rem',
  background: 'rgba(255,255,255,0.03)',
  border: '1px solid rgba(255,255,255,0.06)',
  borderRadius: 12,
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

const formatTime = (value?: string) => (value ? new Date(value).toLocaleString() : '—');

const statusColor = (status: string) => (status === 'sealed' ? 'var(--accent-green)' : 'var(--accent-yellow)');

const WaveManagementPage: React.FC = () => {
  const { projectId } = useProject();
  const queryClient = useQueryClient();
  const [selectedWave, setSelectedWave] = useState<SchedulerWave | null>(null);

  const wavesQuery = useQuery({
    queryKey: ['scheduler-waves', projectId],
    queryFn: () => schedulerApi.listWaves(projectId),
    enabled: !!projectId,
    refetchInterval: 10_000,
  });

  const detailQuery = useQuery({
    queryKey: ['scheduler-wave', projectId, selectedWave?.dispatch_ref, selectedWave?.wave],
    queryFn: () => schedulerApi.getWave(projectId, selectedWave!.dispatch_ref, selectedWave!.wave),
    enabled: !!projectId && !!selectedWave,
  });

  const sealMutation = useMutation({
    mutationFn: (wave: SchedulerWave) => schedulerApi.sealWave(projectId, wave.dispatch_ref, wave.wave),
    onSuccess: async (sealedWave) => {
      setSelectedWave(sealedWave);
      await queryClient.invalidateQueries({ queryKey: ['scheduler-waves', projectId] });
      await queryClient.invalidateQueries({ queryKey: ['scheduler-wave', projectId, sealedWave.dispatch_ref, sealedWave.wave] });
    },
  });

  const waves = wavesQuery.data ?? [];
  const activeWave = detailQuery.data ?? selectedWave;

  return (
    <div style={{ display: 'grid', gap: '1.5rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', gap: '1rem' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--accent-cyan)', marginBottom: '0.5rem' }}>
            <Layers3 size={16} />
            <span style={{ fontSize: '0.75rem', fontWeight: 700, letterSpacing: '2px' }}>WAVE CONTROL SURFACE</span>
          </div>
          <h2 className="neon-text" style={{ fontSize: '2.2rem' }}>Wave 管理</h2>
          <p style={{ color: 'var(--text-secondary)' }}>查看项目 Wave 状态、任务计数，并对已完成批次执行 seal 操作。</p>
        </div>
        <button
          type="button"
          onClick={() => wavesQuery.refetch()}
          disabled={wavesQuery.isFetching}
          style={{ ...buttonStyle, color: 'var(--accent-cyan)' }}
        >
          <RefreshCw size={14} style={{ verticalAlign: 'middle', marginRight: 6 }} />
          {wavesQuery.isFetching ? '刷新中' : '刷新'}
        </button>
      </div>

      <div className="glass-card" style={{ display: 'grid', gap: '1rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          <KanbanSquare size={18} color="var(--accent-yellow)" />
          <strong>Wave 列表</strong>
          <span style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>{projectId || '未选择项目'}</span>
        </div>

        {wavesQuery.isLoading && <p style={{ color: 'var(--text-secondary)' }}>加载 Wave 数据中…</p>}
        {wavesQuery.error && <ErrorLine message="Wave 数据加载失败" />}
        {!wavesQuery.isLoading && !waves.length && (
          <p style={{ color: 'var(--text-secondary)' }}>当前项目还没有 Wave 记录。</p>
        )}

        <div style={{ display: 'grid', gap: '0.75rem' }}>
          {waves.map((wave) => (
            <button
              key={`${wave.dispatch_ref}:${wave.wave}`}
              type="button"
              onClick={() => setSelectedWave(wave)}
              style={{
                ...cardStyle,
                textAlign: 'left',
                cursor: 'pointer',
                borderColor: activeWave?.dispatch_ref === wave.dispatch_ref && activeWave?.wave === wave.wave ? 'rgba(0, 242, 234, 0.55)' : 'rgba(255,255,255,0.06)',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: '1rem', alignItems: 'center' }}>
                <div>
                  <div className="mono" style={{ color: 'var(--text-primary)', fontWeight: 700 }}>{wave.dispatch_ref}</div>
                  <div style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>Wave {wave.wave} · 创建 {formatTime(wave.created_at)}</div>
                </div>
                <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
                  <span style={{ color: statusColor(wave.status), fontWeight: 700 }}>{wave.status}</span>
                  <span style={{ color: 'var(--text-secondary)' }}>{wave.task_count} tasks</span>
                </div>
              </div>
            </button>
          ))}
        </div>
      </div>

      {activeWave && (
        <div className="glass-card" style={{ display: 'grid', gap: '1rem' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: '1rem', alignItems: 'flex-start' }}>
            <div>
              <h3>Wave 详情</h3>
              <p className="mono" style={{ color: 'var(--text-secondary)', marginTop: '0.35rem' }}>{activeWave.dispatch_ref} / wave {activeWave.wave}</p>
            </div>
            <button
              type="button"
              onClick={() => sealMutation.mutate(activeWave)}
              disabled={activeWave.status === 'sealed' || sealMutation.isPending}
              style={{
                ...buttonStyle,
                color: activeWave.status === 'sealed' ? 'var(--text-secondary)' : '#071014',
                background: activeWave.status === 'sealed' ? 'rgba(255,255,255,0.04)' : 'var(--accent-cyan)',
              }}
            >
              {activeWave.status === 'sealed' ? '已 sealed' : sealMutation.isPending ? 'Sealing…' : 'Seal Wave'}
            </button>
          </div>

          {detailQuery.error && <ErrorLine message="Wave 详情加载失败" />}
          {sealMutation.error && <ErrorLine message="Seal 操作失败" />}

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '0.75rem' }}>
            <Metric label="状态" value={activeWave.status} color={statusColor(activeWave.status)} />
            <Metric label="任务数" value={String(activeWave.task_count)} />
            <Metric label="创建时间" value={formatTime(activeWave.created_at)} />
            <Metric label="Seal 时间" value={formatTime(activeWave.sealed_at)} />
          </div>

          <div style={cardStyle}>
            <strong>状态分布</strong>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
              {Object.entries(activeWave.counts_by_status).map(([status, count]) => (
                <span key={status} style={{ border: '1px solid var(--border-color)', borderRadius: 999, padding: '0.35rem 0.65rem', color: 'var(--text-secondary)' }}>
                  <span style={{ color: 'var(--text-primary)', fontWeight: 700 }}>{count}</span> {status}
                </span>
              ))}
            </div>
          </div>

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.75rem' }}>
            <Link to="/board" style={{ ...buttonStyle, textDecoration: 'none' }}>查看调度看板</Link>
            <Link to={`/events?dispatch_ref=${encodeURIComponent(activeWave.dispatch_ref)}`} style={{ ...buttonStyle, textDecoration: 'none' }}>查看事件链</Link>
          </div>
        </div>
      )}
    </div>
  );
};

function Metric({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div style={cardStyle}>
      <span style={{ color: 'var(--text-secondary)', fontSize: '0.78rem' }}>{label}</span>
      <strong style={{ color }}>{value}</strong>
    </div>
  );
}

function ErrorLine({ message }: { message: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--accent-magenta)' }}>
      <AlertTriangle size={16} />
      <span>{message}</span>
    </div>
  );
}

export default WaveManagementPage;
