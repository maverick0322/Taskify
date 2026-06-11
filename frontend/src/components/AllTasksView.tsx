"use client"

import { useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Checkbox as CheckboxPrimitive, DropdownMenu, Tabs } from "radix-ui"
import {
  Calendar,
  Check,
  Edit,
  Globe,
  LayoutGrid,
  MoreVertical,
  Trash2,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { formatTaskDueDateLabel } from "@/lib/task-dates"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { NewTaskDialog } from "@/components/taskify/new-task-dialog"
import { invalidateTaskCaches } from "@/components/taskify/task-cache"
import { getBoards } from "@/services/boardService"
import {
  deleteTask,
  getTasks,
  updateTaskStatus,
  type Task,
  type TaskStatus,
} from "@/services/taskService"

interface DisplayTask {
  task: Task
  boardName: string | null
  dueDateLabel: string
  priority: "alta" | "media" | "baja"
}

const priorityLabel: Record<Task["priority"], DisplayTask["priority"]> = {
  high: "alta",
  medium: "media",
  low: "baja",
}

const priorityBadge: Record<DisplayTask["priority"], string> = {
  alta: "border-red-200 bg-red-50 text-red-700",
  media: "border-amber-200 bg-amber-50 text-amber-700",
  baja: "border-blue-200 bg-blue-50 text-blue-700",
}

interface TaskRowProps {
  displayTask: DisplayTask
  onEdit: (task: Task) => void
  onDelete: (task: Task) => void
  onStatusChange: (task: Task, status: TaskStatus) => void
  statusPending: boolean
  deletePending: boolean
}

function TaskRow({
  displayTask,
  onEdit,
  onDelete,
  onStatusChange,
  statusPending,
  deletePending,
}: TaskRowProps) {
  const { task, boardName, dueDateLabel, priority } = displayTask
  const isDone = task.status === "done"

  function handleToggleStatus(checked: boolean | "indeterminate") {
    onStatusChange(task, checked === true ? "done" : "todo")
  }

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onEdit(task)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault()
          onEdit(task)
        }
      }}
      className="flex cursor-pointer items-center justify-between border-b border-border/40 p-4 transition-colors last:border-b-0 hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <div className="flex min-w-0 items-center gap-3">
        <CheckboxPrimitive.Root
          id={`task-${task.id}`}
          checked={isDone}
          disabled={statusPending}
          onCheckedChange={handleToggleStatus}
          onClick={(event) => event.stopPropagation()}
          className="flex size-4 shrink-0 items-center justify-center rounded border border-input bg-background transition-colors data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={
            isDone
              ? `Marcar ${task.title} como pendiente`
              : `Marcar ${task.title} como terminada`
          }
        >
          <CheckboxPrimitive.Indicator>
            <Check className="size-3" />
          </CheckboxPrimitive.Indicator>
        </CheckboxPrimitive.Root>
        <div className="min-w-0">
          <p
            className={cn(
              "truncate text-sm font-medium leading-snug text-foreground",
              isDone && "text-muted-foreground line-through",
            )}
          >
            {task.title}
          </p>
          {task.description ? (
            <p className="mt-0.5 truncate text-xs text-muted-foreground">
              {task.description}
            </p>
          ) : null}
        </div>
      </div>

      <div className="ml-4 flex shrink-0 items-center gap-4">
        <Badge
          variant="outline"
          className={cn("hidden text-xs font-normal capitalize sm:inline-flex", priorityBadge[priority])}
        >
          {priority}
        </Badge>

        {boardName ? (
          <Badge variant="secondary" className="text-xs font-normal">
            {boardName}
          </Badge>
        ) : (
          <Badge
            variant="outline"
            className="gap-1 text-xs font-normal text-muted-foreground"
          >
            <Globe className="size-3" />
            Global
          </Badge>
        )}

        <div className="hidden items-center gap-1.5 text-xs text-muted-foreground sm:flex">
          <Calendar className="size-3.5" />
          <span>{dueDateLabel}</span>
        </div>

        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-7 text-muted-foreground"
              aria-label="Más opciones"
              onClick={(event) => event.stopPropagation()}
            >
              <MoreVertical className="size-4" />
            </Button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content
              align="end"
              sideOffset={6}
              className="z-50 min-w-36 rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-md"
              onClick={(event) => event.stopPropagation()}
            >
              <DropdownMenu.Item
                className="flex cursor-default select-none items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none focus:bg-accent focus:text-accent-foreground"
                onSelect={() => onEdit(task)}
              >
                <Edit className="size-3.5" />
                Editar
              </DropdownMenu.Item>
              <DropdownMenu.Item
                className="flex cursor-default select-none items-center gap-2 rounded-md px-2 py-1.5 text-sm text-red-600 outline-none focus:bg-red-50 focus:text-red-700"
                disabled={deletePending}
                onSelect={() => onDelete(task)}
              >
                <Trash2 className="size-3.5" />
                Eliminar
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </div>
    </div>
  )
}

