import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, X } from 'lucide-react'
import { useStockSearch } from '../../hooks'

interface StockSearchModalProps {
  onClose: () => void
}

export function StockSearchModal({ onClose }: StockSearchModalProps) {
  const [query, setQuery] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()
  const { data: results, isLoading } = useStockSearch(query)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onClose])

  const handleSelect = (symbol: string) => {
    navigate(`/stock/${symbol}`)
    onClose()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]"
      style={{ backgroundColor: 'rgba(0, 0, 0, 0.6)' }}
      onClick={onClose}
    >
      <div
        className="w-full max-w-[520px] overflow-hidden rounded-xl shadow-2xl"
        style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border)' }}
        onClick={(e) => e.stopPropagation()}
      >
        <div
          className="flex items-center gap-3 px-4 py-3"
          style={{ borderBottom: '1px solid var(--border)' }}
        >
          <Search className="h-5 w-5 shrink-0" style={{ color: 'var(--text-muted)' }} />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="종목명 또는 심볼 검색 (예: 삼성전자, AAPL)..."
            className="flex-1 bg-transparent text-sm outline-none"
            style={{ color: 'var(--text-primary)', border: 'none' }}
          />
          <button
            onClick={onClose}
            className="cursor-pointer rounded p-1 transition-colors hover:opacity-80"
            style={{ backgroundColor: 'transparent', border: 'none', color: 'var(--text-muted)' }}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="max-h-100 overflow-y-auto">
          {isLoading && (
            <div className="px-4 py-8 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
              Searching...
            </div>
          )}
          {!isLoading && query.length >= 2 && results && results.length === 0 && (
            <div className="px-4 py-8 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
              No results found
            </div>
          )}
          {!isLoading && query.length < 2 && (
            <div className="px-4 py-8 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
              Type at least 2 characters to search
            </div>
          )}
          {results?.map((result) => (
            <button
              key={result.symbol}
              onClick={() => handleSelect(result.symbol)}
              className="flex w-full cursor-pointer items-center gap-3 px-4 py-3 text-left transition-colors hover:brightness-125"
              style={{
                backgroundColor: 'transparent',
                border: 'none',
                borderBottom: '1px solid var(--border)',
                color: 'var(--text-primary)',
              }}
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold">{result.symbol}</span>
                  <span
                    className="rounded px-1.5 py-0.5 text-xs"
                    style={{
                      backgroundColor: 'var(--bg-tertiary)',
                      color: 'var(--text-muted)',
                    }}
                  >
                    {result.exchange}
                  </span>
                </div>
                <div className="truncate text-xs" style={{ color: 'var(--text-secondary)' }}>
                  {result.name}
                </div>
              </div>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
