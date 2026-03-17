import { useState, useEffect, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { generateStrategy } from '../api/insights'

export function useAIStrategy(symbol: string) {
  const queryClient = useQueryClient()
  const [enabledSymbol, setEnabledSymbol] = useState<string | null>(null)

  useEffect(() => {
    setEnabledSymbol(null)
  }, [symbol])

  const query = useQuery({
    queryKey: ['strategy', symbol],
    queryFn: () => generateStrategy(symbol),
    enabled: enabledSymbol === symbol,
    staleTime: Infinity,
    gcTime: 30 * 60 * 1000,
    retry: false,
  })

  const generate = useCallback(() => {
    queryClient.removeQueries({ queryKey: ['strategy', symbol] })
    setEnabledSymbol(symbol)
  }, [queryClient, symbol])

  const reset = useCallback(() => {
    queryClient.removeQueries({ queryKey: ['strategy', symbol] })
    setEnabledSymbol(null)
  }, [queryClient, symbol])

  return {
    strategy: query.data ?? null,
    isGenerating: query.isFetching,
    generateError: query.error,
    generate,
    reset,
  }
}
