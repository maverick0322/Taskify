"use client"

import { type FormEvent, useEffect, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Calendar } from "@/components/ui/calendar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { CalendarIcon } from "lucide-react"
import { cn } from "@/lib/utils"
import { invalidateTaskCaches } from "@/components/taskify/task-cache"
import {
  parseTaskDueDate,
  taskDueDateInputTime,
  taskDueDateToISOString,
} from "@/lib/task-dates"
import { getFriendlyErrorMessage } from "@/services/api"
import type { Board } from "@/services/boardService"
import {
  createTask,
  updateTask,
  type Task,
  type TaskPriority,
  type TaskStatus,
} from "@/services/taskService"

// Month names for formatting the selected date label
const MONTH_NAMES = [
  "Ene","Feb","Mar","Abr","May","Jun",
  "Jul","Ago","Sep","Oct","Nov","Dic",
]

function formatDate(date: Date): string {
  return `${date.getDate()} ${MONTH_NAMES[date.getMonth()]} ${date.getFullYear()}`
}

interface NewTaskDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  boards?: Board[]
  selectedBoardId?: string
  taskToEdit?: Task | null
  initialStatus?: TaskStatus
  initialColumnId?: string
}

const GLOBAL_BOARD_VALUE = "__global__"

const priorityMap: Record<string, TaskPriority> = {
  Alta: "high",
  Media: "medium",
  Baja: "low",
}

const priorityLabelMap: Record<TaskPriority, string> = {
  high: "Alta",
  medium: "Media",
  low: "Baja",
}

