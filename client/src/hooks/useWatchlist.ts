import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getWatchlist, addToWatchlist, removeFromWatchlist } from '../api/watchlist'
import type { WatchlistItem } from '../types'

export function useWatchlist() {
  const queryClient = useQueryClient()

  const query = useQuery<WatchlistItem[]>({
    queryKey: ['watchlist'],
    queryFn: getWatchlist,
  })

  const addMutation = useMutation({
    mutationFn: ({ symbol, name }: { symbol: string; name: string }) =>
      addToWatchlist(symbol, name),
    onMutate: async ({ symbol, name }) => {
      await queryClient.cancelQueries({ queryKey: ['watchlist'] })
      const previous = queryClient.getQueryData<WatchlistItem[]>(['watchlist'])
      queryClient.setQueryData<WatchlistItem[]>(['watchlist'], (old) => [
        ...(old ?? []),
        { symbol, name, addedAt: new Date().toISOString() },
      ])
      return { previous }
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['watchlist'], context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] })
    },
  })

  const removeMutation = useMutation({
    mutationFn: (symbol: string) => removeFromWatchlist(symbol),
    onMutate: async (symbol) => {
      await queryClient.cancelQueries({ queryKey: ['watchlist'] })
      const previous = queryClient.getQueryData<WatchlistItem[]>(['watchlist'])
      queryClient.setQueryData<WatchlistItem[]>(['watchlist'], (old) =>
        (old ?? []).filter((item) => item.symbol !== symbol),
      )
      return { previous }
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['watchlist'], context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] })
    },
  })

  const watchlist = query.data ?? []

  return {
    watchlist,
    isLoading: query.isLoading,
    error: query.error,
    addToWatchlist: addMutation.mutate,
    removeFromWatchlist: removeMutation.mutate,
    isAdding: addMutation.isPending,
    isRemoving: removeMutation.isPending,
    isInWatchlist: (symbol: string) => watchlist.some((item) => item.symbol === symbol),
  }
}
