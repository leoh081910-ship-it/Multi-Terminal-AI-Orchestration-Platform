import { useEffect, useRef, useState, useCallback } from 'react';

export interface WSMessage {
  type: string;
  project_id?: string;
  task_id?: string;
  data?: unknown;
  timestamp: string;
}

const WS_URL = (import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws').replace(/\/$/, '');

function getAuthToken(): string | null {
  const match = document.cookie.match(/(?:^|;\s*)auth_token=([^;]*)/);
  if (match) return match[1];
  const stored = localStorage.getItem('auth_token');
  if (stored) return stored;
  return null;
}

export function useWebSocket(projectId?: string) {
  const wsRef = useRef<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const connectRef = useRef<() => void>(() => {});

  const scheduleReconnect = useCallback(() => {
    clearTimeout(reconnectTimer.current);
    reconnectTimer.current = setTimeout(() => {
      connectRef.current();
    }, 3000);
  }, []);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    const token = getAuthToken();
    const params = new URLSearchParams();
    if (projectId) params.set('project', projectId);
    if (token) params.set('token', token);

    const qs = params.toString() ? `?${params.toString()}` : '';
    const ws = new WebSocket(`${WS_URL}${qs}`);

    ws.onopen = () => {
      setConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data);
        setLastMessage(msg);
      } catch {
        // ignore non-JSON messages
      }
    };

    ws.onclose = () => {
      setConnected(false);
      wsRef.current = null;
      scheduleReconnect();
    };

    ws.onerror = () => {
      ws.close();
    };

    wsRef.current = ws;
  }, [projectId, scheduleReconnect]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [connect]);

  return { connected, lastMessage };
}
