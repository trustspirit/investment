import { useParams } from 'react-router-dom'
import { StockDashboard } from '../components/dashboard/StockDashboard'

export default function DashboardPage() {
  const { symbol } = useParams<{ symbol: string }>()

  if (!symbol) return null

  return <StockDashboard symbol={symbol} />
}
