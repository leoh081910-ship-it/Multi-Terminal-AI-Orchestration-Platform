import { useEffect, useRef, useState, useMemo, useReducer } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import type { WSMessage } from '../hooks/useWebSocket';

interface LogEntry {
  id: string;
  timestamp: string;
  level: string;
  message: string;
  source?: string;
}

interface LogState {
  entries: LogEntry[];
  maxLines: number;
}

type LogAction =
  | { type: 'append'; entry: LogEntry; maxLines: number }
  | { type: 'clear' };

function logReducer(state: LogState, action: LogAction): LogState {
  switch (action.type) {
    case 'append': {
      const next = [...state.entries, action.entry];
      return { entries: next.length > action.maxLines ? next.slice(-action.maxLines) : next, maxLines: action.maxLines };
    }
    case 'clear':
      return { ...state, entries: [] };
  }
}

interface LogViewerProps {
  projectId?: string;
  taskId?: string;
  maxLines?: number;
  height?: string;
}

export default function LogViewer({ projectId, taskId, maxLines = 500, height = '400px' }: LogViewerProps) {
  const { lastMessage, connected } = useWebSocket(projectId);
  const [{ entries: logs }, dispatchLogs] = useReducer(logReducer, { entries: [], maxLines });
  const [search, setSearch] = useState('');
  const [autoScroll, setAutoScroll] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  // Process WebSocket messages into log entries
  useEffect(() => {
    if (!lastMessage) return;

    const entry = wsToLogEntry(lastMessage);
    if (!entry) return;

    // Filter by taskId if specified
    if (taskId && lastMessage.task_id && lastMessage.task_id !== taskId) return;

    dispatchLogs({ type: 'append', entry, maxLines });
  }, [lastMessage, taskId, maxLines]);

  // Auto-scroll
  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  // Detect manual scroll
  const handleScroll = () => {
    if (!containerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    const atBottom = scrollHeight - scrollTop - clientHeight < 30;
    setAutoScroll(atBottom);
  };

  const filtered = useMemo(() => {
    if (!search.trim()) return logs;
    const q = search.toLowerCase();
    return logs.filter(l => l.message.toLowerCase().includes(q) || l.level.toLowerCase().includes(q));
  }, [logs, search]);

  return (
    <div className="bg-gray-900 rounded-lg overflow-hidden border border-gray-700">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-2 bg-gray-800 border-b border-gray-700">
        <div className="flex items-center gap-3">
          <div className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
          <span className="text-xs text-gray-400">
            {filtered.length} entries{search ? ` (filtered)` : ''}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder="Search logs..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="px-2 py-1 bg-gray-700 text-gray-200 text-xs rounded border border-gray-600 focus:outline-none focus:border-blue-500 w-48"
          />
          <button
            onClick={() => dispatchLogs({ type: 'clear' })}
            className="px-2 py-1 text-xs text-gray-400 hover:text-gray-200 bg-gray-700 rounded"
          >
            Clear
          </button>
          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`px-2 py-1 text-xs rounded ${autoScroll ? 'text-blue-400 bg-blue-900/30' : 'text-gray-400 bg-gray-700'}`}
          >
            {autoScroll ? 'Auto-scroll ON' : 'Auto-scroll OFF'}
          </button>
        </div>
      </div>

      {/* Log content */}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="overflow-y-auto font-mono text-xs leading-5 p-2"
        style={{ height }}
      >
        {filtered.length === 0 ? (
          <div className="text-gray-600 text-center py-8">Waiting for log events...</div>
        ) : (
          filtered.map(entry => (
            <div key={entry.id} className="flex gap-2 hover:bg-gray-800/50 px-1">
              <span className="text-gray-600 shrink-0 w-20">{formatTime(entry.timestamp)}</span>
              <span className={`shrink-0 w-14 ${levelColor(entry.level)}`}>{entry.level.toUpperCase()}</span>
              {entry.source && <span className="text-gray-500 shrink-0">[{entry.source}]</span>}
              <span className="text-gray-300 break-all">{entry.message}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function wsToLogEntry(msg: WSMessage): LogEntry | null {
  const id = String(Date.now()) + '-' + String(Math.random()).slice(2, 8);
  const data = msg.data as Record<string, string> | undefined;

  switch (msg.type) {
    case 'task.state_changed':
      return { id, timestamp: msg.timestamp, level: 'info', message: `Task ${msg.task_id} state changed`, source: msg.task_id };
    case 'task.created':
      return { id, timestamp: msg.timestamp, level: 'info', message: `Task created: ${msg.task_id}`, source: 'scheduler' };
    case 'task.completed':
      return { id, timestamp: msg.timestamp, level: 'info', message: `Task completed: ${msg.task_id}`, source: 'scheduler' };
    case 'task.failed':
      return { id, timestamp: msg.timestamp, level: 'error', message: `Task failed: ${msg.task_id}${data?.error ? ': ' + data.error : ''}`, source: 'scheduler' };
    case 'agent.heartbeat':
      return { id, timestamp: msg.timestamp, level: 'debug', message: `Agent ${msg.agent_id} heartbeat`, source: 'heartbeat' };
    case 'agent.offline':
      return { id, timestamp: msg.timestamp, level: 'warn', message: `Agent ${msg.agent_id} went offline`, source: 'heartbeat' };
    default:
      return { id, timestamp: msg.timestamp, level: 'info', message: JSON.stringify(msg), source: msg.type };
  }
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return ts;
  }
}

function levelColor(level: string): string {
  switch (level) {
    case 'error': return 'text-red-400';
    case 'warn': return 'text-amber-400';
    case 'info': return 'text-blue-400';
    case 'debug': return 'text-gray-500';
    default: return 'text-gray-400';
  }
}
