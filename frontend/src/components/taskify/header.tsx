"use client"

import { useEffect, useRef, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet"
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Sidebar } from "@/components/taskify/sidebar"
import { NewTaskDialog } from "@/components/taskify/new-task-dialog"
import { useTheme } from "@/components/theme-provider"
import { ProfileAvatar } from "@/components/profile-avatar"
import type { CurrentView } from "@/components/taskify/navigation"
import { updateBoardName, type Board } from "@/services/boardService"
import {
  getNotifications,
  markNotificationAsRead,
} from "@/services/notification_api"
import { notifyAppNotification } from "@/lib/notifications"
import { Plus, Bell, Menu, Moon, Pencil, Sun } from "lucide-react"

interface HeaderProps {
  activeView?: CurrentView
  boards?: Board[]
  boardsError?: string
  boardsLoading?: boolean
  onViewChange?: (view: CurrentView) => void
  selectedBoardId?: string | null
  selectedBoardName?: string
  subtitle?: string
  onBoardSelect?: (board: Board) => void
}

const viewTitle: Record<CurrentView, string> = {
  dashboard: "Panel de Control",
  tasks: "Mis Tareas",
  agenda: "Agenda",
  financial: "Control financiero",
}

const fallbackViewSubtitle: Record<CurrentView, string> = {
  dashboard: "Resumen general de tu espacio de trabajo",
  tasks: "0 tareas",
  agenda: "0 tareas",
  financial: "Seguimiento financiero de tus proyectos",
}

