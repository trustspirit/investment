interface BadgeProps {
  label: string
  variant: 'positive' | 'negative' | 'neutral' | 'info'
}

const variantStyles: Record<BadgeProps['variant'], { bg: string; color: string }> = {
  positive: { bg: 'rgba(34, 197, 94, 0.15)', color: 'var(--positive)' },
  negative: { bg: 'rgba(239, 68, 68, 0.15)', color: 'var(--negative)' },
  neutral: { bg: 'rgba(136, 136, 160, 0.15)', color: 'var(--text-secondary)' },
  info: { bg: 'rgba(167, 139, 250, 0.15)', color: 'var(--accent-purple)' },
}

export function Badge({ label, variant }: BadgeProps) {
  const style = variantStyles[variant]

  return (
    <span
      className="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize"
      style={{ backgroundColor: style.bg, color: style.color }}
    >
      {label}
    </span>
  )
}