function TaskList({
  tasks,
  onEdit,
  onDelete,
  onStatusChange,
  statusPending,
  deletePending,
}: {
  tasks: DisplayTask[]
  onEdit: (task: Task) => void
  onDelete: (task: Task) => void
  onStatusChange: (task: Task, status: TaskStatus) => void
  statusPending: boolean
  deletePending: boolean
}) {
  return (
    <div className="mt-4 flex flex-col overflow-hidden rounded-lg border border-border/40 bg-card shadow-sm">
      {tasks.map((task) => (
        <TaskRow
          key={task.task.id}
          displayTask={task}
          onEdit={onEdit}
          onDelete={onDelete}
          onStatusChange={onStatusChange}
          statusPending={statusPending}
          deletePending={deletePending}
        />
      ))}
    </div>
  )
}

function EmptyTaskState({
  icon: Icon,
  message,
}: {
  icon: typeof Globe
  message: string
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
      <Icon className="mb-3 size-8 opacity-30" />
      <p className="text-sm">{message}</p>
    </div>
  )
}

function BoardGroupedList({
  tasks,
  onEdit,
  onDelete,
  onStatusChange,
  statusPending,
  deletePending,
}: {
  tasks: DisplayTask[]
  onEdit: (task: Task) => void
  onDelete: (task: Task) => void
  onStatusChange: (task: Task, status: TaskStatus) => void
  statusPending: boolean
  deletePending: boolean
}) {
  const boards = Array.from(new Set(tasks.map((task) => task.boardName ?? "Sin tablero")))

  return (
    <div className="mt-4 flex flex-col overflow-hidden rounded-lg border border-border/40 bg-card shadow-sm">
      {boards.map((board, index) => {
        const groupedTasks = tasks.filter(
          (task) => (task.boardName ?? "Sin tablero") === board,
        )

        return (
          <div
            key={board}
            className={index > 0 ? "border-t border-border/40" : undefined}
          >
            <div className="flex items-center gap-2 border-b border-border/40 bg-muted/30 px-4 py-2.5">
              <LayoutGrid className="size-3.5 text-muted-foreground" />
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {board}
              </span>
              <span className="ml-auto text-xs font-normal text-muted-foreground">
                {groupedTasks.length} {groupedTasks.length === 1 ? "tarea" : "tareas"}
              </span>
            </div>
            {groupedTasks.map((task) => (
              <TaskRow
                key={task.task.id}
                displayTask={task}
                onEdit={onEdit}
                onDelete={onDelete}
                onStatusChange={onStatusChange}
                statusPending={statusPending}
                deletePending={deletePending}
              />
            ))}
          </div>
        )
      })}
    </div>
  )
}

