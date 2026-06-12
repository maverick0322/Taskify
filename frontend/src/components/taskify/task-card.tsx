"use client"

import { Draggable } from "@hello-pangea/dnd"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState, type CSSProperties, type MouseEvent } from "react"

import { cn } from "@/lib/utils"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { invalidateTaskCaches } from "@/components/taskify/task-cache"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Clock, Paperclip, MessageSquare, Pencil, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  deleteTask,
  updateTaskStatus,
  type Task,
  type TaskStatus,
} from "@/services/taskService"

type Priority = "Alta" | "Media" | "Baja"

interface TaskCardProps {
  id: string
  index: number
  task: Task
  selectedBoardId?: string
  onEditTask: (task: Task) => void
  title: string
  description?: string
  priority: Priority
  dueDate: string
  tag?: string
  assignees?: { name: string; seed: string }[]
  comments?: number
  attachments?: number
}

const priorityConfig: Record<Priority, { label: string; className: string }> = {
  Alta: {
    label: "Alta",
    className:
      "bg-red-100 text-red-700 border-red-200 dark:bg-red-950/50 dark:text-red-400 dark:border-red-900",
  },
  Media: {
    label: "Media",
    className:
      "bg-amber-100 text-amber-700 border-amber-200 dark:bg-amber-950/50 dark:text-amber-400 dark:border-amber-900",
  },
  Baja: {
    label: "Baja",
    className:
      "bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-950/50 dark:text-blue-400 dark:border-blue-900",
  },
}

const statusLabels: Record<TaskStatus, string> = {
  todo: "Por hacer",
  in_progress: "En progreso",
  done: "Terminado",
}

const taskStatusOptions: TaskStatus[] = ["todo", "in_progress", "done"]

