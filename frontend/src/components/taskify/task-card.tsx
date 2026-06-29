"use client"

import { Draggable } from "@hello-pangea/dnd"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState, type CSSProperties, type KeyboardEvent } from "react"

import { cn } from "@/lib/utils"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { invalidateTaskCaches } from "@/components/taskify/task-cache"
import { TaskDetailsModal } from "@/components/taskify/task-details-modal"
import { Badge } from "@/components/ui/badge"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Clock, Paperclip, MessageSquare } from "lucide-react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  deleteTask,
  type Task,
} from "@/services/taskService"
import type { DisplayColumn } from "@/components/taskify/kanban-column-helpers"

type Priority = "Alta" | "Media" | "Baja"

interface TaskCardProps {
  id: string
  index: number
  task: Task
  selectedBoardId?: string
  columnOptions: DisplayColumn[]
  currentColumnId: string
  movePending?: boolean
  onMoveTask: (taskId: string, columnId: string) => void
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

export function TaskCard({
  id,
  index,
  task,
  selectedBoardId,
  columnOptions,
  currentColumnId,
  movePending = false,
  onMoveTask,
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
  const [detailsOpen, setDetailsOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: () => {
      invalidateTaskCaches(queryClient, selectedBoardId)
      setDeleteDialogOpen(false)
    },
  })

  function handleOpenDetails() {
    setDetailsOpen(true)
  }

  function handleCardKeyDown(event: KeyboardEvent<HTMLElement>) {
    if (event.key !== "Enter" && event.key !== " ") {
      return
    }

    event.preventDefault()
    setDetailsOpen(true)
  }

  function handleEdit() {
    setDetailsOpen(false)
    onEditTask(task)
  }

  function handleDeleteClick() {
    setDetailsOpen(false)
    setDeleteDialogOpen(true)
  }

  function handleConfirmDelete() {
    deleteMutation.mutate(id)
  }

  function handleColumnChange(columnId: string) {
    if (columnId === currentColumnId) {
      return
    }

    onMoveTask(id, columnId)
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
          role="button"
          tabIndex={0}
          className="group relative cursor-pointer rounded-xl border border-border bg-card p-4 shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/30 hover:shadow-md active:cursor-grabbing"
          onClick={handleOpenDetails}
          onKeyDown={handleCardKeyDown}
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
              onClick={(event) => event.stopPropagation()}
              onMouseDown={(event) => event.stopPropagation()}
              onPointerDown={(event) => event.stopPropagation()}
            >
              <Select
                value={currentColumnId}
                onValueChange={handleColumnChange}
                disabled={movePending || columnOptions.length === 0}
              >
                <SelectTrigger
                  size="sm"
                  className="h-6 w-[7.75rem] rounded-full px-2 text-[11px] text-muted-foreground"
                  aria-label="Mover tarea a otra columna"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="end">
                  {columnOptions.map((column) => (
                    <SelectItem key={column.id} value={column.id}>
                      {column.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
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
      <TaskDetailsModal
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
        title={title}
        description={description}
        priorityLabel={label}
        priorityClassName={className}
        dueDate={dueDate}
        tag={tag}
        assignees={assignees}
        comments={comments}
        attachments={attachments}
        deletePending={deleteMutation.isPending}
        onEdit={handleEdit}
        onDelete={handleDeleteClick}
      />
      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title="Eliminar tarea"
        description={`Se eliminará "${title}". Esta acción no se puede deshacer.`}
        confirmLabel="Eliminar tarea"
        isPending={deleteMutation.isPending}
        onConfirm={handleConfirmDelete}
      />
    </>
  )
}
