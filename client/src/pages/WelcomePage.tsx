import { TrendingUp, Search } from 'lucide-react'
import { useState } from 'react'
import { StockSearchModal } from '../components/search/StockSearchModal'

export default function WelcomePage() {
  const [searchOpen, setSearchOpen] = useState(false)

  return (
    <>
      <div className="flex h-screen flex-col items-center justify-center px-8">
        <div
          className="mb-6 flex h-20 w-20 items-center justify-center rounded-2xl"
          style={{ backgroundColor: 'rgba(34, 211, 238, 0.1)' }}
        >
          <TrendingUp className="h-10 w-10" style={{ color: 'var(--accent-cyan)' }} />
        </div>
        <h1 className="mb-3 text-3xl font-bold" style={{ color: 'var(--text-primary)' }}>
          Welcome to InvestDash
        </h1>
        <p className="mb-8 max-w-md text-center" style={{ color: 'var(--text-secondary)' }}>
          Track your investments, analyze stocks with AI insights, and stay updated with real-time
          market data.
        </p>
        <button
          onClick={() => setSearchOpen(true)}
          className="flex cursor-pointer items-center gap-2 rounded-xl px-6 py-3 text-sm font-semibold transition-colors hover:opacity-90"
          style={{
            backgroundColor: 'var(--accent-cyan)',
            color: 'var(--bg-primary)',
            border: 'none',
          }}
        >
          <Search className="h-4 w-4" />
          Search and add your first stock
        </button>
      </div>
      {searchOpen && <StockSearchModal onClose={() => setSearchOpen(false)} />}
    </>
  )
}
