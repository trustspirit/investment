import { useRecommendation } from '../../hooks'
import { LoadingSpinner } from '../common/LoadingSpinner'
import { TrendingUp } from 'lucide-react'
import type { RecommendationData, RecommendationTrend } from '../../types'

interface RecommendationPanelProps {
  symbol: string
}

const ratingLabels: Record<string, string> = {
  strong_buy: '적극 매수',
  buy: '매수',
  hold: '중립',
  underperform: '비중축소',
  sell: '매도',
}

function getRatingLabel(key: string): string {
  return ratingLabels[key] ?? key
}

function getRatingColor(key: string): string {
  switch (key) {
    case 'strong_buy':
      return 'var(--positive)'
    case 'buy':
      return '#4ade80'
    case 'hold':
      return 'var(--text-secondary)'
    case 'underperform':
      return '#f97316'
    case 'sell':
      return 'var(--negative)'
    default:
      return 'var(--text-secondary)'
  }
}

function getGaugePosition(mean: number): number {
  // Yahoo: 1 = Strong Buy, 5 = Strong Sell
  // Map to 0% (Strong Sell / left) to 100% (Strong Buy / right)
  if (mean <= 0) return 50
  const pct = ((5 - mean) / 4) * 100
  return Math.max(0, Math.min(100, pct))
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

function TrendBar({ trend }: { trend: RecommendationTrend }) {
  const total = trend.strongBuy + trend.buy + trend.hold + trend.sell + trend.strongSell
  if (total === 0) return null

  const segments = [
    { count: trend.strongBuy, color: 'var(--positive)', label: '적극 매수' },
    { count: trend.buy, color: '#4ade80', label: '매수' },
    { count: trend.hold, color: 'var(--text-muted)', label: '중립' },
    { count: trend.sell, color: '#f97316', label: '매도' },
    { count: trend.strongSell, color: 'var(--negative)', label: '적극 매도' },
  ]

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex h-3 overflow-hidden rounded-full">
        {segments.map((seg) =>
          seg.count > 0 ? (
            <div
              key={seg.label}
              className="transition-all"
              style={{
                width: `${(seg.count / total) * 100}%`,
                backgroundColor: seg.color,
                minWidth: seg.count > 0 ? '4px' : '0',
              }}
            />
          ) : null,
        )}
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-0.5">
        {segments.map((seg) =>
          seg.count > 0 ? (
            <div key={seg.label} className="flex items-center gap-1 text-xs">
              <span
                className="inline-block h-2 w-2 rounded-full"
                style={{ backgroundColor: seg.color }}
              />
              <span style={{ color: 'var(--text-muted)' }}>{seg.label}</span>
              <span className="font-semibold" style={{ color: 'var(--text-primary)' }}>
                {seg.count}
              </span>
            </div>
          ) : null,
        )}
      </div>
    </div>
  )
}

function GaugeIndicator({ mean, ratingKey }: { mean: number; ratingKey: string }) {
  const position = getGaugePosition(mean)
  const color = getRatingColor(ratingKey)

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="flex items-center gap-2">
        <span className="text-2xl font-bold" style={{ color }}>
          {getRatingLabel(ratingKey)}
        </span>
        <span className="text-sm" style={{ color: 'var(--text-muted)' }}>
          ({mean.toFixed(1)})
        </span>
      </div>
      <div className="relative h-2.5 w-full rounded-full" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
        <div
          className="absolute inset-y-0 left-0 rounded-full"
          style={{
            background: 'linear-gradient(90deg, var(--negative), #f97316, var(--text-muted), #4ade80, var(--positive))',
            width: '100%',
            opacity: 0.3,
          }}
        />
        <div
          className="absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full"
          style={{
            left: `${position}%`,
            backgroundColor: color,
            boxShadow: `0 0 8px ${color}`,
            border: '2px solid var(--bg-secondary)',
          }}
        />
      </div>
      <div className="flex w-full justify-between text-xs" style={{ color: 'var(--text-muted)' }}>
        <span>적극 매도</span>
        <span>중립</span>
        <span>적극 매수</span>
      </div>
    </div>
  )
}

function TargetPrices({ data }: { data: RecommendationData }) {
  const cur = data.currency || 'USD'

  if (!data.targetMeanPrice && !data.targetHighPrice && !data.targetLowPrice) {
    return null
  }

  const upside =
    data.targetMeanPrice && data.currentPrice
      ? ((data.targetMeanPrice - data.currentPrice) / data.currentPrice) * 100
      : null

  return (
    <div className="flex flex-col gap-2">
      <h4
        className="text-xs font-semibold uppercase tracking-wider"
        style={{ color: 'var(--text-muted)' }}
      >
        목표 주가
      </h4>
      <div className="grid grid-cols-3 gap-2">
        {data.targetLowPrice != null && (
          <div className="rounded-lg px-3 py-2" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
            <div className="text-xs" style={{ color: 'var(--text-muted)' }}>
              최저
            </div>
            <div className="text-sm font-semibold" style={{ color: 'var(--negative)' }}>
              {formatPrice(data.targetLowPrice, cur)}
            </div>
          </div>
        )}
        {data.targetMeanPrice != null && (
          <div className="rounded-lg px-3 py-2" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
            <div className="text-xs" style={{ color: 'var(--text-muted)' }}>
              평균
            </div>
            <div className="text-sm font-semibold" style={{ color: 'var(--accent-cyan)' }}>
              {formatPrice(data.targetMeanPrice, cur)}
            </div>
            {upside !== null && (
              <div
                className="text-xs font-medium"
                style={{ color: upside >= 0 ? 'var(--positive)' : 'var(--negative)' }}
              >
                {upside >= 0 ? '+' : ''}
                {upside.toFixed(1)}%
              </div>
            )}
          </div>
        )}
        {data.targetHighPrice != null && (
          <div className="rounded-lg px-3 py-2" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
            <div className="text-xs" style={{ color: 'var(--text-muted)' }}>
              최고
            </div>
            <div className="text-sm font-semibold" style={{ color: 'var(--positive)' }}>
              {formatPrice(data.targetHighPrice, cur)}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export function RecommendationContent({ symbol }: RecommendationPanelProps) {
  const { data, isLoading } = useRecommendation(symbol)

  return (
    <div className="flex flex-col gap-4">
      {isLoading && <LoadingSpinner size="sm" />}

      {!isLoading && !data && (
        <p className="py-4 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
          추천 데이터를 불러올 수 없습니다
        </p>
      )}

      {!isLoading && data && (
        <>
          <GaugeIndicator mean={data.recommendationMean} ratingKey={data.recommendationKey} />

          <div className="text-xs" style={{ color: 'var(--text-muted)' }}>
            {data.numberOfAnalysts}명의 애널리스트 의견
          </div>

          {data.trend.length > 0 && <TrendBar trend={data.trend[0]} />}

          <TargetPrices data={data} />
        </>
      )}
    </div>
  )
}

export function RecommendationPanel({ symbol }: RecommendationPanelProps) {
  return (
    <div
      className="flex flex-col overflow-hidden rounded-xl"
      style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border)' }}
    >
      <div
        className="flex items-center gap-2 px-4 py-3"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        <TrendingUp className="h-4 w-4" style={{ color: 'var(--accent-cyan)' }} />
        <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
          투자 제안
        </span>
      </div>
      <div className="px-4 py-3">
        <RecommendationContent symbol={symbol} />
      </div>
    </div>
  )
}