export function TaskCard({
  id,
  index,
  task,
  selectedBoardId,
  onEditTask,
  title,
  description,
  priority,
  dueDate = "Sin fecha",
  tag,
  assignees = [],
  comments = 0,
  attachments = 0,
}: TaskCardProps) {
  const { label, className } = priorityConfig[priority]
  const queryClient = useQueryClient()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const boardTasksQueryKey = ["tasks", selectedBoardId]
  const globalTasksQueryKey = ["tasks", "global"]
  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      invalidateTaskCaches(queryClient, selectedBoardId)
      setDeleteDialogOpen(false)
    },
  })
  const statusMutation = useMutation<
    void,
    Error,
    { taskId: string; status: TaskStatus },
    { previousBoardTasks?: Task[]; previousGlobalTasks?: Task[] }
  >({
    mutationFn: updateTaskStatus,
    onMutate: async ({ taskId, status }) => {
      const queryKeys = selectedBoardId
        ? [boardTasksQueryKey, globalTasksQueryKey]
        : [globalTasksQueryKey]

      await Promise.all(
        queryKeys.map((queryKey) => queryClient.cancelQueries({ queryKey })),
      )

      const previousBoardTasks = selectedBoardId
        ? queryClient.getQueryData<Task[]>(boardTasksQueryKey)
        : undefined
      const previousGlobalTasks =
        queryClient.getQueryData<Task[]>(globalTasksQueryKey)

      if (selectedBoardId) {
        queryClient.setQueryData<Task[]>(boardTasksQueryKey, (currentTasks = []) =>
          currentTasks.map((currentTask) =>
            currentTask.id === taskId
              ? { ...currentTask, status }
              : currentTask,
          ),
        )
      }

      queryClient.setQueryData<Task[]>(globalTasksQueryKey, (currentTasks = []) =>
        currentTasks.map((currentTask) =>
          currentTask.id === taskId ? { ...currentTask, status } : currentTask,
        ),
      )

      return { previousBoardTasks, previousGlobalTasks }
    },
    onError: (_error, _variables, context) => {
      if (selectedBoardId && context?.previousBoardTasks) {
        queryClient.setQueryData(boardTasksQueryKey, context.previousBoardTasks)
      }

      if (context?.previousGlobalTasks) {
        queryClient.setQueryData(globalTasksQueryKey, context.previousGlobalTasks)
      }
    },
    onSettled: () => {
      invalidateTaskCaches(queryClient, selectedBoardId)
    },
  })

  function handleEdit(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation()
    onEditTask(task)
  }

  function handleDeleteClick(event: MouseEvent<HTMLButtonElement>) {
    event.stopPropagation()
    setDeleteDialogOpen(true)
  }

  function handleConfirmDelete() {
    deleteMutation.mutate(id)
  }

  function handleStatusChange(status: TaskStatus) {
    if (status === task.status) {
      return
    }

    statusMutation.mutate({ taskId: id, status })
  }

  return (
    <>
      <Draggable draggableId={id} index={index}>
        {(provided) => {
          const draggableStyle = provided.draggableProps.style as CSSProperties

          return (
        <article
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          style={draggableStyle}
          className="group relative rounded-xl border border-border bg-card p-4 shadow-sm transition-all hover:shadow-md hover:-translate-y-0.5 hover:border-primary/30 cursor-grab active:cursor-grabbing"
        >
          {/* Header row */}
          <div className="mb-3 flex items-start justify-between gap-2">
            <div className="flex flex-wrap gap-1.5">
              <Badge
                variant="outline"
                className={cn("text-[11px] font-semibold px-2 py-0.5 rounded-full border", className)}
              >
                {label}
              </Badge>
              {tag && (
                <Badge
                  variant="secondary"
                  className="text-[11px] px-2 py-0.5 rounded-full"
                >
                  {tag}
                </Badge>
              )}
            </div>
            <div
              className="-mr-1 -mt-0.5 flex shrink-0 gap-1 opacity-100 transition-opacity md:opacity-0 md:group-hover:opacity-100"
              onPointerDown={(event) => event.stopPropagation()}
            >
              <Select
                value={task.status}
                onValueChange={(value) => handleStatusChange(value as TaskStatus)}
                disabled={statusMutation.isPending}
              >
                <SelectTrigger
                  size="sm"
                  className="h-6 w-[6.75rem] rounded-md px-2 text-[11px] text-muted-foreground"
                  aria-label="Cambiar estado de la tarea"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="end">
                  {taskStatusOptions.map((status) => (
                    <SelectItem key={status} value={status}>
                      {statusLabels[status]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                variant="ghost"
                size="icon"
                className="size-6 text-muted-foreground hover:text-foreground"
                aria-label="Editar tarea"
                onClick={handleEdit}
              >
                <Pencil className="size-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="size-6 text-muted-foreground hover:text-red-600"
                aria-label="Eliminar tarea"
                disabled={deleteMutation.isPending}
                onClick={handleDeleteClick}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          </div>

          {/* Title */}
          <h3 className="mb-1.5 text-sm font-semibold leading-snug text-card-foreground">
            {title}
          </h3>

          {/* Description */}
          {description && (
            <p className="mb-3 text-xs leading-relaxed text-muted-foreground line-clamp-2">
              {description}
            </p>
          )}

          {/* Footer */}
          <div className="flex items-center justify-between pt-3 border-t border-border/60">
            {/* Date */}
            <div className="flex items-center gap-1 text-muted-foreground">
              <Clock className="size-3" />
              <span className="text-[11px] font-medium">{dueDate}</span>
            </div>

            <div className="flex items-center gap-2">
              {/* Comments / Attachments */}
              {comments > 0 && (
                <div className="flex items-center gap-1 text-muted-foreground">
                  <MessageSquare className="size-3" />
                  <span className="text-[11px]">{comments}</span>
                </div>
              )}
              {attachments > 0 && (
                <div className="flex items-center gap-1 text-muted-foreground">
                  <Paperclip className="size-3" />
                  <span className="text-[11px]">{attachments}</span>
                </div>
              )}

              {/* Assignee Avatars */}
              {assignees.length > 0 && (
                <div className="flex -space-x-1.5">
                  {assignees.slice(0, 3).map((a) => (
                    <Avatar key={a.seed} className="size-5 ring-1 ring-card">
                      <AvatarImage
                        src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${a.seed}`}
                        alt={a.name}
                      />
                      <AvatarFallback className="bg-primary/10 text-primary text-[9px] font-bold">
                        {a.name.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                  ))}
                </div>
              )}
            </div>
          </div>
        </article>
          )
        }}
      </Draggable>
      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Eliminar tarea"
        description={`Se eliminara "${title}". Esta accion no se puede deshacer.`}
        confirmLabel="Eliminar tarea"
        isPending={deleteMutation.isPending}
        onConfirm={handleConfirmDelete}
      />
    </>
  )
}
