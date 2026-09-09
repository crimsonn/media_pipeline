import type { ReactNode } from 'react'
import { Dialog } from 'radix-ui'
import { Button } from '#/components/ui/button'

interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  pending?: boolean
  destructive?: boolean
  onConfirm: () => void
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  pending = false,
  destructive = false,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        if (!pending) onOpenChange(next)
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40 backdrop-blur-[1px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-[calc(100vw-2rem)] max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-xl border border-border bg-card p-5 shadow-lg data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0 data-[state=open]:zoom-in-95 data-[state=closed]:zoom-out-95">
          <Dialog.Title className="text-sm font-semibold">{title}</Dialog.Title>
          {description ? (
            <Dialog.Description className="mt-1.5 text-sm text-muted-foreground">
              {description}
            </Dialog.Description>
          ) : null}

          <div className="mt-5 flex justify-end gap-2">
            <Dialog.Close asChild>
              <Button type="button" variant="outline" size="sm" disabled={pending}>
                {cancelLabel}
              </Button>
            </Dialog.Close>
            <Button
              type="button"
              size="sm"
              variant={destructive ? 'destructive' : 'default'}
              onClick={onConfirm}
              disabled={pending}
            >
              {pending ? 'Working…' : confirmLabel}
            </Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