export function NewTaskDialog({
  open,
  onOpenChange,
  boards = [],
  selectedBoardId,
  taskToEdit,
  initialStatus,
  initialColumnId,
}: NewTaskDialogProps) {
  const queryClient = useQueryClient()
  const [title,       setTitle]       = useState("")
  const [description, setDescription] = useState("")
  const [priority,    setPriority]    = useState("")
  const [dueDate,     setDueDate]     = useState<Date | undefined>(undefined)
  const [time,        setTime]        = useState("")
  const [calOpen,     setCalOpen]     = useState(false)
  const [errorMessage, setErrorMessage] = useState("")
  const [selectedTaskBoardId, setSelectedTaskBoardId] = useState(GLOBAL_BOARD_VALUE)

  const isEditing = Boolean(taskToEdit)
  const shouldShowBoardSelect = !isEditing && !selectedBoardId
  const selectedBoardForCreate =
    selectedTaskBoardId === GLOBAL_BOARD_VALUE ? undefined : selectedTaskBoardId
  const activeBoardId = taskToEdit?.boardId ?? selectedBoardForCreate ?? selectedBoardId

  const mutation = useMutation({
    mutationFn: async (input: {
      title: string
      description: string
      priority: TaskPriority
      dueDate: string
      boardId?: string
      status?: TaskStatus
      columnId?: string | null
    }) => {
      if (taskToEdit) {
        await updateTask(taskToEdit.id, input)
        return
      }

      await createTask({
        ...input,
        ...(input.boardId ? { boardId: input.boardId } : {}),
      })
    },
    onSuccess: (_data, variables) => {
      invalidateTaskCaches(queryClient, taskToEdit?.boardId ?? variables.boardId)
      reset()
      onOpenChange(false)
    },
    onError: (error) => {
      setErrorMessage(
        getFriendlyErrorMessage(
          error,
          "No pudimos guardar la tarea. Intentalo de nuevo.",
        )
      )
    },
  })

  useEffect(() => {
    if (!open) {
      return
    }

    if (!taskToEdit) {
      setSelectedTaskBoardId(selectedBoardId ?? GLOBAL_BOARD_VALUE)
      reset()
      return
    }

    setTitle(taskToEdit.title)
    setDescription(taskToEdit.description)
    setPriority(priorityLabelMap[taskToEdit.priority])
    const parsedDueDate = taskToEdit.dueDate
      ? parseTaskDueDate(taskToEdit.dueDate)
      : null
    setDueDate(parsedDueDate ?? undefined)
    setTime(parsedDueDate ? taskDueDateInputTime(parsedDueDate) : "")
    setSelectedTaskBoardId(taskToEdit.boardId ?? GLOBAL_BOARD_VALUE)
    setCalOpen(false)
    setErrorMessage("")
  }, [open, taskToEdit])

  function reset() {
    setTitle("")
    setDescription("")
    setPriority("")
    setDueDate(undefined)
    setTime("")
    setSelectedTaskBoardId(selectedBoardId ?? GLOBAL_BOARD_VALUE)
    setCalOpen(false)
    setErrorMessage("")
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const mappedPriority = priorityMap[priority]
    if (!mappedPriority) {
      setErrorMessage("Selecciona una prioridad para la tarea.")
      return
    }

    mutation.mutate({
      title,
      description,
      priority: mappedPriority,
      dueDate: dueDate ? taskDueDateToISOString(dueDate, time) : "",
      ...(initialStatus && !isEditing ? { status: initialStatus } : {}),
      ...(!isEditing && initialColumnId ? { columnId: initialColumnId } : {}),
      ...(isEditing ? { columnId: taskToEdit?.columnId ?? null } : {}),
      ...(activeBoardId ? { boardId: activeBoardId } : {}),
    })
  }

  function handleCancel() {
    if (mutation.isPending) {
      return
    }

    reset()
    onOpenChange(false)
  }

  function handleOpenChange(nextOpen: boolean) {
    if (mutation.isPending) {
      return
    }

    if (!nextOpen) {
      reset()
    }
    onOpenChange(nextOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Editar Tarea" : "Nueva Tarea"}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Actualiza los datos principales de la tarea."
              : "Completa los datos para agregar una nueva tarea al tablero."}
          </DialogDescription>
        </DialogHeader>

        <form className="flex flex-col gap-4 py-2" onSubmit={handleSubmit}>
          {/* Title */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="task-title">Título</Label>
            <Input
              id="task-title"
              placeholder="Ej. Diseñar pantalla de inicio"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={mutation.isPending}
            />
          </div>

          {/* Description */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="task-description">Descripción</Label>
            <Textarea
              id="task-description"
              placeholder="Describe brevemente la tarea..."
              className="min-h-[80px] resize-none"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={mutation.isPending}
            />
          </div>

          {/* Priority */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="task-priority">Prioridad</Label>
            <Select value={priority} onValueChange={setPriority} disabled={mutation.isPending}>
              <SelectTrigger id="task-priority">
                <SelectValue placeholder="Selecciona una prioridad" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Alta">Alta</SelectItem>
                <SelectItem value="Media">Media</SelectItem>
                <SelectItem value="Baja">Baja</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {shouldShowBoardSelect ? (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="task-board">Tablero</Label>
              <Select
                value={selectedTaskBoardId}
                onValueChange={setSelectedTaskBoardId}
                disabled={mutation.isPending}
              >
                <SelectTrigger id="task-board">
                  <SelectValue placeholder="Selecciona un tablero" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={GLOBAL_BOARD_VALUE}>
                    Ninguno (Tarea Global)
                  </SelectItem>
                  {boards.map((board) => (
                    <SelectItem key={board.id} value={board.id}>
                      {board.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ) : null}

          {/* Due date + time — side by side */}
          <div className="grid grid-cols-2 gap-3">
            {/* Date picker */}
            <div className="flex flex-col gap-1.5">
              <Label>Fecha de entrega</Label>
              <Popover open={calOpen} onOpenChange={setCalOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    type="button"
                    disabled={mutation.isPending}
                    className={cn(
                      "w-full justify-start gap-2 text-left font-normal",
                      !dueDate && "text-muted-foreground"
                    )}
                  >
                    <CalendarIcon className="size-4 shrink-0" />
                    <span className="truncate text-sm">
                      {dueDate ? formatDate(dueDate) : "Seleccionar"}
                    </span>
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={dueDate}
                    onSelect={(date) => {
                      setDueDate(date ?? undefined)
                      setCalOpen(false)
                    }}
                  />
                </PopoverContent>
              </Popover>
            </div>

            {/* Time */}
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="task-time">Hora</Label>
              <Input
                id="task-time"
                type="time"
                value={time}
                onChange={(e) => setTime(e.target.value)}
                className="w-full"
                disabled={mutation.isPending}
              />
            </div>
          </div>

          {errorMessage ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-700">
              {errorMessage}
            </p>
          ) : null}

          <DialogFooter className="gap-2 sm:gap-2">
            <Button variant="outline" type="button" onClick={handleCancel} disabled={mutation.isPending}>
              Cancelar
            </Button>
            <Button type="submit" disabled={!title.trim() || mutation.isPending}>
              {mutation.isPending ? "Guardando..." : isEditing ? "Actualizar" : "Guardar"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
