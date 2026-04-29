import React from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  AlertCircle,
  CheckCircle,
  Clock3,
  Cpu,
  Layers,
  Radar,
  Terminal,
  Wifi,
} from 'lucide-react';
import { schedulerApi } from '../api/schedulerApi';
import { useProject } from '../hooks/useProject';
import { Agent, RuntimeStatus } from '../types/scheduler';

const cardStyle: React.CSSProperties = {
  position: 'relative',
  overflow: 'hidden',
};

const listCardStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'flex-start',
  gap: '1rem',
  padding: '0.85rem',
  background: 'rgba(255,255,255,0.02)',
  borderRadius: '10px',
};

const StatCard = ({
  title,
  value,
  subValue,
  icon,
  color,
}: {
  title: string;
  value: React.ReactNode;
  subValue: string;
  icon: React.ReactElement<{ size?: number }>;
  color: string;
}) => (
  <div className="glass-card" style={cardStyle}>
    <div style={{ position: 'absolute', top: '-10px', right: '-10px', opacity: 0.08, color: `var(--accent-${color})` }}>
      {React.cloneElement(icon, { size: 84 })}
    </div>
    <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1rem', color: `var(--accent-${color})` }}>
      {icon}
      <span style={{ fontSize: '0.75rem', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '1px' }}>{title}</span>
    </div>
    <h3 className="mono" style={{ fontSize: '2rem', fontWeight: 800, marginBottom: '0.25rem' }}>{value}</h3>
    <p style={{ color: 'var(--text-secondary)', fontSize: '0.8rem' }}>{subValue}</p>
  </div>
);

