import { useQuery } from '@tanstack/react-query'
import { getChart } from '../api/stocks'
import type { HistoricalDataPoint, ChartRange } from '../types'

export function useStockChart(symbol: string, range: ChartRange = '1d') {
  return useQuery<HistoricalDataPoint[]>({
    queryKey: ['chart', symbol, range],
    queryFn: () => getChart(symbol, range),
    enabled: !!symbol,
  })
}
