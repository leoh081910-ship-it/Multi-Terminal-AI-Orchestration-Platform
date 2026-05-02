import { useQuery, useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { orgApi, type Agent, type AgentScore } from '../api/orgApi';
import { useWebSocket } from '../hooks/useWebSocket';

const STATUS_COLORS: Record<string, string> = {
  idle: '#4a4a6a',
  busy: '#00f2ea',
  offline: '#ff0050',
  error: '#ff0050',
};

export default function AgentWorkbenchPage() {
  const [orgId, setOrgId] = useState('');
  const [nowMs] = useState(() => Date.now());
  useWebSocket();

  const { data: orgs = [] } = useQuery({
    queryKey: ['orgs'],
    queryFn: orgApi.listOrgs,
  });

  const { data: agents = [] } = useQuery({
    queryKey: ['agents', orgId],
    queryFn: () => orgApi.listAgents(orgId),
    enabled: !!orgId,
    refetchInterval: 15000,
  });

  return (
    <div style={{ padding: 24 }}>
      <h2 style={{ color: '#00f2ea', fontSize: 20, fontWeight: 700, marginBottom: 16 }}>
        Agent Workbench
      </h2>

      <select
        value={orgId}
        onChange={(e) => setOrgId(e.target.value)}
        style={{ background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13, marginBottom: 20 }}
      >
        <option value="">Select Organization</option>
        {orgs.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
      </select>

      {/* Agent grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 16, marginBottom: 24 }}>
        {agents.map((agent) => (
          <AgentCard key={agent.id} agent={agent} nowMs={nowMs} />
        ))}
        {orgId && agents.length === 0 && (
          <div style={{ color: '#666', padding: 40, textAlign: 'center', gridColumn: '1 / -1' }}>
            No agents registered in this organization
          </div>
        )}
      </div>

      {/* Routing Preview */}
      {orgId && agents.length > 0 && <RoutingPreview orgId={orgId} />}
    </div>
  );
}

function AgentCard({ agent, nowMs }: { agent: Agent; nowMs: number }) {
  const statusColor = STATUS_COLORS[agent.status] || '#888';
  const isOnline = agent.last_heartbeat_at && (nowMs - new Date(agent.last_heartbeat_at).getTime() < 60000);

  return (
    <div className="glass-card" style={{ padding: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
        <div style={{
          width: 10, height: 10, borderRadius: '50%',
          background: isOnline ? '#00ffaa' : statusColor,
          boxShadow: `0 0 8px ${isOnline ? '#00ffaa' : statusColor}66`,
        }} />
        <span style={{ fontSize: 14, fontWeight: 600, color: '#ddd' }}>{agent.name}</span>
        <span style={{ fontSize: 11, color: '#888', marginLeft: 'auto' }}>{agent.type}</span>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, fontSize: 12 }}>
        <div>
          <div style={{ color: '#666', marginBottom: 2 }}>Status</div>
          <div style={{ color: statusColor, fontWeight: 600 }}>{agent.status}</div>
        </div>
        <div>
          <div style={{ color: '#666', marginBottom: 2 }}>Role</div>
          <div style={{ color: '#ccc' }}>{agent.role_id || '—'}</div>
        </div>
      </div>

      {agent.specialties.length > 0 && (
        <div style={{ marginTop: 10, display: 'flex', gap: 4, flexWrap: 'wrap' }}>
          {agent.specialties.map((s) => (
            <span key={s} style={{ fontSize: 10, padding: '2px 6px', borderRadius: 4, background: 'rgba(0,242,234,0.1)', color: '#00f2ea' }}>{s}</span>
          ))}
        </div>
      )}

      {agent.last_heartbeat_at && (
        <div style={{ marginTop: 8, fontSize: 10, color: '#555' }}>
          Last heartbeat: {new Date(agent.last_heartbeat_at).toLocaleString()}
        </div>
      )}
    </div>
  );
}

function RoutingPreview({ orgId }: { orgId: string }) {
  const [taskType, setTaskType] = useState('');
  const [scores, setScores] = useState<AgentScore[]>([]);

  const preview = useMutation({
    mutationFn: () => orgApi.previewRouting(orgId, { task_type: taskType }),
    onSuccess: (data) => setScores(data as AgentScore[]),
  });

  return (
    <div className="glass-card" style={{ padding: 20 }}>
      <h3 style={{ color: '#a78bfa', fontSize: 14, marginBottom: 12 }}>Routing Preview</h3>
      <p style={{ fontSize: 12, color: '#888', marginBottom: 12 }}>
        See how the router would score agents for a given task type.
      </p>
      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        <input
          placeholder="Task type (e.g., code, analysis, review)"
          value={taskType}
          onChange={(e) => setTaskType(e.target.value)}
          style={{ background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13, flex: 1 }}
        />
        <button
          onClick={() => preview.mutate()}
          disabled={!taskType}
          style={{ background: '#a78bfa', color: '#0a0b10', border: 'none', borderRadius: 6, padding: '8px 20px', fontWeight: 600, cursor: 'pointer', fontSize: 13 }}
        >
          Score Agents
        </button>
      </div>

      {scores.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 80px 80px 80px 80px', gap: 8, fontSize: 11, color: '#888', fontWeight: 600, padding: '0 8px' }}>
            <span>AGENT</span>
            <span style={{ textAlign: 'right' }}>TOTAL</span>
            <span style={{ textAlign: 'right' }}>CAP</span>
            <span style={{ textAlign: 'right' }}>LOAD</span>
            <span style={{ textAlign: 'right' }}>AFFINITY</span>
          </div>
          {scores.map((s, i) => (
            <div key={s.AgentID} style={{
              display: 'grid', gridTemplateColumns: '1fr 80px 80px 80px 80px', gap: 8, fontSize: 12, padding: '8px',
              borderRadius: 6, background: i === 0 ? 'rgba(0,255,170,0.08)' : 'rgba(255,255,255,0.03)',
              border: i === 0 ? '1px solid rgba(0,255,170,0.2)' : '1px solid transparent',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ color: i === 0 ? '#00ffaa' : '#ccc', fontWeight: i === 0 ? 700 : 400 }}>{s.AgentName}</span>
                {!s.Available && <span style={{ fontSize: 9, color: '#ff0050', padding: '1px 4px', borderRadius: 3, background: 'rgba(255,0,80,0.15)' }}>offline</span>}
              </div>
              <span style={{ textAlign: 'right', color: '#00f2ea', fontWeight: 700 }}>{(s.Score * 100).toFixed(0)}%</span>
              <span style={{ textAlign: 'right', color: '#a78bfa' }}>{(s.Capability * 100).toFixed(0)}%</span>
              <span style={{ textAlign: 'right', color: '#ff9f43' }}>{(s.Load * 100).toFixed(0)}%</span>
              <span style={{ textAlign: 'right', color: '#ffee00' }}>{(s.Affinity * 100).toFixed(0)}%</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
