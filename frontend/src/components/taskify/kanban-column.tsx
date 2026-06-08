"use client"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { TaskCard } from "@/components/taskify/task-card"
import type { KanbanTask } from "@/components/taskify/kanban-board"
import { Plus, MoreHorizontal } from "lucide-react"

interface KanbanColumnProps {
  title: string
  tasks: KanbanTask[]
  accentColor: string
  dotColor: string
}

export function KanbanColumn({
  title,
  tasks,
  accentColor,
  dotColor,
}: KanbanColumnProps) {
  return (
    <section
      className={cn(
        "flex w-72 shrink-0 flex-col rounded-2xl border border-border/60 bg-column",
        "md:w-80"
      )}
      aria-label={`Columna: ${title}`}
    >
      {/* Column Header */}
      <div className="flex items-center justify-between rounded-t-2xl border-b border-border/50 px-4 py-3.5">
        <div className="flex items-center gap-2.5">
          <span className={cn("size-2.5 rounded-full shrink-0", dotColor)} />
          <h2 className="text-sm font-semibold text-foreground">{title}</h2>
          <span
            className={cn(
              "flex size-5 items-center justify-center rounded-full text-[11px] font-bold",
              accentColor
            )}
          >
            {tasks.length}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="size-7 text-muted-foreground hover:text-foreground"
            aria-label={`Más opciones para ${title}`}
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </div>
      </div>

      {/* Cards */}
      <div className="flex flex-1 flex-col gap-3 overflow-y-auto p-3">
        {tasks.map((task) => (
          <TaskCard key={task.id} {...task} />
        ))}

        {/* Empty state */}
        {tasks.length === 0 && (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-border/60 p-6 text-center">
            <p className="text-xs text-muted-foreground">Sin tareas aún</p>
          </div>
        )}
      </div>

      {/* Add Task */}
      <div className="border-t border-border/50 p-3">
        <Button
          variant="ghost"
          className="w-full justify-start gap-2 text-muted-foreground hover:text-foreground hover:bg-accent/50 h-9 text-sm"
        >
          <Plus className="size-4" />
          Agregar tarea
        </Button>
      </div>
    </section>
  )
}
