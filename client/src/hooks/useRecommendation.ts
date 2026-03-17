import { useQuery } from '@tanstack/react-query'
import { getRecommendation } from '../api/stocks'
import type { RecommendationData } from '../types'

export function useRecommendation(symbol: string) {
  return useQuery<RecommendationData>({
    queryKey: ['recommendation', symbol],
    queryFn: () => getRecommendation(symbol),
    enabled: !!symbol,
    staleTime: 5 * 60 * 1000,
  })
}
