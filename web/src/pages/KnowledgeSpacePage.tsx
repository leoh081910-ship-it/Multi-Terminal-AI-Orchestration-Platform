import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { orgApi } from '../api/orgApi';
import { knowledgeApi, type Document } from '../api/knowledgeApi';

export default function KnowledgeSpacePage() {
  const queryClient = useQueryClient();
  const [orgId, setOrgId] = useState('');
  const [spaceId, setSpaceId] = useState('');
  const [tab, setTab] = useState<'docs' | 'context' | 'messages'>('docs');
  const [error, setError] = useState('');

  // Create space form
  const [spaceName, setSpaceName] = useState('');
  const [spaceDesc, setSpaceDesc] = useState('');

  // Create doc form
  const [docTitle, setDocTitle] = useState('');
  const [docContent, setDocContent] = useState('');
  const [docType, setDocType] = useState('note');

  // Context form
  const [ctxKey, setCtxKey] = useState('');
  const [ctxValue, setCtxValue] = useState('');

  // Message form
  const [msgContent, setMsgContent] = useState('');
  const [msgFrom, setMsgFrom] = useState('');
  const [msgTo, setMsgTo] = useState('');

  const { data: orgs = [] } = useQuery({ queryKey: ['orgs'], queryFn: orgApi.listOrgs });

  const { data: spaces = [] } = useQuery({
    queryKey: ['spaces', orgId],
    queryFn: () => knowledgeApi.listSpaces(orgId),
    enabled: !!orgId,
  });

  const { data: documents = [] } = useQuery({
    queryKey: ['documents', orgId, spaceId],
    queryFn: () => knowledgeApi.listDocuments(orgId, spaceId),
    enabled: !!orgId && !!spaceId,
  });

  const { data: context = [] } = useQuery({
    queryKey: ['context', orgId, spaceId],
    queryFn: () => knowledgeApi.listContext(orgId, spaceId),
    enabled: !!orgId && !!spaceId,
  });

  const { data: messages = [] } = useQuery({
    queryKey: ['messages', orgId, spaceId],
    queryFn: () => knowledgeApi.listMessages(orgId, spaceId, 50),
    enabled: !!orgId && !!spaceId,
  });

  const createSpace = useMutation({
    mutationFn: () => knowledgeApi.createSpace(orgId, { name: spaceName, description: spaceDesc || undefined }),
    onSuccess: () => { setSpaceName(''); setSpaceDesc(''); setError(''); queryClient.invalidateQueries({ queryKey: ['spaces', orgId] }); },
    onError: (e: Error) => setError(e.message),
  });

  const createDoc = useMutation({
    mutationFn: () => knowledgeApi.createDocument(orgId, spaceId, { title: docTitle, content: docContent, type: docType }),
    onSuccess: () => { setDocTitle(''); setDocContent(''); setError(''); queryClient.invalidateQueries({ queryKey: ['documents', orgId, spaceId] }); },
    onError: (e: Error) => setError(e.message),
  });

  const setContextEntry = useMutation({
    mutationFn: () => knowledgeApi.setContext(orgId, spaceId, { key: ctxKey, value: ctxValue }),
    onSuccess: () => { setCtxKey(''); setCtxValue(''); setError(''); queryClient.invalidateQueries({ queryKey: ['context', orgId, spaceId] }); },
    onError: (e: Error) => setError(e.message),
  });

  const sendMsg = useMutation({
    mutationFn: () => knowledgeApi.sendMessage(orgId, spaceId, { from_agent_id: msgFrom, to_agent_id: msgTo || undefined, content: msgContent }),
    onSuccess: () => { setMsgContent(''); setError(''); queryClient.invalidateQueries({ queryKey: ['messages', orgId, spaceId] }); },
    onError: (e: Error) => setError(e.message),
  });

  const inputS: React.CSSProperties = { background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13 };
  const btn = (bg: string): React.CSSProperties => ({ background: bg, color: '#0a0b10', border: 'none', borderRadius: 6, padding: '8px 16px', fontWeight: 600, cursor: 'pointer', fontSize: 12 });

  const tabBtn = (t: typeof tab, label: string, color: string) => (
    <button
      onClick={() => setTab(t)}
      style={{
        background: tab === t ? `${color}22` : 'transparent',
        border: `1px solid ${tab === t ? color : 'rgba(255,255,255,0.08)'}`,
        color: tab === t ? color : '#888',
        borderRadius: 6, padding: '6px 14px', cursor: 'pointer', fontSize: 12, fontWeight: 600,
      }}
    >
      {label}
    </button>
  );

  return (
    <div style={{ padding: 24 }}>
      <h2 style={{ color: '#00f2ea', fontSize: 20, fontWeight: 700, marginBottom: 16 }}>Knowledge Space</h2>
      {error && <div style={{ color: '#ff0050', fontSize: 12, marginBottom: 12 }}>{error}</div>}

      {/* Selectors */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
        <select value={orgId} onChange={(e) => { setOrgId(e.target.value); setSpaceId(''); }} style={{ ...inputS, minWidth: 180 }}>
          <option value="">Select Organization</option>
          {orgs.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
        </select>
        <select value={spaceId} onChange={(e) => setSpaceId(e.target.value)} style={{ ...inputS, minWidth: 200 }}>
          <option value="">Select Space</option>
          {spaces.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>
      </div>

      {/* Create Space */}
      <div className="glass-card" style={{ padding: 20, marginBottom: 20 }}>
        <h3 style={{ color: '#ccc', fontSize: 14, marginBottom: 12 }}>Create Knowledge Space</h3>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          <input placeholder="Name" value={spaceName} onChange={(e) => setSpaceName(e.target.value)} style={{ ...inputS, flex: 1, minWidth: 160 }} />
          <input placeholder="Description" value={spaceDesc} onChange={(e) => setSpaceDesc(e.target.value)} style={{ ...inputS, flex: 2, minWidth: 200 }} />
          <button onClick={() => createSpace.mutate()} disabled={!orgId || !spaceName} style={btn('#00f2ea')}>Create</button>
        </div>
      </div>

      {spaceId && (
        <div className="glass-card" style={{ padding: 20 }}>
          {/* Tabs */}
          <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
            {tabBtn('docs', 'Documents', '#a78bfa')}
            {tabBtn('context', 'Context', '#00f2ea')}
            {tabBtn('messages', 'Messages', '#ff9f43')}
          </div>

          {/* Documents Tab */}
          {tab === 'docs' && (
            <div>
              <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
                <input placeholder="Title" value={docTitle} onChange={(e) => setDocTitle(e.target.value)} style={{ ...inputS, flex: 1, minWidth: 140 }} />
                <select value={docType} onChange={(e) => setDocType(e.target.value)} style={{ ...inputS, width: 110 }}>
                  <option value="note">Note</option>
                  <option value="spec">Spec</option>
                  <option value="decision">Decision</option>
                  <option value="report">Report</option>
                </select>
                <button onClick={() => createDoc.mutate()} disabled={!docTitle || !docContent} style={btn('#a78bfa')}>+</button>
              </div>
              <textarea
                placeholder="Document content..."
                value={docContent}
                onChange={(e) => setDocContent(e.target.value)}
                style={{ ...inputS, width: '100%', minHeight: 80, resize: 'vertical', marginBottom: 12, fontFamily: 'monospace' }}
              />
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {documents.map((d) => (
                  <DocCard key={d.id} doc={d} orgId={orgId} spaceId={spaceId} />
                ))}
                {documents.length === 0 && <div style={{ color: '#555', fontSize: 12, padding: 12 }}>No documents</div>}
              </div>
            </div>
          )}

          {/* Context Tab */}
          {tab === 'context' && (
            <div>
              <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                <input placeholder="Key" value={ctxKey} onChange={(e) => setCtxKey(e.target.value)} style={{ ...inputS, flex: 1 }} />
                <input placeholder="Value" value={ctxValue} onChange={(e) => setCtxValue(e.target.value)} style={{ ...inputS, flex: 2 }} />
                <button onClick={() => setContextEntry.mutate()} disabled={!ctxKey || !ctxValue} style={btn('#00f2ea')}>Set</button>
              </div>
              <div style={{ display: 'grid', gap: 6 }}>
                {context.map((c) => (
                  <div key={c.id} style={{ display: 'flex', gap: 12, padding: '8px 12px', borderRadius: 6, background: 'rgba(255,255,255,0.04)', fontSize: 12 }}>
                    <span style={{ color: '#00f2ea', fontWeight: 600, minWidth: 120 }}>{c.key}</span>
                    <span style={{ color: '#ccc', flex: 1, wordBreak: 'break-all' }}>{c.value}</span>
                    <span style={{ color: '#555', fontSize: 10 }}>{c.updated_by || '—'}</span>
                  </div>
                ))}
                {context.length === 0 && <div style={{ color: '#555', fontSize: 12, padding: 12 }}>No context entries</div>}
              </div>
            </div>
          )}

          {/* Messages Tab */}
          {tab === 'messages' && (
            <div>
              <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
                <input placeholder="From Agent ID" value={msgFrom} onChange={(e) => setMsgFrom(e.target.value)} style={{ ...inputS, width: 140 }} />
                <input placeholder="To Agent ID (optional)" value={msgTo} onChange={(e) => setMsgTo(e.target.value)} style={{ ...inputS, width: 160 }} />
                <input placeholder="Content" value={msgContent} onChange={(e) => setMsgContent(e.target.value)} style={{ ...inputS, flex: 1, minWidth: 160 }} />
                <button onClick={() => sendMsg.mutate()} disabled={!msgFrom || !msgContent} style={btn('#ff9f43')}>Send</button>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6, maxHeight: 400, overflowY: 'auto' }}>
                {messages.map((m) => (
                  <div key={m.id} style={{ padding: '8px 12px', borderRadius: 6, background: 'rgba(255,255,255,0.04)', fontSize: 12 }}>
                    <div style={{ display: 'flex', gap: 8, marginBottom: 4 }}>
                      <span style={{ color: '#ff9f43', fontWeight: 600 }}>{m.from_agent_id}</span>
                      {m.to_agent_id && <span style={{ color: '#888' }}>→ {m.to_agent_id}</span>}
                      <span style={{ color: '#555', marginLeft: 'auto', fontSize: 10 }}>{new Date(m.created_at).toLocaleString()}</span>
                    </div>
                    <div style={{ color: '#ccc' }}>{m.content}</div>
                  </div>
                ))}
                {messages.length === 0 && <div style={{ color: '#555', fontSize: 12, padding: 12 }}>No messages</div>}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function DocCard({ doc, orgId, spaceId }: { doc: Document; orgId: string; spaceId: string }) {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [content, setContent] = useState(doc.content);

  const updateDoc = useMutation({
    mutationFn: () => knowledgeApi.updateDocument(orgId, spaceId, doc.id, { content }),
    onSuccess: () => { setEditing(false); queryClient.invalidateQueries({ queryKey: ['documents', orgId, spaceId] }); },
  });

  const TYPE_COLORS: Record<string, string> = { spec: '#a78bfa', decision: '#ff9f43', report: '#00f2ea', note: '#888' };

  return (
    <div style={{ padding: '10px 12px', borderRadius: 8, background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.06)' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
        <span style={{ fontSize: 10, padding: '2px 6px', borderRadius: 4, background: `${TYPE_COLORS[doc.type] || '#888'}22`, color: TYPE_COLORS[doc.type] || '#888' }}>{doc.type}</span>
        <span style={{ fontSize: 13, fontWeight: 600, color: '#ddd', flex: 1 }}>{doc.title}</span>
        <span style={{ fontSize: 10, color: '#555' }}>v{doc.version}</span>
        <button onClick={() => setEditing(!editing)} style={{ background: 'transparent', border: '1px solid rgba(255,255,255,0.1)', color: '#888', borderRadius: 4, padding: '2px 8px', cursor: 'pointer', fontSize: 10 }}>
          {editing ? 'Cancel' : 'Edit'}
        </button>
      </div>
      {editing ? (
        <div>
          <textarea value={content} onChange={(e) => setContent(e.target.value)} style={{ width: '100%', minHeight: 80, background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: 8, color: '#ccc', fontSize: 12, fontFamily: 'monospace', resize: 'vertical' }} />
          <button onClick={() => updateDoc.mutate()} style={{ ...{ background: '#00f2ea', color: '#0a0b10', border: 'none', borderRadius: 4, padding: '4px 12px', fontWeight: 600, cursor: 'pointer', fontSize: 11 }, marginTop: 6 }}>Save</button>
        </div>
      ) : (
        <div style={{ fontSize: 12, color: '#aaa', whiteSpace: 'pre-wrap', maxHeight: 120, overflow: 'hidden' }}>{doc.content}</div>
      )}
      <div style={{ fontSize: 10, color: '#555', marginTop: 4 }}>
        {doc.author_agent_id && <span>by {doc.author_agent_id} · </span>}
        {new Date(doc.updated_at).toLocaleString()}
      </div>
    </div>
  );
}
