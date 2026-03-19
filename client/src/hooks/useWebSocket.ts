import { useCallback, useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { WSPriceUpdate, StockQuote } from '../types'

export function useWebSocket() {
  const queryClient = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>()
  const reconnectDelay = useRef(1000)
  const subscribedSymbols = useRef<Set<string>>(new Set())
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
        if (data.type === 'priceUpdate' && data.quote.price > 0) {
          const q = data.quote
          queryClient.setQueryData<StockQuote>(['quote', data.symbol], (prev) => {
            if (!prev) return prev
            // Only update price from WS; change/changePercent come from REST refetch
            // to avoid stale WS ticks overwriting correct values
            return { ...prev, price: q.price }
          })
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
  }, [queryClient])

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

  return { subscribe, unsubscribe, isConnected }
}
