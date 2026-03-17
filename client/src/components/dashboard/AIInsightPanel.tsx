import { useAIInsight } from '../../hooks'
import { LoadingSpinner } from '../common/LoadingSpinner'
import { Badge } from '../common/Badge'
import { Sparkles } from 'lucide-react'

interface AIInsightPanelProps {
  symbol: string
}

function sentimentVariant(sentiment: string): 'positive' | 'negative' | 'neutral' {
  if (sentiment === 'bullish') return 'positive'
  if (sentiment === 'bearish') return 'negative'
  return 'neutral'
}

export function AIInsightPanel({ symbol }: AIInsightPanelProps) {
  const { insight, isLoading, generate, isGenerating } = useAIInsight(symbol)

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
          <Sparkles className="h-4 w-4" style={{ color: 'var(--accent-purple)' }} />
          <span className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
            AI Insight
          </span>
        </div>
        <button
          onClick={() => generate()}
          disabled={isGenerating}
          className="cursor-pointer rounded-md px-3 py-1.5 text-xs font-medium transition-colors hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-50"
          style={{
            backgroundColor: 'rgba(167, 139, 250, 0.15)',
            color: 'var(--accent-purple)',
            border: 'none',
          }}
        >
          {isGenerating ? 'Generating...' : 'Generate'}
        </button>
      </div>

      <div className="max-h-[400px] overflow-y-auto px-4 py-3">
        {(isLoading || isGenerating) && <LoadingSpinner size="sm" />}

        {!isLoading && !isGenerating && !insight && (
          <div className="py-6 text-center">
            <Sparkles className="mx-auto mb-2 h-8 w-8" style={{ color: 'var(--text-muted)' }} />
            <p className="text-sm" style={{ color: 'var(--text-muted)' }}>
              Click Generate to get AI-powered investment insights
            </p>
          </div>
        )}

        {insight && !isGenerating && (
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-2">
              <Badge label={insight.sentiment} variant={sentimentVariant(insight.sentiment)} />
              <span className="text-xs" style={{ color: 'var(--text-muted)' }}>
                via {insight.provider}
              </span>
            </div>

            <p className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>
              {insight.summary}
            </p>

            {insight.keyPoints.length > 0 && (
              <div>
                <h4
                  className="mb-1.5 text-xs font-semibold uppercase tracking-wider"
                  style={{ color: 'var(--text-muted)' }}
                >
                  Key Points
                </h4>
                <ul className="flex flex-col gap-1 pl-0">
                  {insight.keyPoints.map((point, i) => (
                    <li key={i} className="flex gap-2 text-xs" style={{ color: 'var(--text-secondary)' }}>
                      <span style={{ color: 'var(--accent-cyan)' }}>•</span>
                      {point}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              {insight.risks.length > 0 && (
                <div>
                  <h4
                    className="mb-1.5 text-xs font-semibold uppercase tracking-wider"
                    style={{ color: 'var(--negative)' }}
                  >
                    Risks
                  </h4>
                  <ul className="flex flex-col gap-1 pl-0">
                    {insight.risks.map((risk, i) => (
                      <li key={i} className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                        · {risk}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {insight.opportunities.length > 0 && (
                <div>
                  <h4
                    className="mb-1.5 text-xs font-semibold uppercase tracking-wider"
                    style={{ color: 'var(--positive)' }}
                  >
                    Opportunities
                  </h4>
                  <ul className="flex flex-col gap-1 pl-0">
                    {insight.opportunities.map((opp, i) => (
                      <li key={i} className="text-xs" style={{ color: 'var(--text-secondary)' }}>
                        · {opp}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>

            {insight.recommendation && (
              <div
                className="rounded-lg px-3 py-2.5"
                style={{ backgroundColor: 'var(--bg-tertiary)' }}
              >
                <h4
                  className="mb-1 text-xs font-semibold uppercase tracking-wider"
                  style={{ color: 'var(--accent-purple)' }}
                >
                  Recommendation
                </h4>
                <p className="text-xs leading-relaxed" style={{ color: 'var(--text-primary)' }}>
                  {insight.recommendation}
                </p>
              </div>
            )}

            <div className="text-xs" style={{ color: 'var(--text-muted)' }}>
              Generated {new Date(insight.generatedAt).toLocaleString()}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
