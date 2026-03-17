import { useAIStrategy } from '../../hooks'
import { LoadingSpinner } from '../common/LoadingSpinner'
import { Target, TrendingUp, TrendingDown, ShieldAlert, Zap } from 'lucide-react'
import type { AITradeStrategy, PriceRange, TimingAnalysis } from '../../types'

interface AIStrategyPanelProps {
  symbol: string
}

function currencySymbol(currency: string): string {
  if (currency === 'KRW') return '₩'
  if (currency === 'JPY') return '¥'
  if (currency === 'EUR') return '€'
  if (currency === 'GBP') return '£'
  return '$'
}

function formatPrice(price: number, currency: string): string {
  const sym = currencySymbol(currency)
  if (currency === 'KRW') return `${sym}${Math.round(price).toLocaleString()}`
  return `${sym}${price.toFixed(2)}`
}

function signalColor(signal: string): string {
  if (signal === '적극매수') return 'var(--positive)'
  if (signal === '매수') return '#4ade80'
  if (signal === '중립') return 'var(--text-secondary)'
  if (signal === '매도') return '#f97316'
  if (signal === '적극매도') return 'var(--negative)'
  return 'var(--text-secondary)'
}

function signalBgColor(signal: string): string {
  if (signal === '적극매수') return 'rgba(34, 197, 94, 0.15)'
  if (signal === '매수') return 'rgba(74, 222, 128, 0.15)'
  if (signal === '중립') return 'rgba(136, 136, 160, 0.15)'
  if (signal === '매도') return 'rgba(249, 115, 22, 0.15)'
  if (signal === '적극매도') return 'rgba(239, 68, 68, 0.15)'
  return 'rgba(136, 136, 160, 0.15)'
}

function PriceRangeDisplay({ label, range, currency, color }: { label: string; range: PriceRange; currency: string; color: string }) {
  return (
    <div className="rounded-lg px-3 py-2.5" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
      <div className="mb-1 text-xs font-medium" style={{ color: 'var(--text-muted)' }}>{label}</div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-sm font-bold" style={{ color }}>
          {formatPrice(range.low, currency)}
        </span>
        <span className="text-xs" style={{ color: 'var(--text-muted)' }}>~</span>
        <span className="text-sm font-bold" style={{ color }}>
          {formatPrice(range.high, currency)}
        </span>
      </div>
      <p className="mt-1 text-xs leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
        {range.reason}
      </p>
    </div>
  )
}

