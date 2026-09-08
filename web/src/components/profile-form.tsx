import { type FormEvent, useEffect, useState } from 'react'
import { Dialog } from 'radix-ui'
import { Plus, X } from 'lucide-react'
import { Button } from '#/components/ui/button'
import { apiErrorMessage } from '#/lib/api'
import {
  type CreateProfileInput,
  type Profile,
  createProfile,
} from '#/lib/profiles'
import { type Rendition, listRenditions } from '#/lib/renditions'

const inputClass =
  'h-8 w-full rounded-lg border border-border bg-background px-2.5 text-sm outline-none transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50'

interface ProfileFormProps {
  onCreated: (profile: Profile) => void
}

export function ProfileForm({ onCreated }: ProfileFormProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [hlsSegmentTime, setHlsSegmentTime] = useState(6)
  const [isDefault, setIsDefault] = useState(true)
  const [selected, setSelected] = useState<number[]>([])

  const [renditions, setRenditions] = useState<Rendition[]>([])
  const [renditionsError, setRenditionsError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    setRenditionsError(null)
    listRenditions()
      .then((data) => {
        if (!cancelled) setRenditions(data)
      })
      .catch((err) => {
        if (!cancelled) setRenditionsError(apiErrorMessage(err))
      })
    return () => {
      cancelled = true
    }
  }, [open])

  function reset() {
    setName('')
    setDescription('')
    setHlsSegmentTime(6)
    setIsDefault(true)
    setSelected([])
    setError(null)
    setSubmitting(false)
  }

  function toggle(id: number) {
    setSelected((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    )
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (selected.length === 0) {
      setError('Select at least one rendition.')
      return
    }
    setSubmitting(true)
    setError(null)
    const payload: CreateProfileInput = {
      name,
      description,
      hls_segment_time: hlsSegmentTime,
      is_default: isDefault,
      renditions: selected,
    }
    try {
      const created = await createProfile(payload)
      onCreated(created)
      setOpen(false)
      reset()
    } catch (err) {
      setError(apiErrorMessage(err))
      setSubmitting(false)
    }
  }

  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (!next) reset()
      }}
    >
      <Dialog.Trigger asChild>
        <Button size="sm">
          <Plus />
          New profile
        </Button>
      </Dialog.Trigger>

      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40 backdrop-blur-[1px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 flex max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 flex-col rounded-xl border border-border bg-card p-5 shadow-lg data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0 data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95">
          <div className="flex items-start justify-between">
            <div>
              <Dialog.Title className="text-sm font-semibold">
                New profile
              </Dialog.Title>
              <Dialog.Description className="mt-1 text-xs text-muted-foreground">
                Creates a profile via POST /transcoder/profiles.
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <Button variant="ghost" size="icon-sm" aria-label="Close">
                <X />
              </Button>
            </Dialog.Close>
          </div>

          <form
            onSubmit={handleSubmit}
            className="mt-4 flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto"
          >
            <label className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-muted-foreground">
                Name
              </span>
              <input
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="web-hls"
                className={inputClass}
              />
            </label>

            <label className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-muted-foreground">
                Description
              </span>
              <input
                required
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Adaptive HLS ladder for browser playback."
                className={inputClass}
              />
            </label>

            <div className="grid grid-cols-2 gap-3">
              <label className="flex flex-col gap-1.5">
                <span className="text-xs font-medium text-muted-foreground">
                  HLS segment time (s)
                </span>
                <input
                  required
                  type="number"
                  min={1}
                  value={hlsSegmentTime}
                  onChange={(e) =>
                    setHlsSegmentTime(e.target.valueAsNumber || 0)
                  }
                  className={inputClass}
                />
              </label>

              <label className="flex items-center gap-2 self-end pb-1.5 text-sm">
                <input
                  type="checkbox"
                  checked={isDefault}
                  onChange={(e) => setIsDefault(e.target.checked)}
                  className="size-4 rounded border-border"
                />
                Default profile
              </label>
            </div>

            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-medium text-muted-foreground">
                Renditions ({selected.length} selected)
              </span>
              <div className="max-h-48 overflow-y-auto rounded-lg border border-border">
                {renditionsError ? (
                  <p className="px-3 py-2 text-xs text-destructive">
                    {renditionsError}
                  </p>
                ) : renditions.length === 0 ? (
                  <p className="px-3 py-2 text-xs text-muted-foreground">
                    No renditions available. Create one first.
                  </p>
                ) : (
                  renditions.map((r) => {
                    const order = selected.indexOf(r.id)
                    return (
                      <label
                        key={r.id}
                        className="flex items-center gap-2.5 border-b border-border px-3 py-2 text-sm last:border-b-0 hover:bg-muted/50"
                      >
                        <input
                          type="checkbox"
                          checked={order !== -1}
                          onChange={() => toggle(r.id)}
                          className="size-4 rounded border-border"
                        />
                        <span className="font-medium">{r.name}</span>
                        <span className="text-xs text-muted-foreground">
                          {r.width}×{r.height} · {r.video_bitrate} kbps
                        </span>
                        {order !== -1 ? (
                          <span className="ml-auto text-xs text-muted-foreground">
                            #{order}
                          </span>
                        ) : null}
                      </label>
                    )
                  })
                )}
              </div>
            </div>

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
              <Button type="submit" size="sm" disabled={submitting}>
                {submitting ? 'Creating…' : 'Create profile'}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
