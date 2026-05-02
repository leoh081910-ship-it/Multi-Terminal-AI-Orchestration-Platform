import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { orgApi } from '../api/orgApi';

export default function OrganizationPage() {
  const queryClient = useQueryClient();
  const [selectedOrg, setSelectedOrg] = useState('');
  const [selectedDept, setSelectedDept] = useState('');
  const [error, setError] = useState('');

  // Create forms
  const [orgName, setOrgName] = useState('');
  const [orgDesc, setOrgDesc] = useState('');
  const [deptName, setDeptName] = useState('');
  const [deptDesc, setDeptDesc] = useState('');
  const [teamName, setTeamName] = useState('');
  const [teamDesc, setTeamDesc] = useState('');
  const [agentName, setAgentName] = useState('');
  const [agentType, setAgentType] = useState('claude');

  const { data: orgs = [] } = useQuery({
    queryKey: ['orgs'],
    queryFn: orgApi.listOrgs,
  });

  const { data: depts = [] } = useQuery({
    queryKey: ['depts', selectedOrg],
    queryFn: () => orgApi.listDepts(selectedOrg),
    enabled: !!selectedOrg,
  });

  const { data: teams = [] } = useQuery({
    queryKey: ['teams', selectedOrg, selectedDept],
    queryFn: () => orgApi.listTeams(selectedOrg, selectedDept),
    enabled: !!selectedOrg && !!selectedDept,
  });

  const { data: agents = [] } = useQuery({
    queryKey: ['agents', selectedOrg],
    queryFn: () => orgApi.listAgents(selectedOrg),
    enabled: !!selectedOrg,
    refetchInterval: 15000,
  });

  const createOrg = useMutation({
    mutationFn: () => orgApi.createOrg({ name: orgName, description: orgDesc || undefined }),
    onSuccess: () => { setOrgName(''); setOrgDesc(''); setError(''); queryClient.invalidateQueries({ queryKey: ['orgs'] }); },
    onError: (e: Error) => setError(e.message),
  });

  const createDept = useMutation({
    mutationFn: () => orgApi.createDept(selectedOrg, { name: deptName, description: deptDesc || undefined }),
    onSuccess: () => { setDeptName(''); setDeptDesc(''); setError(''); queryClient.invalidateQueries({ queryKey: ['depts', selectedOrg] }); },
    onError: (e: Error) => setError(e.message),
  });

  const createTeam = useMutation({
    mutationFn: () => orgApi.createTeam(selectedOrg, selectedDept, { name: teamName, description: teamDesc || undefined }),
    onSuccess: () => { setTeamName(''); setTeamDesc(''); setError(''); queryClient.invalidateQueries({ queryKey: ['teams', selectedOrg, selectedDept] }); },
    onError: (e: Error) => setError(e.message),
  });

  const createAgent = useMutation({
    mutationFn: () => orgApi.createAgent(selectedOrg, { name: agentName, type: agentType }),
    onSuccess: () => { setAgentName(''); setError(''); queryClient.invalidateQueries({ queryKey: ['agents', selectedOrg] }); },
    onError: (e: Error) => setError(e.message),
  });

  const deleteOrg = useMutation({
    mutationFn: (id: string) => orgApi.deleteOrg(id),
    onSuccess: () => { setSelectedOrg(''); queryClient.invalidateQueries({ queryKey: ['orgs'] }); },
    onError: (e: Error) => setError(e.message),
  });

  const inputS: React.CSSProperties = { background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13 };
  const btn = (bg: string): React.CSSProperties => ({ background: bg, color: '#0a0b10', border: 'none', borderRadius: 6, padding: '8px 16px', fontWeight: 600, cursor: 'pointer', fontSize: 12 });

  return (
    <div style={{ padding: 24 }}>
      <h2 style={{ color: '#00f2ea', fontSize: 20, fontWeight: 700, marginBottom: 16 }}>Organization Management</h2>
      {error && <div style={{ color: '#ff0050', fontSize: 12, marginBottom: 12 }}>{error}</div>}

      {/* Create Org */}
      <div className="glass-card" style={{ padding: 20, marginBottom: 20 }}>
        <h3 style={{ color: '#ccc', fontSize: 14, marginBottom: 12 }}>Create Organization</h3>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          <input placeholder="Name" value={orgName} onChange={(e) => setOrgName(e.target.value)} style={{ ...inputS, flex: 1, minWidth: 160 }} />
          <input placeholder="Description" value={orgDesc} onChange={(e) => setOrgDesc(e.target.value)} style={{ ...inputS, flex: 2, minWidth: 200 }} />
          <button onClick={() => createOrg.mutate()} disabled={!orgName} style={btn('#00f2ea')}>Create</button>
        </div>
      </div>

      {/* Org selector */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, alignItems: 'center' }}>
        <select value={selectedOrg} onChange={(e) => { setSelectedOrg(e.target.value); setSelectedDept(''); }} style={{ ...inputS, minWidth: 200 }}>
          <option value="">Select Organization</option>
          {orgs.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
        </select>
        {selectedOrg && (
          <button onClick={() => deleteOrg.mutate(selectedOrg)} style={{ ...btn('#ff0050'), fontSize: 11, padding: '6px 12px' }}>Delete Org</button>
        )}
      </div>

      {selectedOrg && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: 16 }}>
          {/* Departments */}
          <div className="glass-card" style={{ padding: 20 }}>
            <h3 style={{ color: '#a78bfa', fontSize: 14, marginBottom: 12 }}>Departments</h3>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
              <input placeholder="Name" value={deptName} onChange={(e) => setDeptName(e.target.value)} style={{ ...inputS, flex: 1 }} />
              <button onClick={() => createDept.mutate()} disabled={!deptName} style={btn('#a78bfa')}>+</button>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {depts.map((d) => (
                <div
                  key={d.id}
                  onClick={() => setSelectedDept(d.id === selectedDept ? '' : d.id)}
                  style={{
                    padding: '8px 12px', borderRadius: 6, cursor: 'pointer', fontSize: 13,
                    background: d.id === selectedDept ? 'rgba(167,139,250,0.15)' : 'rgba(255,255,255,0.04)',
                    border: `1px solid ${d.id === selectedDept ? 'rgba(167,139,250,0.4)' : 'transparent'}`,
                    color: '#ccc',
                  }}
                >
                  {d.name}
                  {d.description && <span style={{ fontSize: 11, color: '#666', marginLeft: 8 }}>{d.description}</span>}
                </div>
              ))}
              {depts.length === 0 && <div style={{ color: '#555', fontSize: 12 }}>No departments</div>}
            </div>
          </div>

          {/* Teams */}
          <div className="glass-card" style={{ padding: 20 }}>
            <h3 style={{ color: '#00f2ea', fontSize: 14, marginBottom: 12 }}>Teams {selectedDept && `in ${depts.find(d => d.id === selectedDept)?.name || ''}`}</h3>
            {selectedDept ? (
              <>
                <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                  <input placeholder="Name" value={teamName} onChange={(e) => setTeamName(e.target.value)} style={{ ...inputS, flex: 1 }} />
                  <button onClick={() => createTeam.mutate()} disabled={!teamName} style={btn('#00f2ea')}>+</button>
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {teams.map((t) => (
                    <div key={t.id} style={{ padding: '8px 12px', borderRadius: 6, fontSize: 13, background: 'rgba(255,255,255,0.04)', color: '#ccc' }}>
                      {t.name}
                      {t.description && <span style={{ fontSize: 11, color: '#666', marginLeft: 8 }}>{t.description}</span>}
                    </div>
                  ))}
                  {teams.length === 0 && <div style={{ color: '#555', fontSize: 12 }}>No teams</div>}
                </div>
              </>
            ) : (
              <div style={{ color: '#555', fontSize: 12 }}>Select a department first</div>
            )}
          </div>

          {/* Agents */}
          <div className="glass-card" style={{ padding: 20 }}>
            <h3 style={{ color: '#00ffaa', fontSize: 14, marginBottom: 12 }}>Agents</h3>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
              <input placeholder="Name" value={agentName} onChange={(e) => setAgentName(e.target.value)} style={{ ...inputS, flex: 1 }} />
              <select value={agentType} onChange={(e) => setAgentType(e.target.value)} style={{ ...inputS, width: 110 }}>
                <option value="claude">Claude</option>
                <option value="gemini">Gemini</option>
                <option value="codex">Codex</option>
                <option value="custom">Custom</option>
              </select>
              <button onClick={() => createAgent.mutate()} disabled={!agentName} style={btn('#00ffaa')}>+</button>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              {agents.map((a) => (
                <div key={a.id} style={{ padding: '8px 12px', borderRadius: 6, fontSize: 13, background: 'rgba(255,255,255,0.04)', color: '#ccc', display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ width: 8, height: 8, borderRadius: '50%', background: a.status === 'idle' ? '#4a4a6a' : a.status === 'busy' ? '#00f2ea' : '#ff0050' }} />
                  <span style={{ flex: 1 }}>{a.name}</span>
                  <span style={{ fontSize: 10, color: '#888' }}>{a.type}</span>
                </div>
              ))}
              {agents.length === 0 && <div style={{ color: '#555', fontSize: 12 }}>No agents</div>}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