function TimingDisplay({ label, timing, icon }: { label: string; timing: TimingAnalysis; icon: React.ReactNode }) {
  return (
    <div className="rounded-lg px-3 py-2.5" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
      <div className="mb-1 flex items-center gap-1.5">
        {icon}
        <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--text-muted)' }}>
          {label}
        </span>
        <span className="ml-auto rounded px-1.5 py-0.5 text-xs" style={{ backgroundColor: 'rgba(34, 211, 238, 0.1)', color: 'var(--accent-cyan)' }}>
          {timing.timeframe}
        </span>
      </div>
      <p className="mb-1.5 text-xs leading-relaxed" style={{ color: 'var(--text-primary)' }}>
        {timing.recommendation}
      </p>
      {timing.conditions.length > 0 && (
        <ul className="flex flex-col gap-0.5 pl-0">
          {timing.conditions.map((c, i) => (
            <li key={i} className="flex gap-1.5 text-xs" style={{ color: 'var(--text-secondary)' }}>
              <span style={{ color: 'var(--accent-cyan)' }}>•</span>
              {c}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function ConfidenceMeter({ confidence }: { confidence: number }) {
  const color = confidence >= 70 ? 'var(--positive)' : confidence >= 40 ? 'var(--accent-cyan)' : 'var(--negative)'
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 flex-1 overflow-hidden rounded-full" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
        <div
          className="h-full rounded-full transition-all"
          style={{ width: `${confidence}%`, backgroundColor: color }}
        />
      </div>
      <span className="text-xs font-bold" style={{ color }}>{confidence}%</span>
    </div>
  )
}

function StrategyContent({ strategy }: { strategy: AITradeStrategy }) {
  const cur = strategy.currency || 'USD'

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <span
          className="rounded-lg px-3 py-1.5 text-sm font-bold"
          style={{ backgroundColor: signalBgColor(strategy.signal), color: signalColor(strategy.signal) }}
        >
          {strategy.signal}
        </span>
        <div className="flex-1">
          <div className="mb-0.5 text-xs" style={{ color: 'var(--text-muted)' }}>분석 확신도</div>
          <ConfidenceMeter confidence={strategy.confidence} />
        </div>
      </div>

      <div className="text-xs" style={{ color: 'var(--text-muted)' }}>
        현재가: <span style={{ color: 'var(--text-primary)' }}>{formatPrice(strategy.currentPrice, cur)}</span>
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <PriceRangeDisplay label="진입 가격대" range={strategy.entryPrice} currency={cur} color="var(--accent-cyan)" />
        <PriceRangeDisplay label="손절 가격대" range={strategy.stopLoss} currency={cur} color="var(--negative)" />
        <PriceRangeDisplay label="익절 가격대" range={strategy.takeProfit} currency={cur} color="var(--positive)" />
      </div>

      <div className="flex items-center gap-2 rounded-lg px-3 py-2" style={{ backgroundColor: 'rgba(167, 139, 250, 0.1)' }}>
        <ShieldAlert className="h-4 w-4" style={{ color: 'var(--accent-purple)' }} />
        <span className="text-xs" style={{ color: 'var(--text-muted)' }}>리스크/보상:</span>
        <span className="text-xs font-bold" style={{ color: 'var(--accent-purple)' }}>{strategy.riskReward}</span>
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <TimingDisplay
          label="매수 타이밍"
          timing={strategy.buyTiming}
          icon={<TrendingUp className="h-3.5 w-3.5" style={{ color: 'var(--positive)' }} />}
        />
        <TimingDisplay
          label="매도 타이밍"
          timing={strategy.sellTiming}
          icon={<TrendingDown className="h-3.5 w-3.5" style={{ color: 'var(--negative)' }} />}
        />
      </div>

      <div className="rounded-lg px-3 py-2.5" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
        <h4 className="mb-1 text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--text-muted)' }}>
          시장 상황
        </h4>
        <p className="text-xs leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
          {strategy.marketCondition}
        </p>
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <div className="rounded-lg px-3 py-2.5" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
          <h4 className="mb-1 text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--accent-cyan)' }}>
            단기 전망
          </h4>
          <p className="text-xs leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
            {strategy.shortTermView}
          </p>
        </div>
        <div className="rounded-lg px-3 py-2.5" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
          <h4 className="mb-1 text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--accent-purple)' }}>
            중기 전망
          </h4>
          <p className="text-xs leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
            {strategy.midTermView}
          </p>
        </div>
      </div>

      {strategy.analysisBasis.length > 0 && (
        <div>
          <h4 className="mb-1.5 text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--text-muted)' }}>
            판단 근거
          </h4>
          <ul className="flex flex-col gap-1 pl-0">
            {strategy.analysisBasis.map((basis, i) => (
              <li key={i} className="flex gap-2 text-xs" style={{ color: 'var(--text-secondary)' }}>
                <span style={{ color: 'var(--accent-cyan)' }}>•</span>
                {basis}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="rounded-lg px-3 py-2" style={{ backgroundColor: 'rgba(239, 68, 68, 0.05)', border: '1px solid rgba(239, 68, 68, 0.15)' }}>
        <p className="text-xs leading-relaxed" style={{ color: 'var(--text-muted)' }}>
          {strategy.disclaimer}
        </p>
      </div>

      <div className="flex items-center justify-between text-xs" style={{ color: 'var(--text-muted)' }}>
        <span>via {strategy.provider}</span>
        <span>{new Date(strategy.analysisTime).toLocaleString()}</span>
      </div>
    </div>
  )
}

export function AIStrategyContent({ symbol }: AIStrategyPanelProps) {
  const { strategy, isGenerating, generate, generateError } = useAIStrategy(symbol)

  return (
    <div className="flex flex-col gap-3">
      <div className="flex justify-end">
        <button
          onClick={() => generate()}
          disabled={isGenerating}
          className="cursor-pointer rounded-md px-3 py-1.5 text-xs font-medium transition-colors hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-50"
          style={{
            backgroundColor: 'rgba(34, 211, 238, 0.15)',
            color: 'var(--accent-cyan)',
            border: 'none',
          }}
        >
          {isGenerating ? '분석 중...' : '전략 분석'}
        </button>
      </div>

      {isGenerating && (
        <div className="flex flex-col items-center gap-2 py-8">
          <LoadingSpinner size="sm" />
          <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
            AI가 종합적인 투자 전략을 분석하고 있습니다...
          </p>
        </div>
      )}

      {!isGenerating && generateError && (
        <div className="py-4 text-center">
          <p className="text-sm" style={{ color: 'var(--negative)' }}>
            전략 생성에 실패했습니다
          </p>
          <p className="mt-1 text-xs" style={{ color: 'var(--text-muted)' }}>
            {generateError.message}
          </p>
        </div>
      )}

      {!isGenerating && !strategy && !generateError && (
        <div className="py-6 text-center">
          <Zap className="mx-auto mb-2 h-8 w-8" style={{ color: 'var(--text-muted)' }} />
          <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
            전략 분석 버튼을 눌러 AI 투자 전략을 생성하세요
          </p>
          <p className="mt-1 text-xs" style={{ color: 'var(--text-muted)' }}>
            진입/손절/익절 가격대, 매수·매도 타이밍 등을 분석합니다
          </p>
        </div>
      )}

      {!isGenerating && strategy && <StrategyContent strategy={strategy} />}
    </div>
  )
}

export function AIStrategyPanel({ symbol }: AIStrategyPanelProps) {
  return (
    <div
      className="flex flex-col overflow-hidden rounded-xl"
      style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border)' }}
    >
      <div
        className="flex items-center gap-2 px-4 py-3"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        <Target className="h-4 w-4" style={{ color: 'var(--accent-cyan)' }} />
        <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
          AI 투자 전략
        </span>
      </div>
      <div className="max-h-[600px] overflow-y-auto px-4 py-3">
        <AIStrategyContent symbol={symbol} />
      </div>
    </div>
  )
}
