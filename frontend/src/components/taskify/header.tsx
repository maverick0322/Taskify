"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet"
import { Sidebar } from "@/components/taskify/sidebar"
import { NewTaskDialog } from "@/components/taskify/new-task-dialog"
import type { CurrentView } from "@/components/taskify/navigation"
import type { Board } from "@/services/boardService"
import { useAuthStore } from "@/store/useAuthStore"
import { Search, Plus, Bell, Menu, SlidersHorizontal } from "lucide-react"

interface HeaderProps {
  activeView?: CurrentView
  boards?: Board[]
  boardsError?: string
  boardsLoading?: boolean
  onViewChange?: (view: CurrentView) => void
  selectedBoardId?: string
  selectedBoardName?: string
  onBoardSelect?: (board: Board) => void
}

const viewTitle: Record<CurrentView, string> = {
  dashboard: "Panel de Control",
  tasks: "Mis Tareas",
  agenda: "Agenda",
  automations: "Automatizaciones",
}

const viewSubtitle: Record<CurrentView, string> = {
  dashboard: "Resumen general de tu espacio de trabajo",
  tasks: "12 tareas · Actualizado hace 5 min",
  agenda: "15 tareas · Junio 2026",
  automations: "Flujos inteligentes para tu equipo",
}

export function Header({
  activeView = "tasks",
  boards = [],
  boardsError,
  boardsLoading = false,
  onViewChange,
  selectedBoardId,
  selectedBoardName,
  onBoardSelect,
}: HeaderProps) {
  const user = useAuthStore((state) => state.user)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [newTaskOpen, setNewTaskOpen] = useState(false)

  return (
    <>
      <NewTaskDialog
        open={newTaskOpen}
        onOpenChange={setNewTaskOpen}
        selectedBoardId={selectedBoardId}
      />

      {/* Mobile Sidebar Sheet */}
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="w-64 p-0 border-r-0">
          <SheetTitle className="sr-only">Menú de navegación</SheetTitle>
          <Sidebar
            className="h-full"
            activeView={activeView}
            boards={boards}
            boardsError={boardsError}
            boardsLoading={boardsLoading}
            onViewChange={(view) => {
              onViewChange?.(view)
              setMobileOpen(false)
            }}
            onBoardSelect={(board) => {
              onBoardSelect?.(board)
              setMobileOpen(false)
            }}
            selectedBoardId={selectedBoardId}
          />
        </SheetContent>
      </Sheet>

      <header className="flex h-16 shrink-0 items-center gap-4 border-b border-border bg-card px-4 md:px-6">
        {/* Mobile hamburger */}
        <Button
          variant="ghost"
          size="icon"
          className="size-9 md:hidden text-muted-foreground"
          onClick={() => setMobileOpen(true)}
          aria-label="Abrir menú de navegación"
        >
          <Menu className="size-5" />
        </Button>

        {/* Board Title */}
        <div className="flex-1 min-w-0">
          <h1 className="text-xl font-bold tracking-tight text-foreground truncate text-balance">
            {activeView === "tasks" && selectedBoardName
              ? selectedBoardName
              : viewTitle[activeView]}
          </h1>
          <p className="hidden text-xs text-muted-foreground md:block">
            {viewSubtitle[activeView]}
          </p>
        </div>

        {/* Search bar — visible sm+ */}
        <div className="relative hidden sm:flex w-56 lg:w-72">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
          <Input
            placeholder="Buscar tareas..."
            className="h-9 pl-9 bg-muted/60 border-transparent focus-visible:border-border focus-visible:bg-card text-sm"
          />
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {/* Filter — md+ */}
          <Button
            variant="outline"
            size="sm"
            className="hidden md:flex h-9 gap-2 text-muted-foreground border-border/60"
          >
            <SlidersHorizontal className="size-3.5" />
            <span className="text-sm">Filtrar</span>
          </Button>

          {/* Mobile search icon only */}
          <Button
            variant="ghost"
            size="icon"
            className="size-9 sm:hidden text-muted-foreground"
            aria-label="Buscar tarea"
          >
            <Search className="size-4" />
          </Button>

          {/* Notifications */}
          <Button
            variant="ghost"
            size="icon"
            className="relative size-9 text-muted-foreground"
            aria-label="Notificaciones"
          >
            <Bell className="size-4" />
            <span className="absolute right-1.5 top-1.5 size-2 rounded-full bg-primary" aria-hidden="true" />
          </Button>

          {/* Primary CTA */}
          <Button
            size="sm"
            className="h-9 gap-1.5 text-sm font-semibold"
            onClick={() => setNewTaskOpen(true)}
            disabled={!selectedBoardId}
          >
            <Plus data-icon="inline-start" className="size-4" />
            <span className="hidden sm:inline">Nueva Tarea</span>
            <span className="sm:hidden">Nueva</span>
          </Button>

          {/* Avatar — mobile only */}
          <Avatar className="size-8 md:hidden">
            <AvatarImage src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${user?.id || "taskify"}`} alt={user?.fullName ?? "Taskify User"} />
            <AvatarFallback className="bg-primary text-primary-foreground text-xs font-semibold">{user?.initials ?? "TU"}</AvatarFallback>
          </Avatar>
        </div>
      </header>
    </>
  )
}
