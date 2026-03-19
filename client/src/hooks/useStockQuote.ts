import { useQuery } from '@tanstack/react-query'
import { getQuote } from '../api/stocks'
import type { StockQuote } from '../types'

export function useStockQuote(symbol: string) {
  return useQuery<StockQuote>({
    queryKey: ['quote', symbol],
    queryFn: () => getQuote(symbol),
    enabled: !!symbol,
    staleTime: 10000,
    refetchInterval: 30000,
  })
}
