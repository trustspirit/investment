import { fetchAPI } from './client'
import type { WatchlistItem } from '../types'

export function getWatchlist(): Promise<WatchlistItem[]> {
  return fetchAPI<WatchlistItem[]>('/api/watchlist')
}

export function addToWatchlist(symbol: string, name: string): Promise<WatchlistItem> {
  return fetchAPI<WatchlistItem>('/api/watchlist', {
    method: 'POST',
    body: JSON.stringify({ symbol, name }),
  })
}

export function removeFromWatchlist(symbol: string): Promise<void> {
  return fetchAPI<void>(`/api/watchlist/${encodeURIComponent(symbol)}`, {
    method: 'DELETE',
  })
}

export function reorderWatchlist(symbols: string[]): Promise<void> {
  return fetchAPI<void>('/api/watchlist/reorder', {
    method: 'PUT',
    body: JSON.stringify({ symbols }),
  })
}
