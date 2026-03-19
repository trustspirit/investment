import { useEffect, useRef, useState } from 'react'
import { createChart, CandlestickSeries, HistogramSeries } from 'lightweight-charts'
import type { IChartApi, ISeriesApi, CandlestickData, HistogramData, Time } from 'lightweight-charts'
import type { HistoricalDataPoint, ChartRange } from '../../types'
import { useStockChart } from '../../hooks'
import { LoadingSpinner } from '../common/LoadingSpinner'

const RANGES: { label: string; value: ChartRange }[] = [
  { label: 'Pre', value: 'pre' },
  { label: '1D', value: '1d' },
  { label: '5D', value: '5d' },
  { label: '1M', value: '1mo' },
  { label: '6M', value: '6mo' },
  { label: '1Y', value: '1y' },
  { label: '5Y', value: '5y' },
  { label: 'MAX', value: 'max' },
]

function isKoreanStock(symbol: string): boolean {
  return symbol.endsWith('.KS') || symbol.endsWith('.KQ')
}

function toLocalTimestamp(timestamp: string): Time {
  const date = new Date(timestamp)
  const utcSeconds = date.getTime() / 1000
  const offsetSeconds = -date.getTimezoneOffset() * 60
  return (utcSeconds + offsetSeconds) as Time
}

// Interval in seconds for each range
const RANGE_INTERVAL: Record<string, number> = {
  'pre': 60,
  '1d': 300,
}

// Session duration in seconds for each market & range
const SESSION_DURATION: Record<string, Record<string, number>> = {
  kr: { '1d': 6.5 * 3600, 'pre': 0.5 * 3600 },
  us: { '1d': 6.5 * 3600, 'pre': 5.5 * 3600 },
}

// Korean market absolute hours including overtime (08:00-20:00)
const KR_MARKET_HOURS: Record<string, { open: [number, number]; close: [number, number] }> = {
  '1d': { open: [8, 0], close: [20, 0] },
}

function getSessionBounds(symbol: string, range: ChartRange, data: HistoricalDataPoint[]): { start: number; end: number } | null {
  if (data.length === 0) return null
  if (range !== '1d' && range !== 'pre') return null

  const isKR = isKoreanStock(symbol)

  if (isKR) {
    // Korean stocks: use absolute KST market hours
    const hours = KR_MARKET_HOURS[range]
    if (!hours) return null
    const refDate = new Date(data[0].timestamp)
    const openDate = new Date(refDate)
    openDate.setHours(hours.open[0], hours.open[1], 0, 0)
    const closeDate = new Date(refDate)
    closeDate.setHours(hours.close[0], hours.close[1], 0, 0)
    const offsetSeconds = -openDate.getTimezoneOffset() * 60
    return {
      start: openDate.getTime() / 1000 + offsetSeconds,
      end: closeDate.getTime() / 1000 + offsetSeconds,
    }
  }

  // Non-Korean stocks: use first data point + session duration
  const duration = SESSION_DURATION['us']?.[range]
  if (!duration) return null
  const firstTime = toLocalTimestamp(data[0].timestamp) as number
  return {
    start: firstTime,
    end: firstTime + duration,
  }
}

function buildWhitespacePoints(symbol: string, range: ChartRange, data: HistoricalDataPoint[]): { time: Time }[] {
  if (data.length === 0) return []
  if (range !== '1d' && range !== 'pre') return []

  const interval = RANGE_INTERVAL[range]
  if (!interval) return []

  const bounds = getSessionBounds(symbol, range, data)
  if (!bounds) return []

  const firstTime = toLocalTimestamp(data[0].timestamp) as number
  const lastTime = toLocalTimestamp(data[data.length - 1].timestamp) as number

  const points: { time: Time }[] = []

  // Pre-data whitespace (e.g. 08:00 → 09:00 for Korean stocks)
  let t = bounds.start
  while (t < firstTime) {
    points.push({ time: t as Time })
    t += interval
  }

  // Post-data whitespace (e.g. 15:30 → 20:00 for Korean stocks)
  t = lastTime + interval
  while (t <= bounds.end) {
    points.push({ time: t as Time })
    t += interval
  }

  return points
}

