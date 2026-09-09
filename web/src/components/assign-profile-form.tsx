import { type FormEvent, useEffect, useState } from 'react'
import { Dialog } from 'radix-ui'
import { X } from 'lucide-react'
import { Button } from '#/components/ui/button'
import { apiErrorMessage } from '#/lib/api'
import { type Profile, listProfiles } from '#/lib/profiles'
import { type PendingFile, enqueuePendingFile } from '#/lib/pending'

const inputClass =
  'h-8 w-full rounded-lg border border-border bg-background px-2.5 text-sm outline-none transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50'

interface AssignProfileFormProps {
  /** The pending file to assign a profile to, or `null` when the dialog is closed. */
  file: PendingFile | null
  onOpenChange: (open: boolean) => void
  /** Called after the file is successfully enqueued. */
  onAssigned: (file: PendingFile) => void
}

export function AssignProfileForm({
  file,
  onOpenChange,
  onAssigned,
}: AssignProfileFormProps) {
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [profilesError, setProfilesError] = useState<string | null>(null)
  const [profileId, setProfileId] = useState<number | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const open = file !== null

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setProfilesError(null)
    setProfileId(null)
    setError(null)
    listProfiles()
      .then((data) => {
        if (cancelled) return
        setProfiles(data)
        if (data.length > 0) setProfileId(data[0].id)
      })
      .catch((err) => {
        if (!cancelled) setProfilesError(apiErrorMessage(err))
      })
    return () => {
      cancelled = true
    }
  }, [open])

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (!file || profileId === null) {
      setError('Select a transcode profile.')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await enqueuePendingFile({
        id: file.id,
        file_name: file.file_name,
        transcode_profile_id: profileId,
      })
      onAssigned(file)
      onOpenChange(false)
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        if (!submitting) onOpenChange(next)
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40 backdrop-blur-[1px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-[calc(100vw-2rem)] max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-card p-5 shadow-lg data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0 data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95">
          <div className="flex items-start justify-between">
            <div>
              <Dialog.Title className="text-sm font-semibold">
                Assign transcode profile
              </Dialog.Title>
              <Dialog.Description className="mt-1 text-xs text-muted-foreground">
                Enqueues{' '}
                <span className="font-medium text-foreground">
                  {file?.file_name}
                </span>{' '}
                via POST /pending/files.
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <Button variant="ghost" size="icon-sm" aria-label="Close">
                <X />
              </Button>
            </Dialog.Close>
          </div>

          <form onSubmit={handleSubmit} className="mt-4 flex flex-col gap-3">
            <label className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-muted-foreground">
                Transcode profile
              </span>
              {profilesError ? (
                <p className="rounded-lg bg-destructive/10 px-3 py-2 text-xs text-destructive">
                  {profilesError}
                </p>
              ) : profiles.length === 0 ? (
                <p className="rounded-lg border border-border px-3 py-2 text-xs text-muted-foreground">
                  No profiles available. Create one first.
                </p>
              ) : (
                <select
                  required
                  value={profileId ?? ''}
                  onChange={(e) => setProfileId(Number(e.target.value))}
                  className={inputClass}
                >
                  {profiles.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} ({p.renditions.length} renditions)
                    </option>
                  ))}
                </select>
              )}
            </label>

            {error ? (
              <p className="rounded-lg bg-destructive/10 px-3 py-2 text-xs text-destructive">
                {error}
              </p>
            ) : null}

            <div className="mt-1 flex justify-end gap-2">
              <Dialog.Close asChild>
                <Button type="button" variant="outline" size="sm">
                  Cancel
                </Button>
              </Dialog.Close>
              <Button
                type="submit"
                size="sm"
                disabled={submitting || profiles.length === 0}
              >
                {submitting ? 'Assigning…' : 'Assign & enqueue'}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
