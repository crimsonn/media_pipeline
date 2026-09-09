import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Film, Gauge, ListVideo, SlidersHorizontal } from 'lucide-react'
import { Page } from '#/components/page'
import { apiErrorMessage } from '#/lib/api'
import { type Job, type JobStatus, jobProgress, listLatestJobs } from '#/lib/jobs'
import { listProfiles } from '#/lib/profiles'

export const Route = createFileRoute('/_dashboard/')({
  component: HomePage,
})

/** How often the recent-activity feed re-fetches while a tab is visible. */
const POLL_MS = 5000

const stats = [
  { label: 'Renditions', value: '6', icon: Film },
  { label: 'Profiles', value: '3', icon: SlidersHorizontal },
  { label: 'Active jobs', value: '2', icon: Gauge },
  { label: 'Completed today', value: '48', icon: ListVideo },
]

function HomePage() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [profileNames, setProfileNames] = useState<Map<number, string>>(
    () => new Map(),
  )
  const [error, setError] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)

  // Profile names are looked up once so each job can show its profile by name
  // instead of a bare id.
  useEffect(() => {
    let cancelled = false
    listProfiles()
      .then((ps) => {
        if (!cancelled) {
          setProfileNames(new Map(ps.map((p) => [p.id, p.name])))
        }
      })
      .catch(() => {
        // Non-critical: fall back to showing the profile id.
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Poll the latest jobs every POLL_MS so an in-progress transcode updates
  // roughly in real time. Skips fetches while the tab is hidden.
  useEffect(() => {
    let cancelled = false

    async function tick() {
      try {
        const data = await listLatestJobs()
        if (cancelled) return
        setJobs(data)
        setError(null)
      } catch (err) {
        if (!cancelled) setError(apiErrorMessage(err))
      } finally {
        if (!cancelled) setLoaded(true)
      }
    }

    void tick()
    const id = setInterval(() => {
      if (document.visibilityState === 'visible') void tick()
    }, POLL_MS)

    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [])

  return (
    <Page title="Home" description="Overview of your transcoding pipeline.">
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
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold">Recent activity</h2>
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className="size-1.5 rounded-full bg-primary" />
            Live · every {POLL_MS / 1000}s
          </span>
        </div>

        {error ? (
          <div className="px-4 py-3 text-sm text-destructive">{error}</div>
        ) : null}

        {loaded && !error && jobs.length === 0 ? (
          <div className="px-4 py-10 text-center text-sm text-muted-foreground">
            No transcoding jobs yet.
          </div>
        ) : null}

        <ul className="divide-y divide-border">
          {jobs.map((job) => (
            <JobRow
              key={job.id}
              job={job}
              profileName={profileNames.get(job.profile_id)}
            />
          ))}
        </ul>
      </div>
    </Page>
  )
}

function JobRow({ job, profileName }: { job: Job; profileName?: string }) {
  const total = job.tasks.length
  const done = job.tasks.filter(
    (t) => t.status === 'completed' || t.status === 'failed',
  ).length
  const percent = Math.round(jobProgress(job) * 100)
  const active = job.status === 'running' || job.status === 'pending'

  return (
    <li className="px-4 py-3">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{job.file_name}</p>
          <p className="text-xs text-muted-foreground">
            Profile: {profileName ?? `#${job.profile_id}`}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-3">
          <StatusBadge status={job.status} />
          <span className="text-xs text-muted-foreground">
            {formatRelative(job.updated_at)}
          </span>
        </div>
      </div>

      {total > 0 ? (
        <div className="mt-2.5">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>
              {done}/{total} renditions
            </span>
            <span className="tabular-nums">{percent}%</span>
          </div>
          <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className={`h-full rounded-full transition-all ${
                job.status === 'failed' ? 'bg-destructive' : 'bg-primary'
              } ${active ? 'animate-pulse' : ''}`}
              style={{ width: `${percent}%` }}
            />
          </div>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {job.tasks.map((task) => (
              <span
                key={task.id}
                title={
                  task.error_message ||
                  `${task.rendition.name} · ${task.status}`
                }
                className={`rounded-md px-2 py-0.5 text-xs font-medium ${taskChipClass(task.status)}`}
              >
                {task.rendition.name}
              </span>
            ))}
          </div>
        </div>
      ) : null}

      {job.error_message ? (
        <p className="mt-2 text-xs text-destructive">{job.error_message}</p>
      ) : null}
    </li>
  )
}

function StatusBadge({ status }: { status: JobStatus }) {
  const styles: Record<JobStatus, string> = {
    pending: 'bg-muted text-muted-foreground',
    running: 'bg-primary/10 text-primary',
    completed: 'bg-primary/10 text-primary',
    failed: 'bg-destructive/10 text-destructive',
  }
  return (
    <span
      className={`rounded-md px-2 py-0.5 text-xs font-medium capitalize ${styles[status]}`}
    >
      {status}
    </span>
  )
}

function taskChipClass(status: JobStatus): string {
  switch (status) {
    case 'completed':
      return 'bg-primary/10 text-primary'
    case 'running':
      return 'bg-primary/10 text-primary animate-pulse'
    case 'failed':
      return 'bg-destructive/10 text-destructive'
    default:
      return 'border border-border text-muted-foreground'
  }
}

/** Compact "just now / 5m ago / 2h ago / 3d ago" formatting. */
function formatRelative(iso: string): string {
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const seconds = Math.max(0, Math.round((Date.now() - then) / 1000))
  if (seconds < 10) return 'just now'
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}
