import { createFileRoute } from '@tanstack/react-router'
import { Film, Gauge, ListVideo, SlidersHorizontal } from 'lucide-react'
import { Page } from '#/components/page'

export const Route = createFileRoute('/_dashboard/')({
  component: HomePage,
})

const stats = [
  { label: 'Renditions', value: '6', icon: Film },
  { label: 'Profiles', value: '3', icon: SlidersHorizontal },
  { label: 'Active jobs', value: '2', icon: Gauge },
  { label: 'Completed today', value: '48', icon: ListVideo },
]

const activity = [
  { file: 'trailer-4k.mov', profile: 'web-hls', state: 'completed', time: '2m ago' },
  { file: 'lecture-07.mp4', profile: 'archive', state: 'in progress', time: '9m ago' },
  { file: 'promo-vertical.mp4', profile: 'social', state: 'completed', time: '31m ago' },
  { file: 'keynote-day1.mkv', profile: 'web-hls', state: 'error', time: '1h ago' },
]

function HomePage() {
  return (
    <Page
      title="Home"
      description="Overview of your transcoding pipeline."
    >
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map(({ label, value, icon: Icon }) => (
          <div
            key={label}
            className="rounded-xl border border-border bg-card p-4"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-muted-foreground">
                {label}
              </span>
              <Icon className="size-4 text-muted-foreground" />
            </div>
            <p className="mt-3 text-2xl font-semibold tabular-nums">{value}</p>
          </div>
        ))}
      </div>

      <div className="mt-6 rounded-xl border border-border bg-card">
        <div className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold">Recent activity</h2>
        </div>
        <ul className="divide-y divide-border">
          {activity.map((item) => (
            <li
              key={item.file}
              className="flex items-center justify-between gap-4 px-4 py-3"
            >
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{item.file}</p>
                <p className="text-xs text-muted-foreground">
                  Profile: {item.profile}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-3">
                <StateBadge state={item.state} />
                <span className="text-xs text-muted-foreground">{item.time}</span>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </Page>
  )
}

function StateBadge({ state }: { state: string }) {
  const styles: Record<string, string> = {
    completed: 'bg-primary/10 text-primary',
    'in progress': 'bg-muted text-muted-foreground',
    error: 'bg-destructive/10 text-destructive',
  }

  return (
    <span
      className={`rounded-md px-2 py-0.5 text-xs font-medium capitalize ${
        styles[state] ?? 'bg-muted text-muted-foreground'
      }`}
    >
      {state}
    </span>
  )
}
