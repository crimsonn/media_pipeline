import { useCallback, useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { RotateCw } from 'lucide-react'
import { HlsPlayer } from '#/components/hls-player'
import { Page } from '#/components/page'
import { Button } from '#/components/ui/button'
import { apiErrorMessage } from '#/lib/api'
import { type PlaybackOutput, listOutputs, playbackUrl } from '#/lib/playback'

export const Route = createFileRoute('/_dashboard/playback')({
  component: PlaybackPage,
})

function PlaybackPage() {
  const [outputs, setOutputs] = useState<PlaybackOutput[]>([])
  const [selected, setSelected] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await listOutputs()
      setOutputs(data)
      setSelected((cur) => cur ?? data[0]?.name ?? null)
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

  const current = outputs.find((o) => o.name === selected) ?? null

  return (
    <Page
      title="Playback"
      description="Stream finished HLS transcodes from the output folder."
      actions={
        <Button
          variant="outline"
          size="icon-sm"
          onClick={() => void load()}
          disabled={loading}
          aria-label="Refresh"
        >
          <RotateCw className={loading ? 'animate-spin' : undefined} />
        </Button>
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

      {loading && outputs.length === 0 ? (
        <div className="rounded-xl border border-border bg-card px-4 py-12 text-center text-sm text-muted-foreground">
          Loading outputs…
        </div>
      ) : null}

      {loaded && !loading && !error && outputs.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border bg-card px-4 py-12 text-center text-sm text-muted-foreground">
          No finished transcodes yet. Assign a profile to a pending file to get
          started.
        </div>
      ) : null}

      {outputs.length > 0 ? (
        <div className="grid gap-4 lg:grid-cols-[240px_1fr]">
          <ul className="flex flex-col gap-1 lg:max-h-[70vh] lg:overflow-y-auto">
            {outputs.map((o) => (
              <li key={o.name}>
                <button
                  type="button"
                  onClick={() => setSelected(o.name)}
                  className={`w-full rounded-lg border px-3 py-2 text-left text-sm transition-colors ${
                    o.name === selected
                      ? 'border-border bg-muted font-medium'
                      : 'border-transparent hover:bg-muted/50'
                  }`}
                >
                  <span className="block truncate">{o.name}</span>
                  <span className="block text-xs text-muted-foreground">
                    {o.renditions.length} renditions ·{' '}
                    {new Date(o.modified_at).toLocaleDateString()}
                  </span>
                </button>
              </li>
            ))}
          </ul>

          <div className="min-w-0">
            {current ? (
              <>
                <HlsPlayer
                  key={current.name}
                  src={playbackUrl(current.master_path)}
                />
                <div className="mt-3 flex flex-wrap items-center gap-1.5">
                  {current.renditions.map((r) => (
                    <span
                      key={r}
                      className="rounded-md border border-border bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground"
                    >
                      {r}
                    </span>
                  ))}
                </div>
                <p className="mt-2 truncate text-xs text-muted-foreground">
                  {playbackUrl(current.master_path)}
                </p>
              </>
            ) : null}
          </div>
        </div>
      ) : null}
    </Page>
  )
}
