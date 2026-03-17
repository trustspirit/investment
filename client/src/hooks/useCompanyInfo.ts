import { useQuery } from '@tanstack/react-query'
import { getCompanyInfo } from '../api/stocks'
import type { CompanyInfo } from '../types'

export function useCompanyInfo(symbol: string) {
  return useQuery<CompanyInfo>({
    queryKey: ['companyInfo', symbol],
    queryFn: () => getCompanyInfo(symbol),
    enabled: !!symbol,
    staleTime: 5 * 60 * 1000,
  })
}
