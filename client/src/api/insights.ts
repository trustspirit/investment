import { fetchAPI } from './client'
import type { AIInsight, AITradeStrategy } from '../types'

export async function getInsight(symbol: string): Promise<AIInsight | null> {
  const res = await fetch(`/api/insights/${encodeURIComponent(symbol)}`)
  if (res.status === 204) {
    return null
  }
  if (!res.ok) {
    throw new Error(`Failed to fetch insight: ${res.status}`)
  }
  return res.json() as Promise<AIInsight>
}

export function generateInsight(symbol: string): Promise<AIInsight> {
  return fetchAPI<AIInsight>(`/api/insights/${encodeURIComponent(symbol)}/generate`, {
    method: 'POST',
  })
}

export function generateStrategy(symbol: string): Promise<AITradeStrategy> {
  return fetchAPI<AITradeStrategy>(`/api/insights/${encodeURIComponent(symbol)}/strategy`, {
    method: 'POST',
  })
}
