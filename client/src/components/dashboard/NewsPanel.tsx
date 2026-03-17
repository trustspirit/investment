import { useState, useEffect } from 'react'
import { useStockNews } from '../../hooks'
import { LoadingSpinner } from '../common/LoadingSpinner'
import { Badge } from '../common/Badge'
import { ExternalLink, RefreshCw } from 'lucide-react'
import type { NewsArticle } from '../../types'

interface NewsPanelProps {
  symbol: string
}

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

function categoryBadge(category: NewsArticle['category']): { label: string; variant: 'positive' | 'negative' | 'neutral' | 'info' } {
  switch (category) {
    case 'company':
      return { label: 'Company', variant: 'info' }
    case 'sector':
      return { label: 'Sector', variant: 'positive' }
    case 'market':
      return { label: 'Market', variant: 'neutral' }
    case 'geopolitical':
      return { label: '한국 뉴스', variant: 'negative' }
    default:
      return { label: 'News', variant: 'neutral' }
  }
}

function sentimentBadge(sentiment: NewsArticle['sentiment']): { label: string; variant: 'positive' | 'negative' | 'neutral' } {
  switch (sentiment) {
    case 'positive':
      return { label: '긍정적', variant: 'positive' }
    case 'negative':
      return { label: '부정적', variant: 'negative' }
    default:
      return { label: '중립', variant: 'neutral' }
  }
}

function formatUpdateTime(date: Date): string {
  return date.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

export function NewsPanel({ symbol }: NewsPanelProps) {
  const { data: news, isLoading, dataUpdatedAt, refetch, isFetching } = useStockNews(symbol)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)

  useEffect(() => {
    if (dataUpdatedAt > 0) {
      setLastUpdated(new Date(dataUpdatedAt))
    }
  }, [dataUpdatedAt])

  if (isLoading) return <LoadingSpinner size="sm" />

  return (
    <div
      className="flex flex-col overflow-hidden rounded-xl"
      style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border)' }}
    >
      <div
        className="flex items-center justify-between px-4 py-3"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        <div className="flex items-center gap-2">
          <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
            News
          </span>
          {lastUpdated && (
            <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
              {formatUpdateTime(lastUpdated)}
            </span>
          )}
        </div>
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="cursor-pointer rounded-md p-1.5 transition-colors hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-50"
          style={{
            backgroundColor: 'transparent',
            border: 'none',
            color: 'var(--text-muted)',
          }}
        >
          <RefreshCw
            className="h-3.5 w-3.5"
            style={{ animation: isFetching ? 'spin 1s linear infinite' : 'none' }}
          />
        </button>
      </div>
      <div className="max-h-[400px] overflow-y-auto">
        {!news || news.length === 0 ? (
          <p className="px-4 py-6 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
            No recent news
          </p>
        ) : (
          news.map((article, i) => {
            const sentBadge = sentimentBadge(article.sentiment)
            return (
              <a
                key={`${article.link}-${i}`}
                href={article.link}
                target="_blank"
                rel="noopener noreferrer"
                className="flex gap-3 px-4 py-3 no-underline transition-colors hover:brightness-125"
                style={{ borderBottom: '1px solid var(--border)', color: 'inherit' }}
              >
                {article.thumbnail && (
                  <img
                    src={article.thumbnail}
                    alt=""
                    className="h-14 w-20 shrink-0 rounded-md object-cover"
                  />
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex items-start gap-2">
                    <p
                      className="line-clamp-2 flex-1 text-sm font-medium leading-tight"
                      style={{ color: 'var(--text-primary)' }}
                    >
                      {article.title}
                    </p>
                    <Badge label={sentBadge.label} variant={sentBadge.variant} />
                  </div>
                  <div className="mt-1 flex items-center gap-2 text-xs" style={{ color: 'var(--text-muted)' }}>
                    {article.category && (
                      <Badge
                        label={categoryBadge(article.category).label}
                        variant={categoryBadge(article.category).variant}
                      />
                    )}
                    <span>{article.source}</span>
                    <span>·</span>
                    <span>{timeAgo(article.publishedAt)}</span>
                    <ExternalLink className="ml-auto h-3 w-3" />
                  </div>
                </div>
              </a>
            )
          })
        )}
      </div>
    </div>
  )
}
