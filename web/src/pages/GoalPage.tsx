import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { goalApi, type TreeNode } from '../api/goalApi';
import { orgApi } from '../api/orgApi';
import TaskTreeView from '../components/TaskTreeView';

export default function GoalPage() {
  const queryClient = useQueryClient();
  const [orgId, setOrgId] = useState('');
  const [goalId, setGoalId] = useState('');
  const [tree, setTree] = useState<TreeNode | null>(null);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');

  const { data: orgs = [] } = useQuery({
    queryKey: ['orgs'],
    queryFn: orgApi.listOrgs,
  });

  const createGoal = useMutation({
    mutationFn: () => goalApi.createGoal(orgId, { title, description }),
    onSuccess: (data) => {
      try {
        const parsed = JSON.parse(data.card_json);
        setGoalId(parsed.id || '');
      } catch { /* ignore */ }
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
    onError: (e: Error) => setError(e.message),
  });

  const loadTree = async () => {
    if (!orgId || !goalId) return;
    try {
      const t = await goalApi.getGoalTree(orgId, goalId);
      setTree(t);
      setError('');
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'failed to load tree');
    }
  };

  return (
    <div style={{ padding: 24 }}>
      <h2 style={{ color: '#00f2ea', fontSize: 20, fontWeight: 700, marginBottom: 16 }}>
        Goal Management
      </h2>

      {error && <div style={{ color: '#ff0050', fontSize: 12, marginBottom: 12 }}>{error}</div>}

      {/* Create Goal */}
      <div className="glass-card" style={{ padding: 20, marginBottom: 20 }}>
        <h3 style={{ color: '#ccc', fontSize: 14, marginBottom: 12 }}>Create Goal</h3>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          <select
            value={orgId}
            onChange={(e) => setOrgId(e.target.value)}
            style={{ background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13 }}
          >
            <option value="">Select Organization</option>
            {orgs.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
          </select>
          <input
            placeholder="Goal title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            style={{ background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13, flex: 1, minWidth: 200 }}
          />
          <input
            placeholder="Description (optional)"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            style={{ background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13, flex: 2, minWidth: 200 }}
          />
          <button
            onClick={() => createGoal.mutate()}
            disabled={!orgId || !title}
            style={{ background: '#00f2ea', color: '#0a0b10', border: 'none', borderRadius: 6, padding: '8px 20px', fontWeight: 600, cursor: 'pointer', fontSize: 13 }}
          >
            Create
          </button>
        </div>
      </div>

      {/* Load Tree */}
      <div className="glass-card" style={{ padding: 20, marginBottom: 20 }}>
        <h3 style={{ color: '#ccc', fontSize: 14, marginBottom: 12 }}>Task Tree</h3>
        <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
          <input
            placeholder="Goal ID"
            value={goalId}
            onChange={(e) => setGoalId(e.target.value)}
            style={{ background: '#1a1c28', border: '1px solid rgba(255,255,255,0.1)', borderRadius: 6, padding: '8px 12px', color: '#ccc', fontSize: 13, flex: 1 }}
          />
          <button
            onClick={loadTree}
            disabled={!orgId || !goalId}
            style={{ background: '#a78bfa', color: '#0a0b10', border: 'none', borderRadius: 6, padding: '8px 20px', fontWeight: 600, cursor: 'pointer', fontSize: 13 }}
          >
            Load Tree
          </button>
        </div>
        {tree && <TaskTreeView tree={tree} />}
      </div>
    </div>
  );
}
