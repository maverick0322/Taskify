"use client"

import React, { useState } from "react"
import { cn } from "@/lib/utils"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import {
  LayoutDashboard,
  Plus,
  CheckSquare,
  Calendar,
  Zap,
  Settings,
  HelpCircle,
  ChevronRight,
} from "lucide-react"

const boards = [
  { id: 1, name: "Desarrollo Web", color: "bg-indigo-500", active: true },
  { id: 2, name: "Diseño UI/UX", color: "bg-violet-500", active: false },
  { id: 3, name: "Marketing Q3", color: "bg-amber-500", active: false },
  { id: 4, name: "Infraestructura", color: "bg-emerald-500", active: false },
]

type ActiveView = "kanban" | "agenda"

const navItems: { icon: React.ElementType; label: string; view?: ActiveView }[] = [
  { icon: LayoutDashboard, label: "Dashboard" },
  { icon: CheckSquare,     label: "Mis Tareas", view: "kanban" },
  { icon: Calendar,        label: "Agenda",     view: "agenda" },
  { icon: Zap,             label: "Automatizaciones" },
]

interface SidebarProps {
  className?: string
  activeView?: ActiveView
  onViewChange?: (view: ActiveView) => void
}

export function Sidebar({ className, activeView = "kanban", onViewChange }: SidebarProps) {
  const [activeBoard, setActiveBoard] = useState(1)

  return (
    <TooltipProvider delayDuration={0}>
      <aside
        className={cn(
          "flex h-full w-64 flex-col bg-sidebar text-sidebar-foreground",
          className
        )}
      >
        {/* Logo */}
        <div className="flex items-center gap-2.5 px-5 py-5">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary">
            <Zap className="size-4 text-primary-foreground" strokeWidth={2.5} />
          </div>
          <span className="text-lg font-bold tracking-tight text-sidebar-foreground">
            Taskify
          </span>
        </div>

        <Separator className="bg-sidebar-border" />

        {/* Nav Items */}
        <nav className="flex flex-col gap-1 px-3 pt-4">
          {navItems.map(({ icon: Icon, label, view }) => {
            const isActive = view !== undefined && view === activeView
            return (
              <button
                key={label}
                onClick={() => view && onViewChange?.(view)}
                className={cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
                  !view && "cursor-default"
                )}
              >
                <Icon className="size-4 shrink-0" />
                {label}
              </button>
            )
          })}
        </nav>

        {/* Boards Section */}
        <div className="mt-6 flex-1 overflow-y-auto px-3">
          <div className="mb-2 flex items-center justify-between px-3">
            <span className="text-xs font-semibold uppercase tracking-widest text-sidebar-foreground/40">
              Mis Tableros
            </span>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-6 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                >
                  <Plus className="size-3.5" />
                  <span className="sr-only">Crear tablero</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">Crear tablero</TooltipContent>
            </Tooltip>
          </div>

          <div className="flex flex-col gap-0.5">
            {boards.map((board) => (
              <button
                key={board.id}
                onClick={() => setActiveBoard(board.id)}
                className={cn(
                  "group flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors",
                  activeBoard === board.id
                    ? "bg-sidebar-accent text-sidebar-foreground font-medium"
                    : "text-sidebar-foreground/65 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground"
                )}
              >
                <span className={cn("size-2.5 shrink-0 rounded-full", board.color)} />
                <span className="flex-1 truncate text-left">{board.name}</span>
                <ChevronRight
                  className={cn(
                    "size-3.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-70",
                    activeBoard === board.id && "opacity-70"
                  )}
                />
              </button>
            ))}
          </div>
        </div>

        {/* Bottom Actions */}
        <div className="border-t border-sidebar-border px-3 py-3">
          <div className="flex gap-1 justify-center mb-3">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-8 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                >
                  <Settings className="size-4" />
                  <span className="sr-only">Configuración</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">Configuración</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-8 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                >
                  <HelpCircle className="size-4" />
                  <span className="sr-only">Ayuda</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">Ayuda</TooltipContent>
            </Tooltip>
          </div>

          {/* User Profile */}
          <div className="flex items-center gap-3 rounded-md px-2 py-2 hover:bg-sidebar-accent/60 cursor-pointer transition-colors">
            <Avatar className="size-8">
              <AvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=taskify" alt="Ana García" />
              <AvatarFallback className="bg-primary text-primary-foreground text-xs font-semibold">AG</AvatarFallback>
            </Avatar>
            <div className="flex-1 overflow-hidden">
              <p className="truncate text-sm font-medium text-sidebar-foreground">Ana García</p>
              <p className="truncate text-xs text-sidebar-foreground/50">Pro Plan</p>
            </div>
            <div className="size-2 rounded-full bg-emerald-400 shrink-0" />
          </div>
        </div>
      </aside>
    </TooltipProvider>
  )
}
