import { type FormEvent, useState } from 'react'
import { Dialog } from 'radix-ui'
import { Plus, X } from 'lucide-react'
import { Button } from '#/components/ui/button'
import { apiErrorMessage } from '#/lib/api'
import {
  type CreateRenditionInput,
  type Rendition,
  createRendition,
} from '#/lib/renditions'

const EMPTY: CreateRenditionInput = {
  name: '',
  width: 1920,
  height: 1080,
  video_bitrate: 5000,
  audio_bitrate: 192,
  video_codec: 'h265',
  audio_codec: 'aac',
  fps: 30,
}

const numberFields = [
  { key: 'width', label: 'Width' },
  { key: 'height', label: 'Height' },
  { key: 'video_bitrate', label: 'Video bitrate (kbps)' },
  { key: 'audio_bitrate', label: 'Audio bitrate (kbps)' },
  { key: 'fps', label: 'FPS' },
] as const

interface RenditionFormProps {
  onCreated: (rendition: Rendition) => void
}

export function RenditionForm({ onCreated }: RenditionFormProps) {
  const [open, setOpen] = useState(false)
  const [values, setValues] = useState<CreateRenditionInput>(EMPTY)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function reset() {
    setValues(EMPTY)
    setError(null)
    setSubmitting(false)
  }

  function set<K extends keyof CreateRenditionInput>(
    key: K,
    value: CreateRenditionInput[K],
  ) {
    setValues((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      const created = await createRendition(values)
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
          New rendition
        </Button>
      </Dialog.Trigger>

      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40 backdrop-blur-[1px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-[calc(100vw-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-card p-5 shadow-lg data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0 data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95">
          <div className="flex items-start justify-between">
            <div>
              <Dialog.Title className="text-sm font-semibold">
                New rendition
              </Dialog.Title>
              <Dialog.Description className="mt-1 text-xs text-muted-foreground">
                Creates a rendition via POST /transcoder/renditions.
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <Button variant="ghost" size="icon-sm" aria-label="Close">
                <X />
              </Button>
            </Dialog.Close>
          </div>

          <form onSubmit={handleSubmit} className="mt-4 grid grid-cols-2 gap-3">
            <Field className="col-span-2" label="Name">
              <input
                required
                value={values.name}
                onChange={(e) => set('name', e.target.value)}
                placeholder="1080p"
                className={inputClass}
              />
            </Field>

            {numberFields.map(({ key, label }) => (
              <Field key={key} label={label}>
                <input
                  required
                  type="number"
                  min={1}
                  value={values[key]}
                  onChange={(e) => set(key, e.target.valueAsNumber || 0)}
                  className={inputClass}
                />
              </Field>
            ))}

            <Field label="Video codec">
              <input
                required
                value={values.video_codec}
                onChange={(e) => set('video_codec', e.target.value)}
                placeholder="h265"
                className={inputClass}
              />
            </Field>
            <Field label="Audio codec">
              <input
                required
                value={values.audio_codec}
                onChange={(e) => set('audio_codec', e.target.value)}
                placeholder="aac"
                className={inputClass}
              />
            </Field>

            {error ? (
              <p className="col-span-2 rounded-lg bg-destructive/10 px-3 py-2 text-xs text-destructive">
                {error}
              </p>
            ) : null}

            <div className="col-span-2 mt-1 flex justify-end gap-2">
              <Dialog.Close asChild>
                <Button type="button" variant="outline" size="sm">
                  Cancel
                </Button>
              </Dialog.Close>
              <Button type="submit" size="sm" disabled={submitting}>
                {submitting ? 'Creating…' : 'Create rendition'}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

const inputClass =
  'h-8 w-full rounded-lg border border-border bg-background px-2.5 text-sm outline-none transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50'

function Field({
  label,
  className,
  children,
}: {
  label: string
  className?: string
  children: React.ReactNode
}) {
  return (
    <label className={`flex flex-col gap-1.5 ${className ?? ''}`}>
      <span className="text-xs font-medium text-muted-foreground">{label}</span>
      {children}
    </label>
  )
}
