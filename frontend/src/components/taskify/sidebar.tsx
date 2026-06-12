"use client"

import React, { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { cn } from "@/lib/utils"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { invalidateTaskCaches } from "@/components/taskify/task-cache"
import type { CurrentView } from "@/components/taskify/navigation"
import type { Board } from "@/services/boardService"
import { deleteBoard } from "@/services/boardService"
import { useAuthStore } from "@/store/useAuthStore"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { NewBoardDialog } from "@/components/taskify/new-board-dialog"
import { Separator } from "@/components/ui/separator"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import {
  LayoutDashboard,
  Plus,
  CheckSquare,
  Calendar,
  PieChart,
  Zap,
  Settings,
  HelpCircle,
  ChevronRight,
  LogOut,
  Trash2,
} from "lucide-react"

const boardColors = ["bg-indigo-500", "bg-violet-500", "bg-amber-500", "bg-emerald-500"]

const navItems: { icon: React.ElementType; label: string; view: CurrentView }[] = [
  { icon: LayoutDashboard, label: "Dashboard", view: "dashboard" },
  { icon: CheckSquare,     label: "Mis Tareas", view: "tasks" },
  { icon: Calendar,        label: "Agenda", view: "agenda" },
  { icon: PieChart,        label: "Control financiero", view: "financial" },
]

interface SidebarProps {
  className?: string
  activeView?: CurrentView
  boards?: Board[]
  boardsError?: string
  boardsLoading?: boolean
  onViewChange?: (view: CurrentView) => void
  selectedBoardId?: string | null
  onBoardSelect?: (board: Board) => void
}

export function Sidebar({
  className,
  activeView = "tasks",
  boards = [],
  boardsError,
  boardsLoading = false,
  onViewChange,
  selectedBoardId,
  onBoardSelect,
}: SidebarProps) {
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)
  const [newBoardOpen, setNewBoardOpen] = useState(false)
  const [boardToDelete, setBoardToDelete] = useState<Board | null>(null)
  const deleteBoardMutation = useMutation({
    mutationFn: deleteBoard,
    onSuccess: (_data, boardId) => {
      queryClient.invalidateQueries({ queryKey: ["boards"] })
      invalidateTaskCaches(queryClient, boardId)
      setBoardToDelete(null)
    },
  })

  function handleDeleteBoard(board: Board) {
    setBoardToDelete(board)
  }

  function handleConfirmDeleteBoard() {
    if (!boardToDelete) {
      return
    }

    deleteBoardMutation.mutate(boardToDelete.id)
  }

  return (
    <TooltipProvider delayDuration={0}>
      <NewBoardDialog open={newBoardOpen} onOpenChange={setNewBoardOpen} />
      <ConfirmDialog
        open={Boolean(boardToDelete)}
        onOpenChange={(open) => {
          if (!open) {
            setBoardToDelete(null)
          }
        }}
        title="Eliminar tablero"
        description={
          boardToDelete
            ? `Se eliminara "${boardToDelete.name}" junto con sus columnas y tareas. Esta accion no se puede deshacer.`
            : ""
        }
        confirmLabel="Eliminar tablero"
        isPending={deleteBoardMutation.isPending}
        onConfirm={handleConfirmDeleteBoard}
      />
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
            const isActive = view === activeView
            return (
              <button
                key={label}
                onClick={() => onViewChange?.(view)}
                className={cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
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
                  onClick={() => setNewBoardOpen(true)}
                >
                  <Plus className="size-3.5" />
                  <span className="sr-only">Crear tablero</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">Crear tablero</TooltipContent>
            </Tooltip>
          </div>

          <div className="flex flex-col gap-0.5">
            {boardsLoading ? (
              <p className="px-3 py-2 text-xs font-medium text-sidebar-foreground/50">
                Cargando tableros...
              </p>
            ) : null}

            {!boardsLoading && boardsError ? (
              <p className="px-3 py-2 text-xs font-medium text-red-300">
                {boardsError}
              </p>
            ) : null}

            {!boardsLoading && !boardsError && boards.length === 0 ? (
              <p className="px-3 py-2 text-xs font-medium text-sidebar-foreground/50">
                Aun no tienes tableros.
              </p>
            ) : null}

            {!boardsLoading && !boardsError ? boards.map((board, index) => (
              <div
                key={board.id}
                className={cn(
                  "group flex items-center gap-1 rounded-md px-1 py-1 text-sm transition-colors",
                  selectedBoardId === board.id
                    ? "bg-sidebar-accent text-sidebar-foreground font-medium"
                    : "text-sidebar-foreground/65 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground"
                )}
              >
                <button
                  type="button"
                  onClick={() => {
                    onViewChange?.("tasks")
                    onBoardSelect?.(board)
                  }}
                  className="flex min-w-0 flex-1 items-center gap-3 rounded-md px-2 py-1.5 text-left"
                >
                  <span className={cn("size-2.5 shrink-0 rounded-full", boardColors[index % boardColors.length])} />
                  <span className="flex-1 truncate">{board.name}</span>
                  <ChevronRight
                    className={cn(
                      "size-3.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-70",
                      selectedBoardId === board.id && "opacity-70"
                    )}
                  />
                </button>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      size="icon"
                      variant="ghost"
                      className={cn(
                        "size-7 shrink-0 text-sidebar-foreground/45 opacity-0 hover:bg-red-500/10 hover:text-red-300 group-hover:opacity-100",
                        selectedBoardId === board.id && "opacity-70",
                      )}
                      disabled={deleteBoardMutation.isPending}
                      onClick={() => handleDeleteBoard(board)}
                    >
                      <Trash2 className="size-3.5" />
                      <span className="sr-only">Eliminar tablero</span>
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="right">Eliminar tablero</TooltipContent>
                </Tooltip>
              </div>
            )) : null}
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
          <div className="flex items-center gap-3 rounded-md px-2 py-2 hover:bg-sidebar-accent/60 transition-colors">
            <Avatar className="size-8">
              <AvatarImage src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${user?.id || "taskify"}`} alt={user?.fullName ?? "Taskify User"} />
              <AvatarFallback className="bg-primary text-primary-foreground text-xs font-semibold">{user?.initials ?? "TU"}</AvatarFallback>
            </Avatar>
            <div className="flex-1 overflow-hidden">
              <p className="truncate text-sm font-medium text-sidebar-foreground">{user?.fullName ?? "Taskify User"}</p>
              <p className="truncate text-xs text-sidebar-foreground/50">{user?.email ?? "Sin correo"}</p>
            </div>
            <Button
              size="icon"
              variant="ghost"
              className="size-8 shrink-0 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
              onClick={logout}
              aria-label="Cerrar sesion"
            >
              <LogOut className="size-4" />
            </Button>
          </div>
        </div>
      </aside>
    </TooltipProvider>
  )
}
