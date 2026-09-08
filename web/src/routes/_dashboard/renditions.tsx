import { useCallback, useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { RotateCw, Trash2 } from 'lucide-react'
import { ConfirmDialog } from '#/components/confirm-dialog'
import { RenditionForm } from '#/components/rendition-form'
import { Page } from '#/components/page'
import { Button } from '#/components/ui/button'
import { apiErrorMessage } from '#/lib/api'
import {
  type Rendition,
  deleteRendition,
  listRenditions,
} from '#/lib/renditions'

export const Route = createFileRoute('/_dashboard/renditions')({
  component: RenditionsPage,
})

function RenditionsPage() {
  const [renditions, setRenditions] = useState<Rendition[]>([])
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [target, setTarget] = useState<Rendition | null>(null)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setRenditions(await listRenditions())
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

  function handleCreated(rendition: Rendition) {
    setRenditions((prev) => [
      rendition,
      ...prev.filter((r) => r.id !== rendition.id),
    ])
  }

  async function confirmDelete() {
    if (!target) return
    setDeleting(true)
    setError(null)
    try {
      await deleteRendition(target.id)
      setRenditions((prev) => prev.filter((r) => r.id !== target.id))
      setTarget(null)
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setDeleting(false)
    }
  }

  return (
    <Page
      title="Renditions"
      description="Encoding targets used to build transcoding profiles."
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
          <RenditionForm onCreated={handleCreated} />
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

      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs text-muted-foreground">
                <th className="px-4 py-2.5 font-medium">Name</th>
                <th className="px-4 py-2.5 font-medium">Resolution</th>
                <th className="px-4 py-2.5 font-medium">Video bitrate</th>
                <th className="px-4 py-2.5 font-medium">Audio bitrate</th>
                <th className="px-4 py-2.5 font-medium">FPS</th>
                <th className="px-4 py-2.5 font-medium">Video codec</th>
                <th className="px-4 py-2.5 font-medium">Audio codec</th>
                <th className="w-px px-4 py-2.5" />
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading && renditions.length === 0 ? (
                <tr>
                  <td
                    colSpan={8}
                    className="px-4 py-10 text-center text-sm text-muted-foreground"
                  >
                    Loading renditions…
                  </td>
                </tr>
              ) : null}

              {loaded && !loading && !error && renditions.length === 0 ? (
                <tr>
                  <td
                    colSpan={8}
                    className="px-4 py-10 text-center text-sm text-muted-foreground"
                  >
                    No renditions yet. Create one to get started.
                  </td>
                </tr>
              ) : null}

              {renditions.map((r) => (
                <tr key={r.id} className="hover:bg-muted/50">
                  <td className="px-4 py-3 font-medium">{r.name}</td>
                  <td className="px-4 py-3 tabular-nums text-muted-foreground">
                    {r.width && r.height ? `${r.width}×${r.height}` : '—'}
                  </td>
                  <td className="px-4 py-3 tabular-nums text-muted-foreground">
                    {r.video_bitrate ? `${r.video_bitrate} kbps` : '—'}
                  </td>
                  <td className="px-4 py-3 tabular-nums text-muted-foreground">
                    {r.audio_bitrate} kbps
                  </td>
                  <td className="px-4 py-3 tabular-nums text-muted-foreground">
                    {r.fps}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {r.video_codec}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {r.audio_codec}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      onClick={() => setTarget(r)}
                      aria-label={`Delete ${r.name}`}
                    >
                      <Trash2 />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <ConfirmDialog
        open={target !== null}
        onOpenChange={(open) => {
          if (!open) setTarget(null)
        }}
        title="Delete rendition"
        description={
          target ? (
            <>
              This permanently deletes <strong>{target.name}</strong> (
              {target.width}×{target.height}). This action cannot be undone.
            </>
          ) : null
        }
        confirmLabel="Delete"
        destructive
        pending={deleting}
        onConfirm={() => void confirmDelete()}
      />
    </Page>
  )
}
