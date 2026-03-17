import { useStockNews } from '../../hooks'
import { LoadingSpinner } from '../common/LoadingSpinner'
import { Badge } from '../common/Badge'
import { ExternalLink } from 'lucide-react'
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

export function NewsPanel({ symbol }: NewsPanelProps) {
  const { data: news, isLoading } = useStockNews(symbol)

  if (isLoading) return <LoadingSpinner size="sm" />

  return (
    <div
      className="flex flex-col overflow-hidden rounded-xl"
      style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border)' }}
    >
      <div
        className="px-4 py-3 text-sm font-semibold"
        style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-primary)' }}
      >
        News
      </div>
      <div className="max-h-[400px] overflow-y-auto">
        {!news || news.length === 0 ? (
          <p className="px-4 py-6 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
            No recent news
          </p>
        ) : (
          news.map((article, i) => (
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
                <p
                  className="line-clamp-2 text-sm font-medium leading-tight"
                  style={{ color: 'var(--text-primary)' }}
                >
                  {article.title}
                </p>
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
          ))
        )}
      </div>
    </div>
  )
}
