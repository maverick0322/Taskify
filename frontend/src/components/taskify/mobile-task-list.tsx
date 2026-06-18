"use client"

import { AlertCircle, Clock, Inbox, MessageSquare, Paperclip } from "lucide-react"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { formatTaskDueDateLabel } from "@/lib/task-dates"
import { cn } from "@/lib/utils"
import type {
  Task,
  TaskAssignee,
  TaskPriority,
  TaskStatus,
} from "@/services/taskService"

type MobilePriority = "Alta" | "Media" | "Baja"
type MobileStatus = "Pendiente" | "En Progreso" | "Completado"

interface MobileTaskListProps {
  tasks: Task[]
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
}

interface MobileTask {
  id: string
  title: string
  description?: string
  priority: MobilePriority
  dueDate: string
  tag?: string
  assignees?: TaskAssignee[]
  comments?: number
  attachments?: number
  status: MobileStatus
}

const priorityConfig: Record<MobilePriority, { className: string; dotColor: string }> = {
  Alta: {
    className: "bg-red-100 text-red-700 border-red-200",
    dotColor: "bg-red-500",
  },
  Media: {
    className: "bg-amber-100 text-amber-700 border-amber-200",
    dotColor: "bg-amber-500",
  },
  Baja: {
    className: "bg-blue-100 text-blue-700 border-blue-200",
    dotColor: "bg-blue-500",
  },
}

const groups: { priority: MobilePriority; label: string }[] = [
  { priority: "Alta", label: "Prioridad Alta" },
  { priority: "Media", label: "Prioridad Media" },
  { priority: "Baja", label: "Prioridad Baja" },
]

const statusConfig: Record<MobileStatus, { className: string }> = {
  Pendiente: { className: "bg-slate-100 text-slate-600 border-slate-200" },
  "En Progreso": { className: "bg-indigo-100 text-indigo-700 border-indigo-200" },
  Completado: { className: "bg-emerald-100 text-emerald-700 border-emerald-200" },
}

