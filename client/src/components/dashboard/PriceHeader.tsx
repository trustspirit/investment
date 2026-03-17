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

  return (
    <div className="flex items-end justify-between px-6 py-5">
      <div>
        <div className="flex items-center gap-3">
          <h1 className="m-0 text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
            {quote.name || quote.symbol}
          </h1>
          <span
            className="rounded px-2 py-0.5 text-sm font-medium"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}
          >
            {quote.symbol}
          </span>
        </div>
        <div className="mt-2 flex items-baseline gap-3">
          <span className="text-4xl font-bold" style={{ color: 'var(--text-primary)' }}>
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
      <div className="text-right text-sm" style={{ color: 'var(--text-muted)' }}>
        <div>Vol: {quote.volume.toLocaleString()}</div>
        <div>MCap: {formatLargeNumber(quote.marketCap, cur)}</div>
      </div>
    </div>
  )
}
