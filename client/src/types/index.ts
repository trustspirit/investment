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
  sentiment: 'positive' | 'negative' | 'neutral'
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

export interface RecommendationTrend {
  period: string
  strongBuy: number
  buy: number
  hold: number
  sell: number
  strongSell: number
}

export interface RecommendationData {
  symbol: string
  recommendationKey: string
  recommendationMean: number
  numberOfAnalysts: number
  targetMeanPrice?: number
  targetHighPrice?: number
  targetLowPrice?: number
  currentPrice: number
  currency: string
  trend: RecommendationTrend[]
}

export interface PriceRange {
  low: number
  high: number
  reason: string
}

export interface TimingAnalysis {
  recommendation: string
  timeframe: string
  conditions: string[]
}

export interface AITradeStrategy {
  symbol: string
  analysisTime: string
  currentPrice: number
  currency: string
  signal: string
  confidence: number
  entryPrice: PriceRange
  stopLoss: PriceRange
  takeProfit: PriceRange
  buyTiming: TimingAnalysis
  sellTiming: TimingAnalysis
  riskReward: string
  analysisBasis: string[]
  marketCondition: string
  shortTermView: string
  midTermView: string
  disclaimer: string
  provider: string
}

export interface MarketIndicator {
  symbol: string
  name: string
  price: number
  change: number
  changePercent: number
  currency: string
}
