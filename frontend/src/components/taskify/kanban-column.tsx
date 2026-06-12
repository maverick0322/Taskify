"use client"

import { Droppable } from "@hello-pangea/dnd"
import { useEffect, useState } from "react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { TaskCard } from "@/components/taskify/task-card"
import type { KanbanTask } from "@/components/taskify/kanban-board"
import type { Task } from "@/services/taskService"
import { Check, MoreHorizontal, Palette, Plus, X } from "lucide-react"

export const COLUMN_COLORS = [
  {
    value: "slate",
    dotColor: "bg-slate-400",
    accentColor: "bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300",
  },
  {
    value: "indigo",
    dotColor: "bg-indigo-500",
    accentColor: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/50 dark:text-indigo-300",
  },
  {
    value: "emerald",
    dotColor: "bg-emerald-500",
    accentColor: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300",
  },
  {
    value: "amber",
    dotColor: "bg-amber-500",
    accentColor: "bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300",
  },
  {
    value: "rose",
    dotColor: "bg-rose-500",
    accentColor: "bg-rose-100 text-rose-700 dark:bg-rose-900/50 dark:text-rose-300",
  },
] as const

export type ColumnColor = (typeof COLUMN_COLORS)[number]["value"]

interface KanbanColumnProps {
  columnId: string
  title: string
  color: string
  tasks: KanbanTask[]
  selectedBoardId?: string
  onEditTask: (task: Task) => void
  onAddTask: (columnId: string) => void
  onUpdateColumn: (columnId: string, name: string, color: string) => void
  updatePending: boolean
}

export function columnColorConfig(color: string) {
  return (
    COLUMN_COLORS.find((columnColor) => columnColor.value === color) ??
    COLUMN_COLORS[0]
  )
}

export function KanbanColumn({
  columnId,
  title,
  color,
  tasks,
  selectedBoardId,
  onEditTask,
  onAddTask,
  onUpdateColumn,
  updatePending,
}: KanbanColumnProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [draftTitle, setDraftTitle] = useState(title)
  const [draftColor, setDraftColor] = useState(color)
  const visual = columnColorConfig(draftColor)

  useEffect(() => {
    if (!isEditing) {
      setDraftTitle(title)
      setDraftColor(color)
    }
  }, [color, isEditing, title])

  function commitChanges() {
    const trimmedTitle = draftTitle.trim()
    if (!trimmedTitle || updatePending) {
      return
    }

    if (trimmedTitle !== title || draftColor !== color) {
      onUpdateColumn(columnId, trimmedTitle, draftColor)
    }
    setIsEditing(false)
  }

  function cancelEditing() {
    setDraftTitle(title)
    setDraftColor(color)
    setIsEditing(false)
  }

  return (
    <section
      className={cn(
        "flex w-72 shrink-0 flex-col rounded-2xl border border-border/60 bg-column",
        "md:w-80",
      )}
      aria-label={`Columna: ${title}`}
    >
      <div className="flex items-start justify-between gap-2 rounded-t-2xl border-b border-border/50 px-4 py-3.5">
        <div className="flex min-w-0 flex-1 items-start gap-2.5">
          <span
            className={cn("mt-1.5 size-2.5 shrink-0 rounded-full", visual.dotColor)}
          />
          {isEditing ? (
            <div
              className="flex min-w-0 flex-1 flex-col gap-2"
              onPointerDown={(event) => event.stopPropagation()}
            >
              <Input
                autoFocus
                value={draftTitle}
                onChange={(event) => setDraftTitle(event.target.value)}
                onBlur={commitChanges}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault()
                    commitChanges()
                  }
                  if (event.key === "Escape") {
                    event.preventDefault()
                    cancelEditing()
                  }
                }}
                className="h-8 text-sm font-semibold"
                disabled={updatePending}
              />
              <div className="flex items-center gap-1">
                <Palette className="mr-1 size-3.5 text-muted-foreground" />
                {COLUMN_COLORS.map((columnColor) => (
                  <button
                    key={columnColor.value}
                    type="button"
                    aria-label={`Color ${columnColor.value}`}
                    className={cn(
                      "flex size-5 items-center justify-center rounded-full border border-border transition-transform hover:scale-110",
                      columnColor.dotColor,
                    )}
                    onMouseDown={(event) => event.preventDefault()}
                    onClick={() => setDraftColor(columnColor.value)}
                  >
                    {draftColor === columnColor.value ? (
                      <Check className="size-3 text-white" />
                    ) : null}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <button
              type="button"
              className="min-w-0 rounded-sm text-left text-sm font-semibold text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              onClick={() => setIsEditing(true)}
              onPointerDown={(event) => event.stopPropagation()}
            >
              <span className="block truncate">{title}</span>
            </button>
          )}
          <span
            className={cn(
              "mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full text-[11px] font-bold",
              visual.accentColor,
            )}
          >
            {tasks.length}
          </span>
        </div>
        <div className="flex items-center gap-1">
          {isEditing ? (
            <Button
              variant="ghost"
              size="icon"
              className="size-7 text-muted-foreground hover:text-foreground"
              aria-label="Cancelar edición"
              onMouseDown={(event) => event.preventDefault()}
              onClick={cancelEditing}
              disabled={updatePending}
            >
              <X className="size-4" />
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="icon"
              className="size-7 text-muted-foreground hover:text-foreground"
              aria-label={`Más opciones para ${title}`}
            >
              <MoreHorizontal className="size-4" />
            </Button>
          )}
        </div>
      </div>

      <Droppable droppableId={columnId}>
        {(provided) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className="flex flex-1 flex-col gap-3 overflow-y-auto p-3"
          >
            {tasks.map((task, index) => (
              <TaskCard
                key={task.id}
                index={index}
                selectedBoardId={selectedBoardId}
                onEditTask={onEditTask}
                {...task}
              />
            ))}

            {tasks.length === 0 && (
              <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-border/60 p-6 text-center">
                <p className="text-xs text-muted-foreground">Sin tareas aún</p>
              </div>
            )}
            {provided.placeholder}
          </div>
        )}
      </Droppable>

      <div className="border-t border-border/50 p-3">
        <Button
          variant="ghost"
          className="h-9 w-full justify-start gap-2 text-sm text-muted-foreground hover:bg-accent/50 hover:text-foreground"
          onClick={() => onAddTask(columnId)}
        >
          <Plus className="size-4" />
          Agregar tarea
        </Button>
      </div>
    </section>
  )
}
