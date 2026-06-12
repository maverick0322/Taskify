import { DragDropContext, type DropResult } from "@hello-pangea/dnd"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"

import {
  COLUMN_COLORS,
  KanbanColumn,
  columnColorConfig,
  type ColumnColor,
} from "@/components/taskify/kanban-column"
import { NewTaskDialog } from "@/components/taskify/new-task-dialog"
import { invalidateTaskCaches } from "@/components/taskify/task-cache"
import { formatTaskDueDateLabel } from "@/lib/task-dates"
import { cn } from "@/lib/utils"
import { Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  createColumn,
  getBoardColumns,
  updateColumn,
  type BoardColumn,
} from "@/services/boardService"
import {
  moveTaskToColumn,
  updateTaskStatus,
  type Task,
  type TaskPriority,
  type TaskStatus,
} from "@/services/taskService"

export type KanbanTaskPriority = "Alta" | "Media" | "Baja"

export interface KanbanTask {
  id: string
  task: Task
  title: string
  description?: string
  priority: KanbanTaskPriority
  dueDate: string
  tag?: string
  assignees?: { name: string; seed: string }[]
  comments?: number
  attachments?: number
}

const DEFAULT_COLUMNS: Array<{
  name: string
  color: ColumnColor
  status: TaskStatus
}> = [
  { name: "Pendiente", color: "slate", status: "todo" },
  { name: "En Progreso", color: "indigo", status: "in_progress" },
  { name: "Completado", color: "emerald", status: "done" },
]

interface KanbanBoardProps {
  selectedBoardId?: string
  tasks: Task[]
}

interface MoveTaskVariables {
  taskId: string
  columnId: string | null
  status?: TaskStatus
}

interface MoveTaskContext {
  previousTasks?: Task[]
}

