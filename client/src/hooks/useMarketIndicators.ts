import { useQuery } from '@tanstack/react-query'
import { getMarketIndicators } from '../api/stocks'
import type { MarketIndicator } from '../types'

export function useMarketIndicators() {
  return useQuery<MarketIndicator[]>({
    queryKey: ['marketIndicators'],
    queryFn: getMarketIndicators,
    staleTime: 60 * 1000,
    refetchInterval: 120_000,
  })
}