export function AllTasksView() {
  const queryClient = useQueryClient()
  const [taskToEdit, setTaskToEdit] = useState<Task | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const {
    data: tasks = [],
    isLoading: tasksLoading,
    isError: tasksIsError,
    error: tasksError,
  } = useQuery({
    queryKey: ["tasks", "global"],
    queryFn: () => getTasks(),
  })
  const { data: boards = [] } = useQuery({
    queryKey: ["boards"],
    queryFn: getBoards,
  })

  const boardNamesByID = useMemo(
    () => new Map(boards.map((board) => [board.id, board.name])),
    [boards],
  )
  const displayTasks = useMemo(
    () =>
      tasks.map((task) => ({
        task,
        boardName: task.boardId ? boardNamesByID.get(task.boardId) ?? "Sin tablero" : null,
        dueDateLabel: formatTaskDueDateLabel(task.dueDate),
        priority: priorityLabel[task.priority],
      })),
    [boardNamesByID, tasks],
  )
  const globalTasks = displayTasks.filter((task) => !task.task.boardId)
  const boardTasks = displayTasks.filter((task) => task.task.boardId)
  const tasksErrorMessage =
    tasksError instanceof Error
      ? tasksError.message
      : "No se pudieron cargar las tareas"

  const statusMutation = useMutation({
    mutationFn: updateTaskStatus,
    onMutate: async ({ taskId, status }) => {
      await queryClient.cancelQueries({ queryKey: ["tasks", "global"] })
      const previousTasks = queryClient.getQueryData<Task[]>(["tasks", "global"])

      queryClient.setQueryData<Task[]>(["tasks", "global"], (currentTasks = []) =>
        currentTasks.map((task) =>
          task.id === taskId ? { ...task, status } : task,
        ),
      )

      return { previousTasks }
    },
    onError: (_error, _variables, context) => {
      if (context?.previousTasks) {
        queryClient.setQueryData(["tasks", "global"], context.previousTasks)
      }
    },
    onSettled: (_data, _error, variables) => {
      const task = tasks.find((currentTask) => currentTask.id === variables.taskId)
      invalidateTaskCaches(queryClient, task?.boardId)
    },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteTask,
    onSuccess: (_data, taskId) => {
      const task = tasks.find((currentTask) => currentTask.id === taskId)
      invalidateTaskCaches(queryClient, task?.boardId)
    },
  })

  function handleEditTask(task: Task) {
    setTaskToEdit(task)
    setEditDialogOpen(true)
  }

  function handleEditDialogOpenChange(open: boolean) {
    setEditDialogOpen(open)
    if (!open) {
      setTaskToEdit(null)
    }
  }

  function handleDeleteTask(task: Task) {
    if (!window.confirm(`¿Eliminar la tarea "${task.title}"?`)) {
      return
    }

    deleteMutation.mutate(task.id)
  }

  function handleStatusChange(task: Task, status: TaskStatus) {
    if (task.status === status) {
      return
    }

    statusMutation.mutate({ taskId: task.id, status })
  }

  if (tasksLoading) {
    return (
      <main className="flex flex-1 items-center justify-center bg-slate-50 p-8 text-sm font-medium text-muted-foreground dark:bg-background">
        Cargando tareas...
      </main>
    )
  }

  if (tasksIsError) {
    return (
      <main className="flex flex-1 items-center justify-center bg-slate-50 px-6 text-center text-sm font-medium text-red-600 dark:bg-background">
        {tasksErrorMessage}
      </main>
    )
  }

  return (
    <>
      <main className="flex flex-1 flex-col gap-6 overflow-y-auto bg-slate-50 p-6 dark:bg-background md:p-8">
        <header className="flex flex-col gap-1">
          <h1 className="text-balance text-3xl font-bold tracking-tight text-foreground">
            Mis Tareas
          </h1>
          <p className="text-sm text-muted-foreground">
            Gestiona y filtra todas tus actividades en un solo lugar.
          </p>
        </header>

        <Tabs.Root defaultValue="all">
          <Tabs.List className="flex h-auto w-full justify-start gap-0 rounded-none border-b border-border/40 bg-transparent p-0">
            <Tabs.Trigger
              value="all"
              className="rounded-none bg-transparent px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=active]:shadow-none"
            >
              Todas
              <span className="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-xs font-normal text-muted-foreground">
                {displayTasks.length}
              </span>
            </Tabs.Trigger>

            <Tabs.Trigger
              value="global"
              className="rounded-none bg-transparent px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=active]:shadow-none"
            >
              Globales
              <span className="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-xs font-normal text-muted-foreground">
                {globalTasks.length}
              </span>
            </Tabs.Trigger>

            <Tabs.Trigger
              value="byboard"
              className="rounded-none bg-transparent px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground data-[state=active]:shadow-none"
            >
              Por Tablero
              <span className="ml-1.5 rounded-full bg-muted px-1.5 py-0.5 text-xs font-normal text-muted-foreground">
                {boardTasks.length}
              </span>
            </Tabs.Trigger>
          </Tabs.List>

          <Tabs.Content value="all" className="mt-0">
            {displayTasks.length > 0 ? (
              <TaskList
                tasks={displayTasks}
                onEdit={handleEditTask}
                onDelete={handleDeleteTask}
                onStatusChange={handleStatusChange}
                statusPending={statusMutation.isPending}
                deletePending={deleteMutation.isPending}
              />
            ) : (
              <EmptyTaskState icon={Calendar} message="No hay tareas todavía" />
            )}
          </Tabs.Content>

          <Tabs.Content value="global" className="mt-0">
            {globalTasks.length > 0 ? (
              <TaskList
                tasks={globalTasks}
                onEdit={handleEditTask}
                onDelete={handleDeleteTask}
                onStatusChange={handleStatusChange}
                statusPending={statusMutation.isPending}
                deletePending={deleteMutation.isPending}
              />
            ) : (
              <EmptyTaskState icon={Globe} message="No hay tareas globales" />
            )}
          </Tabs.Content>

          <Tabs.Content value="byboard" className="mt-0">
            {boardTasks.length > 0 ? (
              <BoardGroupedList
                tasks={boardTasks}
                onEdit={handleEditTask}
                onDelete={handleDeleteTask}
                onStatusChange={handleStatusChange}
                statusPending={statusMutation.isPending}
                deletePending={deleteMutation.isPending}
              />
            ) : (
              <EmptyTaskState
                icon={LayoutGrid}
                message="No hay tareas asociadas a tableros"
              />
            )}
          </Tabs.Content>
        </Tabs.Root>
      </main>

      <NewTaskDialog
        open={editDialogOpen}
        onOpenChange={handleEditDialogOpenChange}
        taskToEdit={taskToEdit}
      />
    </>
  )
}
