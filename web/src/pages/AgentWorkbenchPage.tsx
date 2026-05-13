import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { orgApi, type Agent, type AgentScore, type CapabilityManifest } from '../api/orgApi';
import { useWebSocket } from '../hooks/useWebSocket';

const STATUS_COLORS: Record<string, string> = {
  idle: '#4a4a6a',
  busy: '#00f2ea',
  offline: '#ff0050',
  error: '#ff0050',
};

export default function AgentWorkbenchPage() {
  const [orgId, setOrgId] = useState('');
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
          <AgentCard key={agent.id} agent={agent} orgId={orgId} />
        ))}
        {orgId && agents.length === 0 && (
          <div style={{ color: '#666', padding: 40, textAlign: 'center', gridColumn: '1 / -1' }}>
            No agents registered in this organization
          </div>
        )}
      </div>

      {/* Routing Preview */}
      {orgId && agents.length > 0 && <RoutingPreview orgId={orgId} />}

      {/* HTTP/MCP Agent Registration */}
      {orgId && <HTTPAgentForm orgId={orgId} />}
    </div>
  );
}

const RUNNER_TYPE_META: Record<string, { color: string; label: string }> = {
  http: { color: '#ff9f43', label: 'HTTP' },
  mcp: { color: '#a78bfa', label: 'MCP' },
  cli: { color: '#00f2ea', label: 'CLI' },
};

function AgentCard({ agent, orgId }: { agent: Agent; orgId: string }) {
  const queryClient = useQueryClient();
  const statusColor = STATUS_COLORS[agent.status] || '#888';
  const agentsState = queryClient.getQueryState<Agent[]>(['agents', orgId]);
  const checkedAt = agentsState?.dataUpdatedAt ?? 0;
  const isOnline = Boolean(agent.last_heartbeat_at && checkedAt - new Date(agent.last_heartbeat_at).getTime() < 60000);
  const meta = RUNNER_TYPE_META[agent.runner_type ?? 'cli'] ?? RUNNER_TYPE_META.cli;

  const [showCaps, setShowCaps] = useState(false);
  const [capManifest, setCapManifest] = useState<CapabilityManifest | null>(null);

  const fetchCaps = useMutation({
    mutationFn: () => orgApi.getAgentCapabilities(orgId, agent.id),
    onSuccess: (data) => {
      setCapManifest(data as CapabilityManifest);
      setShowCaps(true);
    },
  });

  const handleViewCaps = () => {
    if (showCaps) { setShowCaps(false); return; }
    fetchCaps.mutate();
  };

  const formatContextWindow = (ctx: number) => {
    if (ctx >= 1000000) return `${(ctx / 1000000).toFixed(0)}M`;
    if (ctx >= 1000) return `${(ctx / 1000).toFixed(0)}K`;
    return ctx.toString();
  };

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

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 6, fontSize: 11, marginBottom: 10 }}>
        <div>
          <div style={{ color: '#555', marginBottom: 1 }}>Status</div>
          <div style={{ color: statusColor, fontWeight: 600 }}>{agent.status}</div>
        </div>
        <div>
          <div style={{ color: '#555', marginBottom: 1 }}>Runner</div>
          <div style={{ color: meta.color, fontWeight: 600 }}>{meta.label}</div>
        </div>
        <div>
          <div style={{ color: '#555', marginBottom: 1 }}>Model</div>
          <div style={{ color: '#ccc', fontWeight: 500, fontSize: 10, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {capManifest?.model_family || agent.config?.model || '—'}
          </div>
        </div>
      </div>

      {/* Task types from agent specialties or manifest */}
      {((agent.specialties?.length ?? 0) > 0 || (capManifest?.task_types?.length ?? 0) > 0) && (
        <div style={{ display: 'flex', gap: 3, flexWrap: 'wrap', marginBottom: 8 }}>
          {(capManifest?.task_types?.length ? capManifest.task_types : agent.specialties ?? []).slice(0, 6).map((t) => (
            <span key={t} style={{ fontSize: 9, padding: '2px 5px', borderRadius: 3, background: 'rgba(167,139,250,0.15)', color: '#a78bfa', fontWeight: 500 }}>{t}</span>
          ))}
        </div>
      )}

      {/* Feature badges from manifest */}
      {capManifest && (
        <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 8 }}>
          {capManifest.supports_thinking && <span style={{ fontSize: 9, padding: '1px 5px', borderRadius: 3, background: 'rgba(0,255,170,0.12)', color: '#00ffaa' }}>Thinking</span>}
          {capManifest.supports_multimodal && <span style={{ fontSize: 9, padding: '1px 5px', borderRadius: 3, background: 'rgba(255,159,67,0.12)', color: '#ff9f43' }}>Multimodal</span>}
          {capManifest.supports_streaming && <span style={{ fontSize: 9, padding: '1px 5px', borderRadius: 3, background: 'rgba(0,242,234,0.12)', color: '#00f2ea' }}>Streaming</span>}
          {capManifest.context_window > 0 && (
            <span style={{ fontSize: 9, padding: '1px 5px', borderRadius: 3, background: 'rgba(255,238,0,0.1)', color: '#ffee00' }}>Ctx {formatContextWindow(capManifest.context_window)}</span>
          )}
        </div>
      )}

      {agent.last_heartbeat_at && (
        <div style={{ fontSize: 9, color: '#444', marginBottom: 6 }}>
          heartbeat: {new Date(agent.last_heartbeat_at).toLocaleTimeString()}
        </div>
      )}

      <button
        onClick={handleViewCaps}
        style={{
          fontSize: 10, padding: '3px 10px', borderRadius: 4,
          background: showCaps ? 'rgba(167,139,250,0.2)' : 'rgba(255,255,255,0.06)',
          border: '1px solid rgba(255,255,255,0.1)',
          color: '#a78bfa', cursor: 'pointer', marginTop: 4,
        }}
      >
        {fetchCaps.isPending ? 'Loading...' : showCaps ? 'Hide Capabilities' : 'View Capabilities'}
      </button>
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

