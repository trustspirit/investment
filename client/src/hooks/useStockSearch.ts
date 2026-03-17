import { useQuery } from '@tanstack/react-query'
import { searchStocks } from '../api/stocks'
import { useState, useEffect } from 'react'
import type { SymbolSearchResult } from '../types'

export function useStockSearch(query: string) {
  const [debouncedQuery, setDebouncedQuery] = useState(query)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 300)
    return () => clearTimeout(timer)
  }, [query])

  return useQuery<SymbolSearchResult[]>({
    queryKey: ['search', debouncedQuery],
    queryFn: () => searchStocks(debouncedQuery),
    enabled: debouncedQuery.length >= 2,
  })
}
