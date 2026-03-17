import { useMutation } from '@tanstack/react-query'
import { generateStrategy } from '../api/insights'

export function useAIStrategy(symbol: string) {
  const mutation = useMutation({
    mutationFn: () => generateStrategy(symbol),
  })

  return {
    strategy: mutation.data ?? null,
    isGenerating: mutation.isPending,
    generateError: mutation.error,
    generate: mutation.mutate,
    reset: mutation.reset,
  }
}
