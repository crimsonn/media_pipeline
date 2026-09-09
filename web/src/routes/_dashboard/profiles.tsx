import { useCallback, useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { RotateCw } from 'lucide-react'
import { ProfileForm } from '#/components/profile-form'
import { Page } from '#/components/page'
import { Button } from '#/components/ui/button'
import { apiErrorMessage } from '#/lib/api'
import { type Profile, listProfiles } from '#/lib/profiles'

export const Route = createFileRoute('/_dashboard/profiles')({
  component: ProfilesPage,
})

function ProfilesPage() {
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setProfiles(await listProfiles())
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setLoading(false)
      setLoaded(true)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  function handleCreated(profile: Profile) {
    setProfiles((prev) => [profile, ...prev.filter((p) => p.id !== profile.id)])
  }

  return (
    <Page
      title="Profiles"
      description="Bundles of renditions applied when transcoding a source file."
      actions={
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon-sm"
            onClick={() => void load()}
            disabled={loading}
            aria-label="Refresh"
          >
            <RotateCw className={loading ? 'animate-spin' : undefined} />
          </Button>
          <ProfileForm onCreated={handleCreated} />
        </div>
      }
    >
      {error ? (
        <div className="mb-4 flex items-center justify-between gap-4 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">
          <span>{error}</span>
          <Button
            variant="outline"
            size="xs"
            onClick={() => void load()}
            disabled={loading}
          >
            Retry
          </Button>
        </div>
      ) : null}

      {loading && profiles.length === 0 ? (
        <div className="rounded-xl border border-border bg-card px-4 py-12 text-center text-sm text-muted-foreground">
          Loading profiles…
        </div>
      ) : null}

      {loaded && !loading && !error && profiles.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border bg-card px-4 py-12 text-center text-sm text-muted-foreground">
          No profiles yet. Create one to get started.
        </div>
      ) : null}

      {profiles.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2">
          {profiles.map((p) => (
            <div
              key={p.id}
              className="flex flex-col rounded-xl border border-border bg-card p-4"
            >
              <div className="flex items-center gap-2">
                <h2 className="text-sm font-semibold">{p.name}</h2>
                <span className="text-xs text-muted-foreground">#{p.id}</span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                {p.description || '—'}
              </p>

              <div className="mt-4 flex flex-wrap gap-1.5">
                {p.renditions.length === 0 ? (
                  <span className="text-xs text-muted-foreground">
                    No renditions linked
                  </span>
                ) : (
                  p.renditions.map((r) => (
                    <span
                      key={r.id}
                      className="rounded-md border border-border bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground"
                    >
                      {r.name}
                    </span>
                  ))
                )}
              </div>

              <div className="mt-4 border-t border-border pt-3 text-xs text-muted-foreground">
                HLS segment: {p.hls_segment_time}s · {p.renditions.length}{' '}
                renditions
              </div>
            </div>
          ))}
        </div>
      ) : null}
    </Page>
  )
}
