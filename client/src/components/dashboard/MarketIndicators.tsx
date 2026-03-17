import { useMarketIndicators } from '../../hooks'
import type { MarketIndicator } from '../../types'

function formatIndicatorPrice(indicator: MarketIndicator): string {
  if (indicator.symbol === 'USDKRW=X') return `₩${indicator.price.toFixed(2)}`
  if (indicator.symbol === '^TNX') return `${indicator.price.toFixed(3)}%`
  return indicator.price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function IndicatorItem({ indicator }: { indicator: MarketIndicator }) {
  const isPositive = indicator.change >= 0
  const changeColor = indicator.change === 0 ? 'var(--text-muted)' : isPositive ? 'var(--positive)' : 'var(--negative)'
  const sign = isPositive ? '+' : ''

  return (
    <div className="flex shrink-0 items-center gap-2 px-3">
      <span className="text-xs font-medium" style={{ color: 'var(--text-secondary)' }}>
        {indicator.name}
      </span>
      <span className="text-xs font-bold" style={{ color: 'var(--text-primary)' }}>
        {formatIndicatorPrice(indicator)}
      </span>
      <span className="text-xs font-medium" style={{ color: changeColor }}>
        {sign}{indicator.changePercent.toFixed(2)}%
      </span>
    </div>
  )
}

export function MarketIndicators() {
  const { data: indicators } = useMarketIndicators()

  if (!indicators || indicators.length === 0) return null

  return (
    <div
      className="flex items-center overflow-x-auto border-b py-2"
      style={{ backgroundColor: 'var(--bg-secondary)', borderColor: 'var(--border)' }}
    >
      {indicators.map((ind) => (
        <IndicatorItem key={ind.symbol} indicator={ind} />
      ))}
    </div>
  )
}
