import type { ReactNode } from 'react'

interface PageProps {
  title: string
  description?: string
  actions?: ReactNode
  children?: ReactNode
}

export function Page({ title, description, actions, children }: PageProps) {
  return (
    <>
      <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-border px-6">
        <h1 className="text-sm font-semibold">{title}</h1>
        {actions}
      </header>
      <main className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto max-w-5xl">
          {description ? (
            <p className="mb-6 text-sm text-muted-foreground">{description}</p>
          ) : null}
          {children}
        </div>
      </main>
    </>
  )
}
