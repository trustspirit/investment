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

function toCandlestickData(points: HistoricalDataPoint[]): CandlestickData<Time>[] {
  return points.map((p) => ({
    time: (new Date(p.timestamp).getTime() / 1000) as Time,
    open: p.open,
    high: p.high,
    low: p.low,
    close: p.close,
  }))
}

function toVolumeData(points: HistoricalDataPoint[]): HistogramData<Time>[] {
  return points.map((p) => ({
    time: (new Date(p.timestamp).getTime() / 1000) as Time,
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

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (entry) {
        chart.applyOptions({
          width: entry.contentRect.width,
          height: entry.contentRect.height,
        })
        chart.timeScale().fitContent()
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
    if (data && candleRef.current && volumeRef.current) {
      candleRef.current.setData(toCandlestickData(data))
      volumeRef.current.setData(toVolumeData(data))
      chartRef.current?.timeScale().fitContent()
    }
  }, [data])

  return (
    <div
      className="flex flex-col"
      style={{ borderBottom: '1px solid var(--border)' }}
    >
      <div className="flex flex-wrap gap-1 px-4 py-3 lg:px-6">
        {RANGES.map((r) => (
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
      <div className="relative h-[400px] px-2">
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