function MobileTaskRow({ task }: { task: MobileTask }) {
  const status = statusConfig[task.status]

  return (
    <article className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 shadow-sm">
      <div className="flex items-start justify-between gap-2">
        <h3 className="flex-1 text-sm font-semibold leading-snug text-card-foreground">
          {task.title}
        </h3>
        <Badge
          variant="outline"
          className={cn(
            "shrink-0 rounded-full border px-2 py-0.5 text-[11px] font-semibold",
            status.className,
          )}
        >
          {task.status}
        </Badge>
      </div>

      {task.description ? (
        <p className="line-clamp-2 text-xs leading-relaxed text-muted-foreground">
          {task.description}
        </p>
      ) : null}

      <div className="flex items-center justify-between border-t border-border/60 pt-2">
        <div className="flex items-center gap-1 text-muted-foreground">
          <Clock className="size-3" />
          <span className="text-[11px] font-medium">{task.dueDate}</span>
        </div>

        <div className="flex items-center gap-2">
          {(task.comments ?? 0) > 0 ? (
            <div className="flex items-center gap-1 text-muted-foreground">
              <MessageSquare className="size-3" />
              <span className="text-[11px]">{task.comments}</span>
            </div>
          ) : null}
          {(task.attachments ?? 0) > 0 ? (
            <div className="flex items-center gap-1 text-muted-foreground">
              <Paperclip className="size-3" />
              <span className="text-[11px]">{task.attachments}</span>
            </div>
          ) : null}
          {(task.assignees?.length ?? 0) > 0 ? (
            <div className="flex -space-x-1.5">
              {task.assignees?.slice(0, 3).map((assignee) => (
                <Avatar key={`${task.id}-${assignee.seed}`} className="size-5 ring-1 ring-card">
                  <AvatarImage
                    src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${assignee.seed}`}
                    alt={assignee.name}
                  />
                  <AvatarFallback className="bg-primary/10 text-[9px] font-bold text-primary">
                    {assignee.name.charAt(0)}
                  </AvatarFallback>
                </Avatar>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </article>
  )
}

export function MobileTaskList({
  tasks,
  isLoading = false,
  isError = false,
  errorMessage,
}: MobileTaskListProps) {
  const mobileTasks = tasks.map(taskResponseToMobileTask)

  return (
    <main
      className="flex-1 overflow-y-auto bg-canvas px-4 py-5"
      aria-label="Lista de tareas por prioridad"
    >
      {isLoading ? <MobileTaskListSkeleton /> : null}

      {!isLoading && isError ? (
        <div className="flex min-h-64 flex-col items-center justify-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-8 text-center text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300">
          <AlertCircle className="size-6" />
          <p className="text-sm font-medium">
            {errorMessage ?? "No pudimos cargar este tablero."}
          </p>
        </div>
      ) : null}

      {!isLoading && !isError && mobileTasks.length === 0 ? (
        <div className="flex min-h-64 flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border bg-card/60 px-4 py-8 text-center">
          <Inbox className="size-7 text-muted-foreground" />
          <div className="flex flex-col gap-1">
            <p className="text-sm font-semibold text-foreground">Este tablero está vacío</p>
            <p className="text-xs text-muted-foreground">
              Crea una tarea para empezar a organizarlo.
            </p>
          </div>
        </div>
      ) : null}

      {!isLoading && !isError && mobileTasks.length > 0 ? (
        <div className="flex flex-col gap-6 pb-6">
          {groups.map(({ priority, label }) => {
            const priorityTasks = mobileTasks.filter((task) => task.priority === priority)
            const { dotColor } = priorityConfig[priority]

            return (
              <section key={priority} aria-labelledby={`group-${priority}`}>
                <div className="mb-3 flex items-center gap-2">
                  <span
                    className={cn("size-2.5 shrink-0 rounded-full", dotColor)}
                    aria-hidden="true"
                  />
                  <h2
                    id={`group-${priority}`}
                    className="text-xs font-semibold uppercase tracking-wider text-muted-foreground"
                  >
                    {label}
                  </h2>
                  <span className="ml-auto text-xs font-medium text-muted-foreground">
                    {priorityTasks.length}
                  </span>
                </div>

                {priorityTasks.length > 0 ? (
                  <div className="flex flex-col gap-3">
                    {priorityTasks.map((task) => (
                      <MobileTaskRow key={task.id} task={task} />
                    ))}
                  </div>
                ) : (
                  <p className="rounded-xl border border-dashed border-border px-4 py-5 text-center text-xs text-muted-foreground">
                    Sin tareas en esta prioridad.
                  </p>
                )}
              </section>
            )
          })}
        </div>
      ) : null}
    </main>
  )
}

function MobileTaskListSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      {Array.from({ length: 4 }).map((_, index) => (
        <Skeleton key={index} className="h-28 rounded-xl" />
      ))}
    </div>
  )
}

function taskResponseToMobileTask(task: Task): MobileTask {
  return {
    id: task.id,
    title: task.title,
    description: task.description || undefined,
    priority: priorityToMobilePriority(task.priority),
    dueDate: formatTaskDueDateLabel(task.dueDate),
    tag: task.tag,
    assignees: task.assignees ?? [],
    comments: task.comments ?? 0,
    attachments: task.attachments ?? 0,
    status: statusToMobileStatus(task.status),
  }
}

function priorityToMobilePriority(priority: TaskPriority): MobilePriority {
  const priorities: Record<TaskPriority, MobilePriority> = {
    high: "Alta",
    medium: "Media",
    low: "Baja",
  }

  return priorities[priority]
}

function statusToMobileStatus(status: TaskStatus): MobileStatus {
  const statuses: Record<TaskStatus, MobileStatus> = {
    todo: "Pendiente",
    in_progress: "En Progreso",
    done: "Completado",
  }

  return statuses[status]
}
