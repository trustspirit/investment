import { Star } from 'lucide-react'
import { useWatchlist } from '../../hooks'
import type { StockQuote } from '../../types'

interface PriceHeaderProps {
  quote: StockQuote
}

function currencySymbol(currency: string): string {
  if (currency === 'KRW') return '₩'
  if (currency === 'JPY') return '¥'
  if (currency === 'EUR') return '€'
  if (currency === 'GBP') return '£'
  if (currency === 'CNY') return '¥'
  return '$'
}

function formatPrice(price: number, currency: string): string {
  const sym = currencySymbol(currency)
  if (currency === 'KRW') return `${sym}${Math.round(price).toLocaleString()}`
  return `${sym}${price.toFixed(2)}`
}

function formatLargeNumber(num: number, currency: string): string {
  const sym = currencySymbol(currency)
  if (num >= 1e12) return `${sym}${(num / 1e12).toFixed(2)}T`
  if (num >= 1e9) return `${sym}${(num / 1e9).toFixed(2)}B`
  if (num >= 1e6) return `${sym}${(num / 1e6).toFixed(2)}M`
  if (num >= 1e3) return `${sym}${(num / 1e3).toFixed(2)}K`
  return `${sym}${num.toLocaleString()}`
}

export function PriceHeader({ quote }: PriceHeaderProps) {
  const isPositive = quote.change >= 0
  const changeColor = isPositive ? 'var(--positive)' : 'var(--negative)'
  const sign = isPositive ? '+' : ''
  const cur = quote.currency || 'USD'
  const { isInWatchlist, addToWatchlist, removeFromWatchlist, isAdding, isRemoving } = useWatchlist()

  const inWatchlist = isInWatchlist(quote.symbol)
  const toggleWatchlist = () => {
    if (inWatchlist) {
      removeFromWatchlist(quote.symbol)
    } else {
      addToWatchlist({ symbol: quote.symbol, name: quote.name || quote.symbol })
    }
  }

  return (
    <div className="flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-end sm:justify-between lg:px-6 lg:py-5">
      <div className="min-w-0">
        <div className="flex items-center gap-3">
          <h1 className="m-0 truncate text-xl font-bold sm:text-2xl" style={{ color: 'var(--text-primary)' }}>
            {quote.name || quote.symbol}
          </h1>
          <span
            className="rounded px-2 py-0.5 text-sm font-medium"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}
          >
            {quote.symbol}
          </span>
          <button
            onClick={toggleWatchlist}
            disabled={isAdding || isRemoving}
            className="cursor-pointer rounded p-1.5 transition-colors hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40"
            style={{ backgroundColor: 'transparent', border: 'none' }}
            title={inWatchlist ? 'Remove from watchlist' : 'Add to watchlist'}
          >
            <Star
              className="h-5 w-5"
              style={{
                color: inWatchlist ? 'var(--warning, #f59e0b)' : 'var(--text-muted)',
                fill: inWatchlist ? 'var(--warning, #f59e0b)' : 'none',
              }}
            />
          </button>
        </div>
        <div className="mt-2 flex items-baseline gap-3">
          <span className="text-3xl font-bold sm:text-4xl" style={{ color: 'var(--text-primary)' }}>
            {formatPrice(quote.price, cur)}
          </span>
          <span className="text-lg font-semibold" style={{ color: changeColor }}>
            {sign}
            {quote.change.toFixed(2)} ({sign}
            {quote.changePercent.toFixed(2)}%)
          </span>
        </div>
        {(quote.preMarket || quote.postMarket) && (
          <div className="mt-1 flex gap-4 text-xs" style={{ color: 'var(--text-muted)' }}>
            {quote.preMarket && <span>Pre-market: {formatPrice(quote.preMarket, cur)}</span>}
            {quote.postMarket && <span>After-hours: {formatPrice(quote.postMarket, cur)}</span>}
          </div>
        )}
      </div>
      <div className="shrink-0 text-right text-sm" style={{ color: 'var(--text-muted)' }}>
        <div>Vol: {quote.volume.toLocaleString()}</div>
        <div>MCap: {formatLargeNumber(quote.marketCap, cur)}</div>
      </div>
    </div>
  )
}