function toCandlestickData(points: HistoricalDataPoint[]): CandlestickData<Time>[] {
  return points.map((p) => ({
    time: toLocalTimestamp(p.timestamp),
    open: p.open,
    high: p.high,
    low: p.low,
    close: p.close,
  }))
}

function toVolumeData(points: HistoricalDataPoint[]): HistogramData<Time>[] {
  return points.map((p) => ({
    time: toLocalTimestamp(p.timestamp),
    value: p.volume,
    color: p.close >= p.open ? 'rgba(34, 197, 94, 0.4)' : 'rgba(239, 68, 68, 0.4)',
  }))
}

interface ChartPanelProps {
  symbol: string
}

export function ChartPanel({ symbol }: ChartPanelProps) {
  const [range, setRange] = useState<ChartRange>('1d')
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const volumeRef = useRef<ISeriesApi<'Histogram'> | null>(null)
  const { data, isLoading } = useStockChart(symbol, range)

  useEffect(() => {
    if (!containerRef.current) return

    const chart = createChart(containerRef.current, {
      autoSize: true,
      layout: {
        background: { color: 'transparent' },
        textColor: '#8888a0',
      },
      grid: {
        vertLines: { color: '#2a2a3c' },
        horzLines: { color: '#2a2a3c' },
      },
      crosshair: {
        vertLine: { color: '#5c5c72', width: 1, style: 3 },
        horzLine: { color: '#5c5c72', width: 1, style: 3 },
      },
      timeScale: {
        borderColor: '#2a2a3c',
        timeVisible: true,
      },
      rightPriceScale: {
        borderColor: '#2a2a3c',
      },
    })

    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderUpColor: '#22c55e',
      borderDownColor: '#ef4444',
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    })

    const volumeSeries = chart.addSeries(HistogramSeries, {
      priceFormat: { type: 'volume' },
      priceScaleId: 'volume',
    })

    chart.priceScale('volume').applyOptions({
      scaleMargins: { top: 0.8, bottom: 0 },
    })

    chartRef.current = chart
    candleRef.current = candleSeries
    volumeRef.current = volumeSeries

    const observer = new ResizeObserver(() => {
      if (containerRef.current) {
        const { width, height } = containerRef.current.getBoundingClientRect()
        chart.applyOptions({ width, height })
      }
    })
    observer.observe(containerRef.current)

    return () => {
      observer.disconnect()
      chart.remove()
      chartRef.current = null
      candleRef.current = null
      volumeRef.current = null
    }
  }, [])

  useEffect(() => {
    if (data && candleRef.current && volumeRef.current && chartRef.current) {
      const candles = toCandlestickData(data)
      const whitespace = buildWhitespacePoints(symbol, range, data)
      candleRef.current.setData(
        [...candles, ...whitespace].sort((a, b) => (a.time as number) - (b.time as number)) as CandlestickData<Time>[],
      )
      volumeRef.current.setData(toVolumeData(data))

      const bounds = getSessionBounds(symbol, range, data)
      if (bounds) {
        chartRef.current.timeScale().applyOptions({ rightOffset: 0 })
        chartRef.current.timeScale().setVisibleRange({
          from: bounds.start as Time,
          to: bounds.end as Time,
        })
      } else {
        chartRef.current.timeScale().applyOptions({ rightOffset: 0 })
        chartRef.current.timeScale().fitContent()
      }
    }
  }, [data, symbol, range])

  return (
    <div
      className="flex flex-col"
      style={{ borderBottom: '1px solid var(--border)' }}
    >
      <div className="flex flex-wrap gap-1 px-4 py-3 lg:px-6">
        {RANGES.filter((r) => !(r.value === 'pre' && isKoreanStock(symbol))).map((r) => (
          <button
            key={r.value}
            onClick={() => setRange(r.value)}
            className="cursor-pointer rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
            style={{
              backgroundColor: range === r.value ? 'rgba(34, 211, 238, 0.15)' : 'transparent',
              color: range === r.value ? 'var(--accent-cyan)' : 'var(--text-muted)',
              border: 'none',
            }}
          >
            {r.label}
          </button>
        ))}
      </div>
      <div className="relative h-100 px-2">
        {isLoading && (
          <div className="absolute inset-0 z-10 flex items-center justify-center">
            <LoadingSpinner />
          </div>
        )}
        <div ref={containerRef} className="h-full w-full" />
      </div>
    </div>
  )
}
