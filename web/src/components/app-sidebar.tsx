import { useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import {
  Film,
  Inbox,
  LayoutDashboard,
  Moon,
  PlayCircle,
  SlidersHorizontal,
  Sun,
} from 'lucide-react'
import { useTheme } from '#/lib/theme'

const navItems = [
  { to: '/', label: 'Home', icon: LayoutDashboard, exact: true },
  { to: '/pending', label: 'Pending files', icon: Inbox, exact: false },
  { to: '/playback', label: 'Playback', icon: PlayCircle, exact: false },
  { to: '/renditions', label: 'Renditions', icon: Film, exact: false },
  { to: '/profiles', label: 'Profiles', icon: SlidersHorizontal, exact: false },
] as const

export function AppSidebar() {
  return (
    <aside className="flex h-svh w-60 shrink-0 flex-col border-r border-sidebar-border bg-sidebar">
      <div className="flex h-14 items-center gap-2.5 border-b border-sidebar-border px-4">
        <div className="flex size-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
          <Film className="size-4" />
        </div>
        <div className="flex flex-col leading-tight">
          <span className="text-sm font-semibold text-sidebar-foreground">
            Transcoder
          </span>
          <span className="text-[11px] text-muted-foreground">Admin Panel</span>
        </div>
      </div>

      <nav className="flex flex-1 flex-col gap-1 p-3">
        {navItems.map(({ to, label, icon: Icon, exact }) => (
          <Link
            key={to}
            to={to}
            activeOptions={{ exact }}
            className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[status=active]:bg-sidebar-accent data-[status=active]:text-sidebar-accent-foreground"
          >
            <Icon className="size-4 shrink-0" />
            {label}
          </Link>
        ))}
      </nav>

      <div className="border-t border-sidebar-border p-3">
        <ThemeToggle />
      </div>
    </aside>
  )
}

function ThemeToggle() {
  const { theme, toggleTheme } = useTheme()
  // Avoid a hydration mismatch: render the same content on the server and on the
  // first client paint, then swap once we know the resolved theme.
  const [mounted, setMounted] = useState(false)
  useEffect(() => setMounted(true), [])
  const isDark = mounted && theme === 'dark'

  return (
    <button
      type="button"
      onClick={toggleTheme}
      className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
    >
      {isDark ? (
        <Sun className="size-4 shrink-0" />
      ) : (
        <Moon className="size-4 shrink-0" />
      )}
      {isDark ? 'Light mode' : 'Dark mode'}
    </button>
  )
}
