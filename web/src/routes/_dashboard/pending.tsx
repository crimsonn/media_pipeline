import { useCallback, useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { RotateCw } from 'lucide-react'
import { AssignProfileForm } from '#/components/assign-profile-form'
import { Page } from '#/components/page'
import { Button } from '#/components/ui/button'
import { apiErrorMessage } from '#/lib/api'
import { type PendingFile, listPendingFiles } from '#/lib/pending'

export const Route = createFileRoute('/_dashboard/pending')({
  component: PendingPage,
})

function PendingPage() {
  const [files, setFiles] = useState<PendingFile[]>([])
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [target, setTarget] = useState<PendingFile | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setFiles(await listPendingFiles())
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

  function handleAssigned(file: PendingFile) {
    // Once enqueued the file is no longer pending assignment.
    setFiles((prev) => prev.filter((f) => f.id !== file.id))
    setTarget(null)
  }

  return (
    <Page
      title="Pending files"
      description="Files dropped into the watch folder that need a transcode profile before they can be encoded."
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

      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs text-muted-foreground">
                <th className="px-4 py-2.5 font-medium">File</th>
                <th className="px-4 py-2.5 font-medium">Status</th>
                <th className="px-4 py-2.5 font-medium">Detected</th>
                <th className="w-px px-4 py-2.5" />
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading && files.length === 0 ? (
                <tr>
                  <td
                    colSpan={4}
                    className="px-4 py-10 text-center text-sm text-muted-foreground"
                  >
                    Loading pending files…
                  </td>
                </tr>
              ) : null}

              {loaded && !loading && !error && files.length === 0 ? (
                <tr>
                  <td
                    colSpan={4}
                    className="px-4 py-10 text-center text-sm text-muted-foreground"
                  >
                    No files waiting for assignment.
                  </td>
                </tr>
              ) : null}

              {files.map((f) => (
                <tr key={f.id} className="hover:bg-muted/50">
                  <td className="px-4 py-3 font-medium">{f.file_name}</td>
                  <td className="px-4 py-3">
                    <span className="rounded-md bg-muted px-2 py-0.5 text-xs font-medium capitalize text-muted-foreground">
                      {f.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {new Date(f.created_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button size="sm" onClick={() => setTarget(f)}>
                      Assign profile
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <AssignProfileForm
        file={target}
        onOpenChange={(open) => {
          if (!open) setTarget(null)
        }}
        onAssigned={handleAssigned}
      />
    </Page>
  )
}
