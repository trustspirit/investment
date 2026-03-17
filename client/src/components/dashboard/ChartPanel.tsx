import { useEffect, useRef, useState } from 'react'
import { createChart, AreaSeries } from 'lightweight-charts'
import type { IChartApi, ISeriesApi, AreaData, Time } from 'lightweight-charts'
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

function toChartData(points: HistoricalDataPoint[]): AreaData<Time>[] {
  return points.map((p) => ({
    time: (new Date(p.timestamp).getTime() / 1000) as Time,
    value: p.close,
  }))
}

interface ChartPanelProps {
  symbol: string
}

export function ChartPanel({ symbol }: ChartPanelProps) {
  const [range, setRange] = useState<ChartRange>('1d')
  const containerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Area'> | null>(null)
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

    const series = chart.addSeries(AreaSeries, {
      lineColor: '#22d3ee',
      topColor: 'rgba(34, 211, 238, 0.3)',
      bottomColor: 'rgba(34, 211, 238, 0.02)',
      lineWidth: 2,
    })

    chartRef.current = chart
    seriesRef.current = series

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (entry) {
        chart.applyOptions({
          width: entry.contentRect.width,
          height: entry.contentRect.height,
        })
      }
    })
    observer.observe(containerRef.current)

    return () => {
      observer.disconnect()
      chart.remove()
      chartRef.current = null
      seriesRef.current = null
    }
  }, [])

  useEffect(() => {
    if (data && seriesRef.current) {
      const chartData = toChartData(data)
      seriesRef.current.setData(chartData)
      chartRef.current?.timeScale().fitContent()
    }
  }, [data])

  return (
    <div
      className="flex flex-col"
      style={{ borderBottom: '1px solid var(--border)' }}
    >
      <div className="flex gap-1 px-6 py-3">
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
