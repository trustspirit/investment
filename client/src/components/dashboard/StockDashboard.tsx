import { useEffect } from 'react'
import { useStockQuote, useWebSocket } from '../../hooks'
import { PriceHeader } from './PriceHeader'
import { ChartPanel } from './ChartPanel'
import { MetricsPanel } from './MetricsPanel'
import { NewsPanel } from './NewsPanel'
import { AIInsightPanel } from './AIInsightPanel'
import { LoadingSpinner } from '../common/LoadingSpinner'
import { ErrorMessage } from '../common/ErrorMessage'

interface StockDashboardProps {
  symbol: string
}

export function StockDashboard({ symbol }: StockDashboardProps) {
  const { data: quote, isLoading, error, refetch } = useStockQuote(symbol)
  const { subscribe, unsubscribe, setOnPriceUpdate } = useWebSocket()

  useEffect(() => {
    subscribe(symbol)
    return () => unsubscribe(symbol)
  }, [symbol, subscribe, unsubscribe])

  useEffect(() => {
    setOnPriceUpdate(() => {
      refetch()
    })
  }, [setOnPriceUpdate, refetch])

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  if (error || !quote) {
    return (
      <div className="flex h-screen items-center justify-center p-8">
        <ErrorMessage
          message={error?.message ?? `Failed to load data for ${symbol}`}
          onRetry={() => refetch()}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      <PriceHeader quote={quote} />
      <ChartPanel symbol={symbol} />
      <div className="grid grid-cols-1 gap-4 p-6 lg:grid-cols-3">
        <MetricsPanel symbol={symbol} />
        <NewsPanel symbol={symbol} />
        <AIInsightPanel symbol={symbol} />
      </div>
    </div>
  )
}
