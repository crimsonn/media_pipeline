import { Outlet, createFileRoute } from '@tanstack/react-router'
import { AppSidebar } from '#/components/app-sidebar'

export const Route = createFileRoute('/_dashboard')({
  component: DashboardLayout,
})

function DashboardLayout() {
  return (
    <div className="flex h-svh bg-background text-foreground">
      <AppSidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Outlet />
      </div>
    </div>
  )
}
