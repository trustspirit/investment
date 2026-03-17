import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Search, TrendingUp, Trash2, Menu, X } from 'lucide-react'
import { useWatchlist } from '../../hooks'
import { getQuote } from '../../api/stocks'
import { StockSearchModal } from '../search/StockSearchModal'
import type { StockQuote, WatchlistItem } from '../../types'

function WatchlistItemPrice({ symbol }: { symbol: string }) {
  const { data: quote } = useQuery<StockQuote>({
    queryKey: ['quote', symbol],
    queryFn: () => getQuote(symbol),
    enabled: !!symbol,
    refetchInterval: 30_000,
    staleTime: 15_000,
  })

  if (!quote) return null

  const isPositive = quote.change >= 0
  const changeColor = quote.change === 0 ? 'var(--text-muted)' : isPositive ? 'var(--positive)' : 'var(--negative)'
  const currSym = quote.currency === 'KRW' ? '₩' : '$'
  const priceStr = quote.currency === 'KRW'
    ? `${currSym}${Math.round(quote.price).toLocaleString()}`
    : `${currSym}${quote.price.toFixed(2)}`
  const sign = isPositive ? '+' : ''

  return (
    <div className="flex flex-col items-end">
      <span className="text-xs font-semibold" style={{ color: 'var(--text-primary)' }}>
        {priceStr}
      </span>
      <span className="text-xs" style={{ color: changeColor }}>
        {sign}{quote.changePercent.toFixed(2)}%
      </span>
    </div>
  )
}

function WatchlistEntry({ item, isActive, onSelect, onRemove }: {
  item: WatchlistItem
  isActive: boolean
  onSelect: () => void
  onRemove: () => void
}) {
  return (
    <li className="group flex items-center">
      <button
        onClick={onSelect}
        className="flex flex-1 cursor-pointer items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors"
        style={{
          backgroundColor: isActive ? 'rgba(34, 211, 238, 0.1)' : 'transparent',
          border: 'none',
          color: isActive ? 'var(--accent-cyan)' : 'var(--text-primary)',
        }}
      >
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium">{item.symbol}</div>
          <div className="truncate text-xs" style={{ color: 'var(--text-muted)' }}>
            {item.name}
          </div>
        </div>
        <WatchlistItemPrice symbol={item.symbol} />
      </button>
      <button
        onClick={(e) => {
          e.stopPropagation()
          onRemove()
        }}
        className="mr-1 cursor-pointer rounded p-1.5 opacity-0 transition-opacity group-hover:opacity-100"
        style={{
          backgroundColor: 'transparent',
          border: 'none',
          color: 'var(--text-muted)',
        }}
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
    </li>
  )
}

export default function Sidebar() {
  const [searchOpen, setSearchOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(true)
  const navigate = useNavigate()
  const { symbol: activeSymbol } = useParams<{ symbol: string }>()
  const { watchlist, removeFromWatchlist } = useWatchlist()

  return (
    <>
      <button
        onClick={() => setCollapsed((c) => !c)}
        className="fixed left-3 top-3 z-50 cursor-pointer rounded-lg p-2 lg:hidden"
        style={{
          backgroundColor: 'var(--bg-secondary)',
          border: '1px solid var(--border)',
          color: 'var(--text-primary)',
        }}
      >
        {collapsed ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
      </button>

      {!collapsed && (
        <div
          className="fixed inset-0 z-30 bg-black/50 lg:hidden"
          onClick={() => setCollapsed(true)}
        />
      )}

      <aside
        className={`fixed left-0 top-0 z-40 flex h-screen w-[280px] flex-col transition-transform duration-200 lg:translate-x-0 ${collapsed ? '-translate-x-full' : 'translate-x-0'}`}
        style={{
          backgroundColor: 'var(--bg-secondary)',
          borderRight: '1px solid var(--border)',
        }}
      >
        <div
          className="flex items-center gap-2 px-5 py-5"
          style={{ borderBottom: '1px solid var(--border)' }}
        >
          <TrendingUp className="h-6 w-6" style={{ color: 'var(--accent-cyan)' }} />
          <span className="text-lg font-semibold" style={{ color: 'var(--text-primary)' }}>
            InvestDash
          </span>
        </div>

        <div className="px-3 py-3">
          <button
            onClick={() => setSearchOpen(true)}
            className="flex w-full cursor-pointer items-center gap-2 rounded-lg px-3 py-2.5 text-sm transition-colors"
            style={{
              backgroundColor: 'var(--bg-tertiary)',
              color: 'var(--text-secondary)',
              border: '1px solid var(--border)',
            }}
          >
            <Search className="h-4 w-4" />
            Search stocks...
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-3 py-2">
          <p
            className="mb-2 px-3 text-xs font-medium uppercase tracking-wider"
            style={{ color: 'var(--text-muted)' }}
          >
            Watchlist
          </p>
          {watchlist.length === 0 ? (
            <p className="px-3 py-4 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
              No stocks in watchlist
            </p>
          ) : (
            <ul className="flex flex-col gap-0.5">
              {watchlist.map((item) => (
                <WatchlistEntry
                  key={item.symbol}
                  item={item}
                  isActive={activeSymbol === item.symbol}
                  onSelect={() => {
                    navigate(`/stock/${item.symbol}`)
                    setCollapsed(true)
                  }}
                  onRemove={() => removeFromWatchlist(item.symbol)}
                />
              ))}
            </ul>
          )}
        </div>
      </aside>

      {searchOpen && <StockSearchModal onClose={() => setSearchOpen(false)} />}
    </>
  )
}