const AgentPill = ({ agent, count }: { agent: Agent; count: number }) => (
  <div
    style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      padding: '0.85rem 1rem',
      background: 'rgba(255,255,255,0.03)',
      borderRadius: '10px',
      border: '1px solid rgba(255,255,255,0.05)',
    }}
  >
    <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
      <div
        style={{
          width: 34,
          height: 34,
          borderRadius: '50%',
          background:
            agent === Agent.CLAUDE
              ? 'rgba(0, 242, 234, 0.1)'
              : agent === Agent.GEMINI
                ? 'rgba(255, 0, 80, 0.1)'
                : 'rgba(255, 238, 0, 0.1)',
          color:
            agent === Agent.CLAUDE
              ? 'var(--accent-cyan)'
              : agent === Agent.GEMINI
                ? 'var(--accent-magenta)'
                : 'var(--accent-yellow)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Cpu size={16} />
      </div>
      <span style={{ fontWeight: 600 }}>{agent}</span>
    </div>
    <span className="mono" style={{ fontSize: '1.1rem', fontWeight: 700 }}>{count}</span>
  </div>
);

const OrchestratorHome: React.FC = () => {
  const { projectId, project } = useProject();
  const { data: stats, isLoading } = useQuery({
    queryKey: ['orchestrator-stats', projectId],
    queryFn: () => schedulerApi.getStats(projectId),
    enabled: Boolean(projectId),
    refetchInterval: 5000,
  });

  const activity = stats?.recent_updates ?? [];
  const completions = stats?.recent_done_tasks ?? [];
  const runtimes = stats?.runtime_health ?? [];

  return (
    <div className="orchestrator-home">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginBottom: '2.5rem', gap: '1rem' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--accent-cyan)', marginBottom: '0.5rem' }}>
            <Terminal size={16} />
            <span style={{ fontSize: '0.75rem', fontWeight: 700, letterSpacing: '2px' }}>ORCHESTRATION OVERVIEW</span>
          </div>
          <h2 className="neon-text" style={{ fontSize: '2.5rem' }}>控制中心</h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '1rem' }}>
            {project ? `当前项目：${project.name}` : '加载当前项目中…'}
          </p>
        </div>
        <div className="glass-card" style={{ padding: '0.85rem 1.5rem', borderLeft: '4px solid var(--accent-yellow)', minWidth: '320px' }}>
          <p style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginBottom: '0.35rem' }}>当前重点</p>
          <p style={{ fontWeight: 700, fontSize: '0.95rem', lineHeight: 1.4 }}>{stats?.current_focus || '暂无焦点任务'}</p>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '1.25rem', marginBottom: '2rem' }}>
        <StatCard title="总任务数" value={stats?.total_tasks || 0} subValue="当前项目正在跟踪的任务总数" icon={<Layers size={24} />} color="cyan" />
        <StatCard title="活跃会话" value={stats?.active_sessions || 0} subValue="当前正在执行的会话数量" icon={<Terminal size={24} />} color="green" />
        <StatCard title="待调度" value={stats?.queued_tasks || 0} subValue="排队中、已派发和运行中的任务" icon={<Radar size={24} />} color="yellow" />
        <StatCard title="派发失败" value={stats?.failed_dispatches || 0} subValue="最近派发失败的任务数" icon={<AlertCircle size={24} />} color="magenta" />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '0.9fr 1.15fr 1.15fr', gap: '1.5rem' }}>
        <div className="glass-card">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1.25rem' }}>
            <Cpu size={20} color="var(--accent-cyan)" />
            <h3 style={{ fontSize: '1.05rem' }}>代理负载</h3>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.9rem', marginBottom: '1.25rem' }}>
            {stats && Object.entries(stats.agent_task_counts).map(([agent, count]) => (
              <AgentPill key={agent} agent={agent as Agent} count={count as number} />
            ))}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '0.85rem' }}>
            <Wifi size={18} color="var(--accent-green)" />
            <h3 style={{ fontSize: '0.95rem' }}>运行时健康</h3>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.7rem' }}>
            {runtimes.map((runtime) => (
              <div key={runtime.owner_agent} style={{ padding: '0.8rem', borderRadius: 10, background: 'rgba(255,255,255,0.03)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.75rem', marginBottom: '0.2rem' }}>
                  <strong style={{ fontSize: '0.84rem' }}>{runtime.owner_agent}</strong>
                  <span className="mono" style={{ fontSize: '0.72rem', color: runtime.status === RuntimeStatus.AVAILABLE ? 'var(--accent-green)' : 'var(--accent-magenta)' }}>
                    {runtime.status === RuntimeStatus.AVAILABLE ? '可用' : '不可用'}
                  </span>
                </div>
                <div style={{ fontSize: '0.74rem', color: 'var(--text-secondary)' }}>{runtime.runtime} · 会话 {runtime.active_sessions}</div>
                <div style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.2rem' }}>
                  {runtime.last_error || (runtime.command_configured ? '命令已配置' : '命令未配置')}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="glass-card">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1.25rem' }}>
            <Clock3 size={20} color="var(--accent-green)" />
            <h3 style={{ fontSize: '1.05rem' }}>最近更新</h3>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            {isLoading ? (
              <p>加载中…</p>
            ) : activity.length ? (
              activity.map((task) => (
                <div key={task.id} style={listCardStyle}>
                  <div style={{ color: 'var(--accent-cyan)' }}><Clock3 size={16} /></div>
                  <div style={{ flex: 1 }}>
                    <p style={{ fontSize: '0.86rem', fontWeight: 600 }}>{task.title}</p>
                    <p style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>{task.owner_agent} · {task.status} · {task.dispatch_status}</p>
                    <p style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>{task.last_dispatch_error || task.next_action || task.description || '暂无额外说明'}</p>
                  </div>
                  <span className="mono" style={{ fontSize: '0.7rem', opacity: 0.6 }}>{new Date(task.updated_at).toLocaleTimeString()}</span>
                </div>
              ))
            ) : (
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>暂无最近更新。</p>
            )}
          </div>
        </div>

        <div className="glass-card">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1.25rem' }}>
            <CheckCircle size={20} color="var(--accent-yellow)" />
            <h3 style={{ fontSize: '1.05rem' }}>最近完成</h3>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            {isLoading ? (
              <p>加载中…</p>
            ) : completions.length ? (
              completions.map((task) => (
                <div key={task.id} style={listCardStyle}>
                  <div style={{ color: 'var(--accent-green)' }}><CheckCircle size={16} /></div>
                  <div style={{ flex: 1 }}>
                    <p style={{ fontSize: '0.86rem', fontWeight: 600 }}>{task.title}</p>
                    <p style={{ fontSize: '0.72rem', color: 'var(--text-secondary)' }}>{task.owner_agent} · {task.id}</p>
                    <p style={{ fontSize: '0.72rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>{task.result_summary || '任务已完成。'}</p>
                  </div>
                  <span className="mono" style={{ fontSize: '0.7rem', opacity: 0.6 }}>{new Date(task.updated_at).toLocaleTimeString()}</span>
                </div>
              ))
            ) : (
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>还没有已完成任务。</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default OrchestratorHome;
