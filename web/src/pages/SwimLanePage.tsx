import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { orgApi } from '../api/orgApi';
import { schedulerApi } from '../api/schedulerApi';
import { useProject } from '../hooks/useProject';
import { useWebSocket } from '../hooks/useWebSocket';
import SwimLaneBoard from '../components/SwimLaneBoard';

export default function SwimLanePage() {
  const { projectId } = useProject();
  const [orgId, setOrgId] = useState('');
  useWebSocket(projectId);

  const { data: orgs = [] } = useQuery({
    queryKey: ['orgs'],
    queryFn: orgApi.listOrgs,
  });

  const { data: tasks = [] } = useQuery({
    queryKey: ['tasks', projectId],
    queryFn: () => schedulerApi.getTasks(projectId),
    refetchInterval: 10000,
  });

  const { data: agents = [] } = useQuery({
    queryKey: ['agents', orgId],
    queryFn: () => orgApi.listAgents(orgId),
    enabled: !!orgId,
    refetchInterval: 15000,
  });

  const agentNames = agents.map((a) => a.name);

  return (
    <div style={{ padding: 24 }}>
      <h2 style={{ color: '#00f2ea', fontSize: 20, fontWeight: 700, marginBottom: 16 }}>
        Swim Lane Board
      </h2>

      <select
        value={orgId}
        onChange={(e) => setOrgId(e.target.value)}
        style={{ background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13, marginBottom: 20 }}
      >
        <option value="">Select Organization</option>
        {orgs.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
      </select>

      <SwimLaneBoard tasks={tasks} agents={agentNames} />
    </div>
  );
}
