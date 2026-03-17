import { useQuery } from '@tanstack/react-query'
import { getNews } from '../api/stocks'
import type { NewsArticle } from '../types'

export function useStockNews(symbol: string) {
  return useQuery<NewsArticle[]>({
    queryKey: ['news', symbol],
    queryFn: () => getNews(symbol),
    enabled: !!symbol,
    refetchInterval: 60000,
  })
}
