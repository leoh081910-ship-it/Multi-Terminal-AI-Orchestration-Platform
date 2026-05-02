import type { ScheduledTask } from '../types/scheduler';

interface SwimLaneBoardProps {
  tasks: ScheduledTask[];
  agents: string[];
}

const COLUMNS = ['backlog', 'ready', 'assigned', 'in_progress', 'review', 'verified', 'blocked', 'done'] as const;

const STATUS_COLORS: Record<string, string> = {
  backlog: '#4a4a6a',
  ready: '#ffee00',
  assigned: '#a78bfa',
  in_progress: '#00f2ea',
  review: '#ff6b6b',
  verified: '#00ffaa',
  blocked: '#ff0050',
  done: '#00ffaa',
};

export default function SwimLaneBoard({ tasks, agents }: SwimLaneBoardProps) {
  const grouped = agents.map((agent) => ({
    agent,
    tasks: COLUMNS.map((col) => ({
      status: col,
      items: tasks.filter((t) => t.owner_agent === agent && t.status === col),
    })),
  }));

  return (
    <div style={{ overflowX: 'auto' }}>
      {/* Header */}
      <div style={{ display: 'flex', minWidth: COLUMNS.length * 180 + 120, borderBottom: '1px solid rgba(255,255,255,0.08)', paddingBottom: 8, marginBottom: 8 }}>
        <div style={{ width: 120, fontSize: 11, color: '#888', fontWeight: 600 }}>AGENT</div>
        {COLUMNS.map((col) => (
          <div key={col} style={{ width: 180, fontSize: 11, color: STATUS_COLORS[col], fontWeight: 600, textTransform: 'uppercase' }}>
            {col}
          </div>
        ))}
      </div>
      {/* Rows */}
      {grouped.map(({ agent, tasks: cols }) => (
        <div key={agent} style={{ display: 'flex', minWidth: COLUMNS.length * 180 + 120, borderBottom: '1px solid rgba(255,255,255,0.04)', minHeight: 48 }}>
          <div style={{ width: 120, fontSize: 12, color: '#ccc', display: 'flex', alignItems: 'center', fontWeight: 600 }}>
            {agent}
          </div>
          {cols.map(({ status, items }) => (
            <div key={status} style={{ width: 180, padding: '4px 4px', display: 'flex', flexWrap: 'wrap', gap: 4, alignContent: 'flex-start' }}>
              {items.map((t) => (
                <div
                  key={t.id}
                  style={{
                    fontSize: 10,
                    padding: '2px 6px',
                    borderRadius: 4,
                    background: 'rgba(255,255,255,0.06)',
                    color: '#ccc',
                    maxWidth: 170,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    borderLeft: `2px solid ${STATUS_COLORS[status]}`,
                  }}
                  title={t.title}
                >
                  {t.title || t.id}
                </div>
              ))}
            </div>
          ))}
        </div>
      ))}
      {agents.length === 0 && (
        <div style={{ textAlign: 'center', color: '#666', padding: 40 }}>No agents configured</div>
      )}
    </div>
  );
}
