import { useCompanyInfo } from '../../hooks'
import { LoadingSpinner } from '../common/LoadingSpinner'
import type { CompanyInfo } from '../../types'

interface MetricsPanelProps {
  symbol: string
}

function currencySymbol(currency: string): string {
  if (currency === 'KRW') return '₩'
  if (currency === 'JPY') return '¥'
  if (currency === 'EUR') return '€'
  if (currency === 'GBP') return '£'
  if (currency === 'CNY') return '¥'
  return '$'
}

function formatMetric(value: number | undefined, prefix = '', suffix = ''): string {
  if (value === undefined || value === null) return '—'
  return `${prefix}${value.toLocaleString(undefined, { maximumFractionDigits: 2 })}${suffix}`
}

interface MetricItem {
  label: string
  value: string
}

function getMetrics(info: CompanyInfo): MetricItem[] {
  const sym = currencySymbol(info.currency || 'USD')
  return [
    { label: 'P/E Ratio', value: formatMetric(info.pe) },
    { label: 'EPS', value: formatMetric(info.eps, sym) },
    { label: 'Beta', value: formatMetric(info.beta) },
    { label: 'Div Yield', value: info.dividendYield ? formatMetric(info.dividendYield * 100, '', '%') : '—' },
    { label: '52W High', value: formatMetric(info['52wHigh'], sym) },
    { label: '52W Low', value: formatMetric(info['52wLow'], sym) },
    { label: 'Employees', value: info.employees ? info.employees.toLocaleString() : '—' },
  ]
}

export function MetricsPanel({ symbol }: MetricsPanelProps) {
  const { data: info, isLoading } = useCompanyInfo(symbol)

  if (isLoading) return <LoadingSpinner size="sm" />
  if (!info) return null

  const metrics = getMetrics(info)

  return (
    <div
      className="overflow-hidden rounded-xl"
      style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border)' }}
    >
      <div
        className="px-4 py-3 text-sm font-semibold"
        style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-primary)' }}
      >
        Key Metrics
      </div>
      <div className="grid grid-cols-2 gap-px" style={{ backgroundColor: 'var(--border)' }}>
        {metrics.map((metric) => (
          <div
            key={metric.label}
            className="px-4 py-3"
            style={{ backgroundColor: 'var(--bg-secondary)' }}
          >
            <div className="text-xs" style={{ color: 'var(--text-muted)' }}>
              {metric.label}
            </div>
            <div className="mt-1 text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>
              {metric.value}
            </div>
          </div>
        ))}
      </div>

      {info.description && (
        <div className="px-4 py-3" style={{ borderTop: '1px solid var(--border)' }}>
          <p
            className="line-clamp-3 text-xs leading-relaxed"
            style={{ color: 'var(--text-secondary)' }}
          >
            {info.description}
          </p>
          <div className="mt-2 flex items-center gap-3 text-xs" style={{ color: 'var(--text-muted)' }}>
            {info.sector && <span>{info.sector}</span>}
            {info.industry && (
              <>
                <span>·</span>
                <span>{info.industry}</span>
              </>
            )}
            {info.website && (
              <>
                <span>·</span>
                <a
                  href={info.website}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ color: 'var(--accent-cyan)' }}
                >
                  Website
                </a>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