// --- HTTP/MCP Agent Registration Form ---

function HTTPAgentForm({ orgId }: { orgId: string }) {
  const [runnerType, setRunnerType] = useState<'http' | 'mcp'>('http');
  const [agentName, setAgentName] = useState('');
  const [endpoint, setEndpoint] = useState('');
  const [authToken, setAuthToken] = useState('');
  const [model, setModel] = useState('');
  const [bodyTemplate, setBodyTemplate] = useState('');
  const [outputPath, setOutputPath] = useState('');
  const [timeoutMs, setTimeoutMs] = useState('300000');
  const [mcpToolName, setMcpToolName] = useState('execute_task');
  const [showForm, setShowForm] = useState(false);
  const [formError, setFormError] = useState('');
  const [successMsg, setSuccessMsg] = useState('');
  const queryClient = useQueryClient();

  const createAgent = useMutation({
    mutationFn: (input: Parameters<typeof orgApi.createAgent>[1]) => orgApi.createAgent(orgId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agents', orgId] });
      setSuccessMsg(`${runnerType.toUpperCase()} Agent registered successfully`);
      setFormError('');
      setAgentName('');
      setEndpoint('');
      setAuthToken('');
      setModel('');
      setBodyTemplate('');
      setOutputPath('');
      setTimeoutMs('300000');
      setMcpToolName('execute_task');
      setShowForm(false);
      setTimeout(() => setSuccessMsg(''), 4000);
    },
    onError: (err: Error) => {
      setFormError(err.message);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');
    if (!agentName.trim()) { setFormError('Agent name is required'); return; }
    if (!endpoint.trim()) { setFormError('Endpoint URL is required'); return; }

    const runnerConfig: Record<string, unknown> = {
      endpoint: endpoint.trim(),
      auth_token: authToken.trim() || undefined,
      timeout_ms: parseInt(timeoutMs) || 300000,
    };
    if (model.trim()) runnerConfig.model = model.trim();
    if (bodyTemplate.trim()) runnerConfig.body_template = bodyTemplate.trim();
    if (outputPath.trim()) runnerConfig.output_path = outputPath.trim();
    if (runnerType === 'mcp') {
      runnerConfig.transport = 'http';
      runnerConfig.tools_enabled = true;
      runnerConfig.tool_name = mcpToolName.trim() || 'execute_task';
    }

    createAgent.mutate({
      name: agentName.trim(),
      type: runnerType === 'http' ? 'http-agent' : 'mcp-agent',
      runner_type: runnerType,
      runner_config: JSON.stringify(runnerConfig),
    });
  };

  return (
    <div className="glass-card" style={{ padding: 20 }}>
      {!showForm ? (
        <button
          onClick={() => setShowForm(true)}
          style={{ background: 'transparent', border: '1px solid rgba(0,242,234,0.3)', color: '#00f2ea', borderRadius: 6, padding: '8px 20px', fontSize: 13, cursor: 'pointer' }}
        >
          + Register HTTP / MCP Agent
        </button>
      ) : (
        <form onSubmit={handleSubmit}>
          <h3 style={{ color: '#ff9f43', fontSize: 14, marginBottom: 16 }}>Register HTTP or MCP Agent</h3>

          <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
              <input type="radio" name="runnerType" value="http" checked={runnerType === 'http'} onChange={() => setRunnerType('http')} />
              <span style={{ color: '#ff9f43', fontSize: 13 }}>HTTP Agent</span>
            </label>
            <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
              <input type="radio" name="runnerType" value="mcp" checked={runnerType === 'mcp'} onChange={() => setRunnerType('mcp')} />
              <span style={{ color: '#a78bfa', fontSize: 13 }}>MCP Agent</span>
            </label>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div style={{ gridColumn: '1 / -1' }}>
              <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>Agent Name *</label>
              <input value={agentName} onChange={(e) => setAgentName(e.target.value)} placeholder="e.g., OpenAI GPT-4o" style={INPUT_STYLE} />
            </div>

            <div style={{ gridColumn: '1 / -1' }}>
              <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>
                Endpoint URL * <span style={{ color: '#666', fontSize: 11 }}>({runnerType === 'http' ? 'e.g., https://api.openai.com/v1/chat/completions' : 'e.g., http://localhost:3000/mcp'})</span>
              </label>
              <input value={endpoint} onChange={(e) => setEndpoint(e.target.value)} placeholder={runnerType === 'http' ? 'https://api.openai.com/v1/chat/completions' : 'http://localhost:3000/mcp'} style={INPUT_STYLE} />
            </div>

            <div style={{ gridColumn: '1 / -1' }}>
              <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>Auth Token / API Key</label>
              <input value={authToken} onChange={(e) => setAuthToken(e.target.value)} placeholder="${OPENAI_API_KEY} or Bearer sk-..." type="password" style={INPUT_STYLE} />
            </div>

            <div>
              <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>Model Name</label>
              <input value={model} onChange={(e) => setModel(e.target.value)} placeholder={runnerType === 'http' ? 'gpt-4o' : 'claude-3'} style={INPUT_STYLE} />
            </div>

            <div>
              <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>Timeout (ms)</label>
              <input value={timeoutMs} onChange={(e) => setTimeoutMs(e.target.value)} placeholder="300000" style={INPUT_STYLE} />
            </div>

            {runnerType === 'mcp' && (
              <div>
                <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>Tool Name</label>
                <input value={mcpToolName} onChange={(e) => setMcpToolName(e.target.value)} placeholder="execute_task" style={INPUT_STYLE} />
              </div>
            )}

            <div style={{ gridColumn: '1 / -1' }}>
              <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>Body Template (Go template syntax)</label>
              <textarea
                value={bodyTemplate}
                onChange={(e) => setBodyTemplate(e.target.value)}
                placeholder={'{\n  "model": "{{.Context.model}}",\n  "messages": [{"role": "user", "content": {{.Context.prompt | toJSON}}}]\n}'}
                rows={4}
                style={{ ...INPUT_STYLE, resize: 'vertical' as const, fontFamily: 'monospace', fontSize: 11 }}
              />
            </div>

            <div style={{ gridColumn: '1 / -1' }}>
              <label style={{ fontSize: 12, color: '#888', display: 'block', marginBottom: 4 }}>Output Path (dot-notation)</label>
              <input value={outputPath} onChange={(e) => setOutputPath(e.target.value)} placeholder="choices[0].message.content" style={INPUT_STYLE} />
            </div>

            <div style={{ gridColumn: '1 / -1' }}>
              <label style={{ fontSize: 12, color: '#666', display: 'block', marginBottom: 4 }}>Config Preview (read-only)</label>
              <pre style={{ background: '#0a0b10', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 6, padding: 10, color: '#555', fontSize: 10, fontFamily: 'monospace', whiteSpace: 'pre-wrap', wordBreak: 'break-all', margin: 0 }}>
                {JSON.stringify({
                  endpoint: endpoint.trim() || '(required)',
                  ...(authToken.trim() ? { auth_token: '***' } : {}),
                  ...(model.trim() ? { model: model.trim() } : {}),
                  ...(bodyTemplate.trim() ? { body_template: '...' } : {}),
                  ...(outputPath.trim() ? { output_path: outputPath.trim() } : {}),
                  timeout_ms: parseInt(timeoutMs) || 300000,
                  ...(runnerType === 'mcp' ? { transport: 'http', tool_name: mcpToolName.trim() || 'execute_task', tools_enabled: true } : {}),
                }, null, 2)}
              </pre>
            </div>
          </div>

          {formError && <div style={{ color: '#ff0050', fontSize: 12, marginTop: 12 }}>{formError}</div>}
          {successMsg && <div style={{ color: '#00ffaa', fontSize: 12, marginTop: 12 }}>{successMsg}</div>}

          <div style={{ display: 'flex', gap: 12, marginTop: 16 }}>
            <button
              type="submit"
              disabled={createAgent.isPending}
              style={{ background: runnerType === 'http' ? '#ff9f43' : '#a78bfa', color: '#0a0b10', border: 'none', borderRadius: 6, padding: '8px 20px', fontWeight: 600, cursor: 'pointer', fontSize: 13 }}
            >
              {createAgent.isPending ? 'Registering...' : 'Register Agent'}
            </button>
            <button
              type="button"
              onClick={() => setShowForm(false)}
              style={{ background: 'transparent', border: '1px solid rgba(255,255,255,0.1)', color: '#888', borderRadius: 6, padding: '8px 20px', cursor: 'pointer', fontSize: 13 }}
            >
              Cancel
            </button>
          </div>
        </form>
      )}
    </div>
  );
}

const INPUT_STYLE: React.CSSProperties = {
  background: '#1a1c28',
  border: '1px solid rgba(255,255,255,0.1)',
  borderRadius: 6,
  padding: '8px 12px',
  color: '#ccc',
  fontSize: 13,
  width: '100%',
  boxSizing: 'border-box' as const,
};
