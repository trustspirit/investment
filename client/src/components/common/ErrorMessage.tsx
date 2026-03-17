import { AlertCircle } from 'lucide-react'

interface ErrorMessageProps {
  message: string
  onRetry?: () => void
}

export function ErrorMessage({ message, onRetry }: ErrorMessageProps) {
  return (
    <div
      className="flex flex-col items-center gap-3 rounded-lg p-6 text-center"
      style={{ backgroundColor: 'var(--bg-tertiary)', border: '1px solid var(--border)' }}
    >
      <AlertCircle className="h-8 w-8" style={{ color: 'var(--negative)' }} />
      <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>
        {message}
      </p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="cursor-pointer rounded-md px-4 py-2 text-sm font-medium transition-colors hover:opacity-80"
          style={{
            backgroundColor: 'var(--accent-cyan)',
            color: 'var(--bg-primary)',
            border: 'none',
          }}
        >
          Retry
        </button>
      )}
    </div>
  )
}
