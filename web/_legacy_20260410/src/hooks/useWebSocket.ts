import { useEffect, useRef, useState } from 'react'

interface WebSocketMessage {
  type: string
  data: unknown
}

interface UseWebSocketOptions {
  onMessage?: (message: WebSocketMessage) => void
  onConnect?: () => void
  onDisconnect?: () => void
  onError?: (error: Event) => void
}

export function useWebSocket(url: string, options: UseWebSocketOptions = {}) {
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const shouldReconnectRef = useRef(true)
  const optionsRef = useRef(options)

  optionsRef.current = options

  useEffect(() => {
    shouldReconnectRef.current = true

    const connect = () => {
      if (!shouldReconnectRef.current) {
        return
      }

      try {
        const ws = new WebSocket(url)
        wsRef.current = ws

        ws.onopen = () => {
          setIsConnected(true)
          optionsRef.current.onConnect?.()
        }

        ws.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data) as WebSocketMessage
            optionsRef.current.onMessage?.(message)
          } catch (err) {
            console.error('Failed to parse WebSocket message:', err)
          }
        }

        ws.onclose = () => {
          setIsConnected(false)
          optionsRef.current.onDisconnect?.()
          if (shouldReconnectRef.current) {
            reconnectTimeoutRef.current = setTimeout(connect, 3000)
          }
        }

        ws.onerror = (error) => {
          optionsRef.current.onError?.(error)
        }
      } catch (err) {
        console.error('Failed to connect WebSocket:', err)
        if (shouldReconnectRef.current) {
          reconnectTimeoutRef.current = setTimeout(connect, 3000)
        }
      }
    }

    connect()

    return () => {
      shouldReconnectRef.current = false
      if (reconnectTimeoutRef.current !== null) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [url])

  const send = (message: WebSocketMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket not connected, cannot send message')
    }
  }

  return { isConnected, send }
}
