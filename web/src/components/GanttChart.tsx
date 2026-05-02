import { useMemo, useState } from 'react';
import type { ScheduledTask } from '../types/scheduler';

interface GanttChartProps {
  tasks: ScheduledTask[];
}

const STATUS_COLORS: Record<string, string> = {
  in_progress: '#00f2ea',
  running: '#00f2ea',
  completed: '#00ffaa',
  done: '#00ffaa',
  verified: '#00ffaa',
  blocked: '#ff0050',
  failed: '#ff0050',
  ready: '#ffee00',
  assigned: '#a78bfa',
  review: '#ff6b6b',
  backlog: '#4a4a6a',
  triage: '#ff9f43',
  review_pending: '#ff6b6b',
};

export default function GanttChart({ tasks }: GanttChartProps) {
  const [nowMs] = useState(() => Date.now());

  const { minTime, maxTime, rows } = useMemo(() => {
    if (tasks.length === 0) return { minTime: 0, maxTime: 0, rows: [] };

    const enriched = tasks.map((t) => {
      const start = t.created_at ? new Date(t.created_at).getTime() : nowMs;
      const end = t.updated_at ? new Date(t.updated_at).getTime() : nowMs;
      return { ...t, startMs: start, endMs: Math.max(end, start + 60000) };
    });

    const minTime = Math.min(...enriched.map((r) => r.startMs));
    const maxTime = Math.max(...enriched.map((r) => r.endMs));
    return { minTime, maxTime, rows: enriched };
  }, [nowMs, tasks]);

  const totalMs = maxTime - minTime || 1;
  const chartWidth = 900;

  return (
    <div style={{ overflowX: 'auto', background: 'rgba(20,22,30,0.6)', borderRadius: 12, border: '1px solid rgba(255,255,255,0.06)' }}>
      <div style={{ minWidth: chartWidth + 200, padding: 16 }}>
        <div style={{ display: 'flex', borderBottom: '1px solid rgba(255,255,255,0.08)', paddingBottom: 8, marginBottom: 8 }}>
          <div style={{ width: 200, fontSize: 12, color: '#888', fontWeight: 600 }}>TASK</div>
          <div style={{ flex: 1, fontSize: 12, color: '#888', fontWeight: 600 }}>TIMELINE</div>
        </div>
        {rows.map((row) => {
          const left = ((row.startMs - minTime) / totalMs) * chartWidth;
          const width = Math.max(((row.endMs - row.startMs) / totalMs) * chartWidth, 8);
          const color = STATUS_COLORS[row.status] || '#4a4a6a';
          return (
            <div key={row.id} style={{ display: 'flex', alignItems: 'center', height: 32, marginBottom: 2 }}>
              <div style={{ width: 200, fontSize: 12, color: '#ccc', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', paddingRight: 8 }}>
                {row.title || row.id}
              </div>
              <div style={{ flex: 1, position: 'relative', height: 20 }}>
                <div
                  style={{
                    position: 'absolute',
                    left,
                    width,
                    height: 16,
                    top: 2,
                    background: color,
                    borderRadius: 4,
                    opacity: 0.85,
                    boxShadow: `0 0 8px ${color}44`,
                  }}
                  title={`${row.title || row.id}: ${row.status}`}
                />
              </div>
            </div>
          );
        })}
        {rows.length === 0 && (
          <div style={{ textAlign: 'center', color: '#666', padding: 40 }}>No tasks to display</div>
        )}
      </div>
    </div>
  );
}
