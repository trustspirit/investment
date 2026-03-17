import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Search, TrendingUp, Trash2 } from 'lucide-react'
import { useWatchlist } from '../../hooks'
import { StockSearchModal } from '../search/StockSearchModal'

export default function Sidebar() {
  const [searchOpen, setSearchOpen] = useState(false)
  const navigate = useNavigate()
  const { symbol: activeSymbol } = useParams<{ symbol: string }>()
  const { watchlist, removeFromWatchlist } = useWatchlist()

  return (
    <>
      <aside
        className="fixed left-0 top-0 flex h-screen w-[280px] flex-col"
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
              {watchlist.map((item) => {
                const isActive = activeSymbol === item.symbol
                return (
                  <li key={item.symbol} className="group flex items-center">
                    <button
                      onClick={() => navigate(`/stock/${item.symbol}`)}
                      className="flex flex-1 cursor-pointer items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors"
                      style={{
                        backgroundColor: isActive
                          ? 'rgba(34, 211, 238, 0.1)'
                          : 'transparent',
                        border: 'none',
                        color: isActive ? 'var(--accent-cyan)' : 'var(--text-primary)',
                      }}
                    >
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium">{item.symbol}</div>
                        <div
                          className="truncate text-xs"
                          style={{ color: 'var(--text-muted)' }}
                        >
                          {item.name}
                        </div>
                      </div>
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        removeFromWatchlist(item.symbol)
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
              })}
            </ul>
          )}
        </div>
      </aside>

      {searchOpen && <StockSearchModal onClose={() => setSearchOpen(false)} />}
    </>
  )
}
