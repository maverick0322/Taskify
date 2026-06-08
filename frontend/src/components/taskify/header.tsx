"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet"
import { Sidebar } from "@/components/taskify/sidebar"
import { NewTaskDialog } from "@/components/taskify/new-task-dialog"
import { Search, Plus, Bell, Menu, SlidersHorizontal } from "lucide-react"

interface HeaderProps {
  activeView?: "kanban" | "agenda"
  onViewChange?: (view: "kanban" | "agenda") => void
}

export function Header({ activeView = "kanban", onViewChange }: HeaderProps) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [newTaskOpen, setNewTaskOpen] = useState(false)

  return (
    <>
      <NewTaskDialog open={newTaskOpen} onOpenChange={setNewTaskOpen} />

      {/* Mobile Sidebar Sheet */}
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="w-64 p-0 border-r-0">
          <SheetTitle className="sr-only">Menú de navegación</SheetTitle>
          <Sidebar
            className="h-full"
            activeView={activeView}
            onViewChange={(view) => {
              onViewChange?.(view)
              setMobileOpen(false)
            }}
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
            {activeView === "agenda" ? "Agenda" : "Desarrollo Web"}
          </h1>
          <p className="hidden text-xs text-muted-foreground md:block">
            {activeView === "agenda"
              ? "15 tareas \u00b7 Junio 2026"
              : "12 tareas \u00b7 Actualizado hace 5 min"}
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
          <Button size="sm" className="h-9 gap-1.5 text-sm font-semibold" onClick={() => setNewTaskOpen(true)}>
            <Plus data-icon="inline-start" className="size-4" />
            <span className="hidden sm:inline">Nueva Tarea</span>
            <span className="sm:hidden">Nueva</span>
          </Button>

          {/* Avatar — mobile only */}
          <Avatar className="size-8 md:hidden">
            <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=taskify" alt="Ana García" />
            <AvatarFallback className="bg-primary text-primary-foreground text-xs font-semibold">AG</AvatarFallback>
          </Avatar>
        </div>
      </header>
    </>
  )
}
