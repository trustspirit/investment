export interface StockQuote {
  symbol: string
  name: string
  price: number
  change: number
  changePercent: number
  volume: number
  marketCap: number
  currency: string
  preMarket?: number
  postMarket?: number
}

export interface HistoricalDataPoint {
  timestamp: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface NewsArticle {
  title: string
  link: string
  source: string
  publishedAt: string
  thumbnail: string
  relatedSymbols: string[]
  category: 'company' | 'sector' | 'market' | 'geopolitical'
}

export interface CompanyInfo {
  symbol: string
  name: string
  sector: string
  industry: string
  description: string
  employees: number
  website: string
  currency: string
  pe?: number
  eps?: number
  dividendYield?: number
  '52wHigh'?: number
  '52wLow'?: number
  beta?: number
}

export interface SymbolSearchResult {
  symbol: string
  name: string
  exchange: string
  type: string
}

export interface WatchlistItem {
  symbol: string
  name: string
  addedAt: string
}

export interface AIInsight {
  symbol: string
  summary: string
  sentiment: 'bullish' | 'bearish' | 'neutral'
  keyPoints: string[]
  risks: string[]
  opportunities: string[]
  recommendation: string
  generatedAt: string
  provider: string
}

export interface WSPriceUpdate {
  type: 'priceUpdate'
  symbol: string
  quote: StockQuote
  timestamp: string
}

export type ChartRange = 'pre' | '1d' | '5d' | '1mo' | '6mo' | '1y' | '5y' | 'max'