export function KanbanBoard({ selectedBoardId, tasks }: KanbanBoardProps) {
  const queryClient = useQueryClient()
  const [taskToEdit, setTaskToEdit] = useState<Task | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [newTaskStatus, setNewTaskStatus] = useState<TaskStatus | undefined>()
  const [newTaskColumnId, setNewTaskColumnId] = useState<string | undefined>()
  const [newColumnOpen, setNewColumnOpen] = useState(false)
  const [newColumnName, setNewColumnName] = useState("")
  const [newColumnColor, setNewColumnColor] = useState<ColumnColor>("slate")
  const [newColumnError, setNewColumnError] = useState("")
  const [bootstrappingBoardId, setBootstrappingBoardId] = useState<string | null>(null)
  const tasksQueryKey = useMemo(() => ["tasks", selectedBoardId], [selectedBoardId])
  const columnsQueryKey = useMemo(
    () => ["boards", selectedBoardId, "columns"],
    [selectedBoardId],
  )

  const {
    data: boardColumns = [],
    isLoading: columnsLoading,
  } = useQuery({
    queryKey: columnsQueryKey,
    queryFn: () => getBoardColumns(selectedBoardId ?? ""),
    enabled: Boolean(selectedBoardId),
  })

  useEffect(() => {
    if (!selectedBoardId || columnsLoading || boardColumns.length > 0) {
      return
    }
    if (bootstrappingBoardId === selectedBoardId) {
      return
    }

    setBootstrappingBoardId(selectedBoardId)
    Promise.all(
      DEFAULT_COLUMNS.map((column, index) =>
        createColumn(selectedBoardId, {
          name: column.name,
          color: column.color,
          position: index,
        }),
      ),
    )
      .then(() => queryClient.invalidateQueries({ queryKey: columnsQueryKey }))
      .finally(() =>
        setBootstrappingBoardId((currentBoardId) =>
          currentBoardId === selectedBoardId ? null : currentBoardId,
        ),
      )
  }, [
    boardColumns.length,
    bootstrappingBoardId,
    columnsLoading,
    columnsQueryKey,
    queryClient,
    selectedBoardId,
  ])

  const visibleColumns = useMemo(
    () => [...boardColumns].sort((first, second) => first.position - second.position),
    [boardColumns],
  )

  const moveMutation = useMutation<void, Error, MoveTaskVariables, MoveTaskContext>({
    mutationFn: async ({ taskId, columnId, status }) => {
      await moveTaskToColumn(taskId, columnId)
      if (status) {
        await updateTaskStatus({ taskId, status })
      }
    },
    onMutate: async ({ taskId, columnId, status }) => {
      await queryClient.cancelQueries({ queryKey: tasksQueryKey })
      const previousTasks = queryClient.getQueryData<Task[]>(tasksQueryKey)

      queryClient.setQueryData<Task[]>(tasksQueryKey, (currentTasks = []) =>
        currentTasks.map((task) =>
          task.id === taskId
            ? { ...task, columnId, ...(status ? { status } : {}) }
            : task,
        ),
      )

      return { previousTasks }
    },
    onError: (_error, _variables, context) => {
      if (context?.previousTasks) {
        queryClient.setQueryData(tasksQueryKey, context.previousTasks)
      }
    },
    onSettled: () => {
      invalidateTaskCaches(queryClient, selectedBoardId)
    },
  })

  const createColumnMutation = useMutation({
    mutationFn: async (input: { name: string; color: string }) => {
      if (!selectedBoardId) {
        throw new Error("Selecciona un tablero para crear columnas.")
      }

      return createColumn(selectedBoardId, {
        name: input.name,
        color: input.color,
        position: visibleColumns.length,
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: columnsQueryKey })
      setNewColumnName("")
      setNewColumnColor("slate")
      setNewColumnError("")
      setNewColumnOpen(false)
    },
    onError: () => {
      setNewColumnError("No pudimos crear la columna. Intentalo de nuevo.")
    },
  })

  const updateColumnMutation = useMutation({
    mutationFn: ({ columnId, name, color }: { columnId: string; name: string; color: string }) =>
      updateColumn(columnId, { name, color }),
    onMutate: async ({ columnId, name, color }) => {
      await queryClient.cancelQueries({ queryKey: columnsQueryKey })
      const previousColumns = queryClient.getQueryData<BoardColumn[]>(columnsQueryKey)
      queryClient.setQueryData<BoardColumn[]>(columnsQueryKey, (currentColumns = []) =>
        currentColumns.map((column) =>
          column.id === columnId ? { ...column, name, color } : column,
        ),
      )
      return { previousColumns }
    },
    onError: (_error, _variables, context) => {
      if (context?.previousColumns) {
        queryClient.setQueryData(columnsQueryKey, context.previousColumns)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: columnsQueryKey })
    },
  })

  function handleDragEnd(result: DropResult) {
    const { destination, draggableId, source } = result

    if (!destination || !selectedBoardId || destination.droppableId === source.droppableId) {
      return
    }

    const status = statusForColumn(destination.droppableId, visibleColumns)
    moveMutation.mutate({
      taskId: draggableId,
      columnId: destination.droppableId,
      ...(status ? { status } : {}),
    })
  }

  function handleEditTask(task: Task) {
    setTaskToEdit(task)
    setNewTaskStatus(undefined)
    setNewTaskColumnId(undefined)
    setEditDialogOpen(true)
  }

  function handleAddTask(columnId: string) {
    setTaskToEdit(null)
    setNewTaskColumnId(columnId)
    setNewTaskStatus(statusForColumn(columnId, visibleColumns))
    setEditDialogOpen(true)
  }

  function handleEditDialogOpenChange(open: boolean) {
    setEditDialogOpen(open)
    if (!open) {
      setTaskToEdit(null)
      setNewTaskStatus(undefined)
      setNewTaskColumnId(undefined)
    }
  }

  function handleCreateColumn() {
    const trimmedName = newColumnName.trim()
    if (!trimmedName) {
      setNewColumnError("Escribe un nombre para la columna.")
      return
    }

    createColumnMutation.mutate({ name: trimmedName, color: newColumnColor })
  }

  return (
    <main
      className="flex-1 overflow-x-auto kanban-scroll bg-canvas"
      aria-label="Tablero Kanban"
    >
      <DragDropContext onDragEnd={handleDragEnd}>
        <div className="flex h-full gap-4 p-5 md:p-6">
          {visibleColumns.map((column) => (
            <KanbanColumn
              key={column.id}
              columnId={column.id}
              title={column.name}
              color={column.color}
              tasks={tasks
                .filter((task) => taskBelongsToColumn(task, column, visibleColumns))
                .map(taskResponseToKanbanTask)}
              selectedBoardId={selectedBoardId}
              onEditTask={handleEditTask}
              onAddTask={handleAddTask}
              onUpdateColumn={(columnId, name, color) =>
                updateColumnMutation.mutate({ columnId, name, color })
              }
              updatePending={updateColumnMutation.isPending}
            />
          ))}

          <div className="flex shrink-0 items-start pt-0.5">
            {newColumnOpen ? (
              <div className="flex w-64 flex-col gap-3 rounded-2xl border border-border/60 bg-column p-3 shadow-sm">
                <Input
                  autoFocus
                  placeholder="Nombre de columna"
                  value={newColumnName}
                  onChange={(event) => {
                    setNewColumnName(event.target.value)
                    setNewColumnError("")
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") {
                      handleCreateColumn()
                    }
                    if (event.key === "Escape") {
                      setNewColumnOpen(false)
                      setNewColumnName("")
                      setNewColumnError("")
                    }
                  }}
                  disabled={createColumnMutation.isPending}
                />
                <div className="flex items-center gap-1.5">
                  {COLUMN_COLORS.map((color) => (
                    <button
                      key={color.value}
                      type="button"
                      aria-label={`Color ${color.value}`}
                      className={cn(
                        "size-5 rounded-full border border-border transition-transform hover:scale-110",
                        columnColorConfig(color.value).dotColor,
                        newColumnColor === color.value && "ring-2 ring-ring ring-offset-2",
                      )}
                      onClick={() => setNewColumnColor(color.value)}
                    />
                  ))}
                </div>
                {newColumnError ? (
                  <p className="text-xs font-medium text-red-600">
                    {newColumnError}
                  </p>
                ) : null}
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    className="flex-1"
                    disabled={createColumnMutation.isPending}
                    onClick={handleCreateColumn}
                  >
                    {createColumnMutation.isPending ? "Creando..." : "Crear"}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={createColumnMutation.isPending}
                    onClick={() => {
                      setNewColumnOpen(false)
                      setNewColumnName("")
                      setNewColumnError("")
                    }}
                  >
                    Cancelar
                  </Button>
                </div>
              </div>
            ) : (
              <Button
                variant="outline"
                className="h-12 w-44 gap-2 border-dashed border-border/80 text-muted-foreground hover:text-foreground hover:border-border hover:bg-column"
                disabled={!selectedBoardId || bootstrappingBoardId === selectedBoardId}
                onClick={() => setNewColumnOpen(true)}
              >
                <Plus className="size-4" />
                Nueva columna
              </Button>
            )}
          </div>
        </div>
      </DragDropContext>
      <NewTaskDialog
        open={editDialogOpen}
        onOpenChange={handleEditDialogOpenChange}
        selectedBoardId={selectedBoardId}
        taskToEdit={taskToEdit}
        initialStatus={newTaskStatus}
        initialColumnId={newTaskColumnId}
      />
    </main>
  )
}

function statusForColumn(columnId: string, columns: BoardColumn[]): TaskStatus | undefined {
  const columnIndex = columns.findIndex((column) => column.id === columnId)
  return DEFAULT_COLUMNS[columnIndex]?.status
}

function taskBelongsToColumn(task: Task, column: BoardColumn, columns: BoardColumn[]) {
  if (task.columnId) {
    return task.columnId === column.id
  }

  return statusForColumn(column.id, columns) === task.status
}

function taskResponseToKanbanTask(task: Task): KanbanTask {
  return {
    id: task.id,
    task,
    title: task.title,
    description: task.description || undefined,
    priority: priorityToKanbanPriority(task.priority),
    dueDate: formatTaskDueDateLabel(task.dueDate),
    tag: task.tag,
    assignees: task.assignees ?? [],
    comments: task.comments ?? 0,
    attachments: task.attachments ?? 0,
  }
}

function priorityToKanbanPriority(priority: TaskPriority): KanbanTaskPriority {
  const priorities: Record<TaskPriority, KanbanTaskPriority> = {
    high: "Alta",
    medium: "Media",
    low: "Baja",
  }

  return priorities[priority]
}
