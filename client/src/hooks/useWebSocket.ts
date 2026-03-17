import { useCallback, useEffect, useRef, useState } from 'react'
import type { WSPriceUpdate } from '../types'

type PriceUpdateHandler = (update: WSPriceUpdate) => void

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>()
  const reconnectDelay = useRef(1000)
  const subscribedSymbols = useRef<Set<string>>(new Set())
  const onPriceUpdateRef = useRef<PriceUpdateHandler | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws`)

    ws.onopen = () => {
      setIsConnected(true)
      reconnectDelay.current = 1000
      subscribedSymbols.current.forEach((symbol) => {
        ws.send(JSON.stringify({ type: 'subscribe', symbol }))
      })
    }

    ws.onmessage = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data as string) as WSPriceUpdate
        if (data.type === 'priceUpdate' && onPriceUpdateRef.current) {
          onPriceUpdateRef.current(data)
        }
      } catch {
        /* malformed message */
      }
    }

    ws.onclose = () => {
      setIsConnected(false)
      wsRef.current = null
      const delay = Math.min(reconnectDelay.current, 30000)
      reconnectTimer.current = setTimeout(() => {
        reconnectDelay.current = delay * 2
        connect()
      }, delay)
    }

    ws.onerror = () => {
      ws.close()
    }

    wsRef.current = ws
  }, [])

  useEffect(() => {
    connect()
    return () => {
      clearTimeout(reconnectTimer.current)
      wsRef.current?.close()
    }
  }, [connect])

  const subscribe = useCallback((symbol: string) => {
    subscribedSymbols.current.add(symbol)
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'subscribe', symbol }))
    }
  }, [])

  const unsubscribe = useCallback((symbol: string) => {
    subscribedSymbols.current.delete(symbol)
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'unsubscribe', symbol }))
    }
  }, [])

  const setOnPriceUpdate = useCallback((handler: PriceUpdateHandler) => {
    onPriceUpdateRef.current = handler
  }, [])

  return { subscribe, unsubscribe, isConnected, setOnPriceUpdate }
}