export function Header({
  activeView = "tasks",
  boards = [],
  boardsError,
  boardsLoading = false,
  onViewChange,
  selectedBoardId,
  selectedBoardName,
  subtitle,
  onBoardSelect,
}: HeaderProps) {
  const queryClient = useQueryClient()
  const { resolvedTheme, setTheme } = useTheme()
  const [mobileOpen, setMobileOpen] = useState(false)
  const [newTaskOpen, setNewTaskOpen] = useState(false)
  const [isEditingBoardName, setIsEditingBoardName] = useState(false)
  const [boardNameDraft, setBoardNameDraft] = useState(selectedBoardName ?? "")
  const [boardNameError, setBoardNameError] = useState("")
  const notifiedThisSession = useRef<Set<string>>(new Set())
  const boardNameInputRef = useRef<HTMLInputElement>(null)
  const skipNextBoardNameBlur = useRef(false)
  const canEditBoardName = activeView === "tasks" && Boolean(selectedBoardId && selectedBoardName)
  const notificationsQuery = useQuery({
    queryKey: ["notifications"],
    queryFn: getNotifications,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
  })
  const markReadMutation = useMutation({
    mutationFn: markNotificationAsRead,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["notifications"] })
    },
  })
  const notifications = notificationsQuery.data ?? []
  const unreadNotifications = notifications.filter((notification) => !notification.isRead)
  const updateBoardNameMutation = useMutation({
    mutationFn: ({ boardId, name }: { boardId: string; name: string }) =>
      updateBoardName(boardId, name),
    onMutate: async ({ boardId, name }) => {
      setBoardNameError("")
      await queryClient.cancelQueries({ queryKey: ["boards"] })
      const previousBoards = queryClient.getQueryData<Board[]>(["boards"])

      queryClient.setQueryData<Board[]>(["boards"], (current = []) =>
        current.map((board) =>
          board.id === boardId
            ? { ...board, name, updatedAt: new Date().toISOString() }
            : board,
        ),
      )

      return { previousBoards }
    },
    onError: (_error, _variables, context) => {
      if (context?.previousBoards) {
        queryClient.setQueryData(["boards"], context.previousBoards)
      }
      setBoardNameDraft(selectedBoardName ?? "")
      setBoardNameError("No se pudo renombrar el tablero")
      setIsEditingBoardName(true)
    },
    onSuccess: async (_data, variables) => {
      setBoardNameDraft(variables.name)
      setIsEditingBoardName(false)
      await queryClient.invalidateQueries({ queryKey: ["boards"] })
      await queryClient.invalidateQueries({ queryKey: ["boards", variables.boardId] })
    },
  })

  useEffect(() => {
    if (!isEditingBoardName) {
      setBoardNameDraft(selectedBoardName ?? "")
      setBoardNameError("")
    }
  }, [isEditingBoardName, selectedBoardName])

  useEffect(() => {
    if (!isEditingBoardName) {
      return
    }
    boardNameInputRef.current?.focus()
    boardNameInputRef.current?.select()
  }, [isEditingBoardName])

  function startBoardNameEdit() {
    if (!canEditBoardName) {
      return
    }
    setBoardNameDraft(selectedBoardName ?? "")
    setBoardNameError("")
    setIsEditingBoardName(true)
  }

  function cancelBoardNameEdit() {
    skipNextBoardNameBlur.current = true
    setBoardNameDraft(selectedBoardName ?? "")
    setBoardNameError("")
    setIsEditingBoardName(false)
  }

  function commitBoardNameEdit() {
    if (!selectedBoardId || !selectedBoardName) {
      setIsEditingBoardName(false)
      return
    }

    const nextName = boardNameDraft.trim()
    const currentName = selectedBoardName.trim()

    if (nextName === currentName) {
      setBoardNameDraft(selectedBoardName)
      setBoardNameError("")
      setIsEditingBoardName(false)
      return
    }

    if (nextName.length < 3) {
      setBoardNameError("Usa al menos 3 caracteres")
      setIsEditingBoardName(true)
      return
    }

    updateBoardNameMutation.mutate({ boardId: selectedBoardId, name: nextName })
  }

  useEffect(() => {
    unreadNotifications.forEach((notification) => {
      if (notifiedThisSession.current.has(notification.id)) {
        return
      }
      notifiedThisSession.current.add(notification.id)
      void notifyAppNotification(notification)
    })
  }, [unreadNotifications])

  return (
    <>
      <NewTaskDialog
        open={newTaskOpen}
        onOpenChange={setNewTaskOpen}
        boards={boards}
        selectedBoardId={selectedBoardId ?? undefined}
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
          {isEditingBoardName && canEditBoardName ? (
            <div className="flex min-w-0 flex-col gap-1">
              <Input
                ref={boardNameInputRef}
                value={boardNameDraft}
                disabled={updateBoardNameMutation.isPending}
                className="h-9 max-w-sm rounded-md border-border bg-background px-2 text-xl font-bold tracking-tight"
                aria-label="Nombre del tablero"
                onChange={(event) => {
                  setBoardNameDraft(event.target.value)
                  if (boardNameError) {
                    setBoardNameError("")
                  }
                }}
                onBlur={() => {
                  if (skipNextBoardNameBlur.current) {
                    skipNextBoardNameBlur.current = false
                    return
                  }
                  commitBoardNameEdit()
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault()
                    commitBoardNameEdit()
                  }
                  if (event.key === "Escape") {
                    event.preventDefault()
                    cancelBoardNameEdit()
                  }
                }}
              />
              {boardNameError ? (
                <p className="text-xs font-medium text-red-600 dark:text-red-400" role="alert">
                  {boardNameError}
                </p>
              ) : null}
            </div>
          ) : (
            <div className="flex min-w-0 items-center gap-2">
              {canEditBoardName ? (
                <button
                  type="button"
                  className="min-w-0 text-left"
                  onClick={startBoardNameEdit}
                  aria-label="Editar nombre del tablero"
                >
                  <span className="block truncate text-xl font-bold tracking-tight text-foreground text-balance">
                    {selectedBoardName}
                  </span>
                </button>
              ) : (
                <h1 className="truncate text-xl font-bold tracking-tight text-foreground text-balance">
                  {viewTitle[activeView]}
                </h1>
              )}
              {canEditBoardName ? (
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8 shrink-0 text-muted-foreground hover:text-foreground"
                  onClick={startBoardNameEdit}
                  aria-label="Editar nombre del tablero"
                >
                  <Pencil className="size-3.5" />
                </Button>
              ) : null}
            </div>
          )}
          <p className="hidden text-xs text-muted-foreground md:block">
            {subtitle ?? fallbackViewSubtitle[activeView]}
          </p>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            className="relative size-9 text-muted-foreground"
            aria-label="Cambiar tema"
            onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
          >
            <Sun className="size-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
            <Moon className="absolute size-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          </Button>

          {/* Notifications */}
          <Popover>
            <PopoverTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="relative size-9 text-muted-foreground"
                aria-label="Notificaciones"
              >
                <Bell className="size-4" />
                {unreadNotifications.length > 0 ? (
                  <span className="absolute right-1.5 top-1.5 size-2 rounded-full bg-primary" aria-hidden="true" />
                ) : null}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="w-80 p-3">
              <PopoverHeader>
                <PopoverTitle>Notificaciones</PopoverTitle>
              </PopoverHeader>
              <div className="mt-2 max-h-96 overflow-y-auto">
                {notificationsQuery.isLoading ? (
                  <p className="px-3 py-6 text-center text-sm text-muted-foreground">
                    Cargando notificaciones...
                  </p>
                ) : notifications.length === 0 ? (
                  <p className="px-3 py-6 text-center text-sm text-muted-foreground">
                    Sin notificaciones
                  </p>
                ) : (
                  <div className="flex flex-col gap-1">
                    {notifications.map((notification) => (
                      <button
                        key={notification.id}
                        type="button"
                        className="flex w-full cursor-pointer items-start gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-muted"
                        disabled={markReadMutation.isPending}
                        onClick={() => {
                          if (!notification.isRead) {
                            markReadMutation.mutate(notification.id)
                          }
                        }}
                      >
                        <span
                          className={
                            notification.isRead
                              ? "mt-1.5 size-2 shrink-0 rounded-full bg-transparent"
                              : "mt-1.5 size-2 shrink-0 rounded-full bg-primary"
                          }
                          aria-hidden="true"
                        />
                        <span className="flex min-w-0 flex-1 flex-col gap-1">
                          <span className="text-sm font-medium text-foreground">
                            {notification.title}
                          </span>
                          <span className="text-xs leading-relaxed text-muted-foreground">
                            {notification.message}
                          </span>
                          <span className="text-[11px] text-muted-foreground/80">
                            {formatNotificationDate(notification.createdAt)}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </PopoverContent>
          </Popover>

          {/* Primary CTA */}
          <Button
            size="sm"
            className="h-9 rounded-lg text-sm font-semibold"
            onClick={() => setNewTaskOpen(true)}
          >
            <Plus data-icon="inline-start" className="size-4" />
            <span className="hidden sm:inline">Nueva Tarea</span>
            <span className="sm:hidden">Nueva</span>
          </Button>

          {/* Avatar — mobile only */}
          <div className="md:hidden">
            <ProfileAvatar className="size-8" />
          </div>
        </div>
      </header>
    </>
  )
}

function formatNotificationDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat("es-MX", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date)
}
