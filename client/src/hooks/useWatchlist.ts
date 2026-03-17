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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] })
    },
  })

  const removeMutation = useMutation({
    mutationFn: (symbol: string) => removeFromWatchlist(symbol),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] })
    },
  })

  return {
    watchlist: query.data ?? [],
    isLoading: query.isLoading,
    error: query.error,
    addToWatchlist: addMutation.mutate,
    removeFromWatchlist: removeMutation.mutate,
    isAdding: addMutation.isPending,
    isRemoving: removeMutation.isPending,
  }
}
