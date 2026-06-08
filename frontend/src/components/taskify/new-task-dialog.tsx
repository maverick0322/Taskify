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
import {
  createTask,
  updateTask,
  type Task,
  type TaskPriority,
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
  selectedBoardId?: string
  taskToEdit?: Task | null
}

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
  selectedBoardId,
  taskToEdit,
}: NewTaskDialogProps) {
  const queryClient = useQueryClient()
  const [title,       setTitle]       = useState("")
  const [description, setDescription] = useState("")
  const [priority,    setPriority]    = useState("")
  const [dueDate,     setDueDate]     = useState<Date | undefined>(undefined)
  const [time,        setTime]        = useState("")
  const [calOpen,     setCalOpen]     = useState(false)
  const [errorMessage, setErrorMessage] = useState("")

  const isEditing = Boolean(taskToEdit)
  const activeBoardId = taskToEdit?.boardId ?? selectedBoardId

  const mutation = useMutation({
    mutationFn: async (input: {
      title: string
      description: string
      priority: TaskPriority
      dueDate: string
    }) => {
      if (taskToEdit) {
        await updateTask(taskToEdit.id, input)
        return
      }

      if (!selectedBoardId) {
        throw new Error("Selecciona un tablero antes de crear una tarea.")
      }

      await createTask({
        ...input,
        boardId: selectedBoardId,
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", activeBoardId] })
      reset()
      onOpenChange(false)
    },
    onError: (error) => {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "No pudimos guardar la tarea. Intentalo de nuevo."
      )
    },
  })

  useEffect(() => {
    if (!open) {
      return
    }

    if (!taskToEdit) {
      reset()
      return
    }

    setTitle(taskToEdit.title)
    setDescription(taskToEdit.description)
    setPriority(priorityLabelMap[taskToEdit.priority])
    setDueDate(taskToEdit.dueDate ? parseAPIDate(taskToEdit.dueDate) : undefined)
    setTime("")
    setCalOpen(false)
    setErrorMessage("")
  }, [open, taskToEdit])

  function reset() {
    setTitle("")
    setDescription("")
    setPriority("")
    setDueDate(undefined)
    setTime("")
    setCalOpen(false)
    setErrorMessage("")
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (!activeBoardId) {
      setErrorMessage("Selecciona un tablero antes de crear una tarea.")
      return
    }

    const mappedPriority = priorityMap[priority]
    if (!mappedPriority) {
      setErrorMessage("Selecciona una prioridad para la tarea.")
      return
    }

    mutation.mutate({
      title,
      description,
      priority: mappedPriority,
      dueDate: dueDate ? formatDateForAPI(dueDate) : "",
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
                      setDueDate(date)
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

          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" type="button" onClick={handleCancel} disabled={mutation.isPending}>
              Cancelar
            </Button>
            <Button type="submit" disabled={!title.trim() || !activeBoardId || mutation.isPending}>
              {mutation.isPending ? "Guardando..." : isEditing ? "Actualizar" : "Guardar"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function formatDateForAPI(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")

  return `${year}-${month}-${day}`
}

function parseAPIDate(rawDate: string): Date | undefined {
  if (!rawDate) {
    return undefined
  }

  const [year, month, day] = rawDate.split("-").map(Number)
  if (!year || !month || !day) {
    return undefined
  }

  return new Date(year, month - 1, day)
}
