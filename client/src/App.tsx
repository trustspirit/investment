import { Routes, Route } from 'react-router-dom'
import Sidebar from './components/layout/Sidebar'
import WelcomePage from './pages/WelcomePage'
import DashboardPage from './pages/DashboardPage'

export default function App() {
  return (
    <div className="flex min-h-screen" style={{ backgroundColor: 'var(--bg-primary)' }}>
      <Sidebar />
      <main className="ml-[280px] flex-1">
        <Routes>
          <Route path="/" element={<WelcomePage />} />
          <Route path="/stock/:symbol" element={<DashboardPage />} />
        </Routes>
      </main>
    </div>
  )
}
