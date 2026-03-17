import { useState } from 'react'
import { TrendingUp, Target } from 'lucide-react'
import { RecommendationContent } from './RecommendationPanel'
import { AIStrategyContent } from './AIStrategyPanel'

type Tab = 'recommendation' | 'strategy'

interface InvestmentPanelProps {
  symbol: string
}

const TABS: { key: Tab; label: string; icon: typeof TrendingUp }[] = [
  { key: 'recommendation', label: '투자 제안', icon: TrendingUp },
  { key: 'strategy', label: 'AI 투자 전략', icon: Target },
]

export function InvestmentPanel({ symbol }: InvestmentPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('recommendation')

  return (
    <div
      className="flex flex-col overflow-hidden rounded-xl"
      style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border)' }}
    >
      <div
        className="flex"
        style={{ borderBottom: '1px solid var(--border)' }}
      >
        {TABS.map((tab) => {
          const Icon = tab.icon
          const isActive = activeTab === tab.key
          return (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className="flex flex-1 cursor-pointer items-center justify-center gap-1.5 px-4 py-3 text-sm font-semibold transition-colors"
              style={{
                backgroundColor: 'transparent',
                color: isActive ? 'var(--accent-cyan)' : 'var(--text-muted)',
                border: 'none',
                borderBottom: isActive ? '2px solid var(--accent-cyan)' : '2px solid transparent',
              }}
            >
              <Icon className="h-4 w-4" />
              {tab.label}
            </button>
          )
        })}
      </div>

      <div className="max-h-[600px] overflow-y-auto px-4 py-3">
        <div style={{ display: activeTab === 'recommendation' ? 'block' : 'none' }}>
          <RecommendationContent symbol={symbol} />
        </div>
        <div style={{ display: activeTab === 'strategy' ? 'block' : 'none' }}>
          <AIStrategyContent symbol={symbol} />
        </div>
      </div>
    </div>
  )
}
