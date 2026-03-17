import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getInsight, generateInsight } from '../api/insights'
import type { AIInsight } from '../types'

export function useAIInsight(symbol: string) {
  const queryClient = useQueryClient()

  const query = useQuery<AIInsight | null>({
    queryKey: ['insight', symbol],
    queryFn: () => getInsight(symbol),
    enabled: !!symbol,
    retry: false,
  })

  const generateMutation = useMutation({
    mutationFn: () => generateInsight(symbol),
    onSuccess: (data) => {
      queryClient.setQueryData(['insight', symbol], data)
    },
  })

  return {
    insight: query.data,
    isLoading: query.isLoading,
    error: query.error,
    generate: generateMutation.mutate,
    isGenerating: generateMutation.isPending,
    generateError: generateMutation.error,
  }
}
