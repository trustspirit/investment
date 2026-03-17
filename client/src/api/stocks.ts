import { fetchAPI } from './client'
import type {
  SymbolSearchResult,
  StockQuote,
  HistoricalDataPoint,
  CompanyInfo,
  NewsArticle,
  ChartRange,
} from '../types'

export function searchStocks(query: string): Promise<SymbolSearchResult[]> {
  return fetchAPI<SymbolSearchResult[]>(`/api/stocks/search?q=${encodeURIComponent(query)}`)
}

export function getQuote(symbol: string): Promise<StockQuote> {
  return fetchAPI<StockQuote>(`/api/stocks/${encodeURIComponent(symbol)}/quote`)
}

export function getChart(symbol: string, range: ChartRange = '1d'): Promise<HistoricalDataPoint[]> {
  return fetchAPI<HistoricalDataPoint[]>(
    `/api/stocks/${encodeURIComponent(symbol)}/chart?range=${range}`,
  )
}

export function getCompanyInfo(symbol: string): Promise<CompanyInfo> {
  return fetchAPI<CompanyInfo>(`/api/stocks/${encodeURIComponent(symbol)}/info`)
}

export function getNews(symbol: string): Promise<NewsArticle[]> {
  return fetchAPI<NewsArticle[]>(`/api/stocks/${encodeURIComponent(symbol)}/news`)
}
