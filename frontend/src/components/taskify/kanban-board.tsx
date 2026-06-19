import { DragDropContext, type DropResult } from "@hello-pangea/dnd"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"

import {
  COLUMN_COLORS,
  KanbanColumn,
  columnColorConfig,
  type ColumnColor,
} from "@/components/taskify/kanban-column"
import {
  DEFAULT_KANBAN_COLUMNS,
  buildVisibleColumns,
  movePayloadForColumn,
  statusForColumn,
  taskBelongsToColumn,
  type DisplayColumn,
} from "@/components/taskify/kanban-column-helpers"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { NewTaskDialog } from "@/components/taskify/new-task-dialog"
import { invalidateTaskCaches } from "@/components/taskify/task-cache"
import { formatTaskDueDateLabel } from "@/lib/task-dates"
import { cn } from "@/lib/utils"
import { Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  createColumn,
  deleteColumn,
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
  previousLocalTasks?: Task[]
  previousTasks?: Task[]
  previousGlobalTasks?: Task[]
}

export function KanbanBoard({ selectedBoardId, tasks }: KanbanBoardProps) {
  const queryClient = useQueryClient()
  const [taskToEdit, setTaskToEdit] = useState<Task | null>(null)
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [localTasks, setLocalTasks] = useState<Task[]>(tasks)
  const [newTaskStatus, setNewTaskStatus] = useState<TaskStatus | undefined>()
  const [newTaskColumnId, setNewTaskColumnId] = useState<string | undefined>()
  const [newColumnOpen, setNewColumnOpen] = useState(false)
  const [newColumnName, setNewColumnName] = useState("")
  const [newColumnColor, setNewColumnColor] = useState<ColumnColor>("slate")
  const [newColumnError, setNewColumnError] = useState("")
  const [bootstrappingBoardId, setBootstrappingBoardId] = useState<string | null>(null)
  const [columnToDelete, setColumnToDelete] = useState<{
    id: string
    title: string
  } | null>(null)
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
      DEFAULT_KANBAN_COLUMNS.map((column, index) =>
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

  const persistedColumns = useMemo(
    () => [...boardColumns].sort((first, second) => first.position - second.position),
    [boardColumns],
  )

  const visibleColumns = useMemo<DisplayColumn[]>(
    () => buildVisibleColumns(persistedColumns, selectedBoardId ?? ""),
    [persistedColumns, selectedBoardId],
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
      await queryClient.cancelQueries({ queryKey: ["tasks", "global"] })
      const previousLocalTasks = localTasks
      const previousTasks = queryClient.getQueryData<Task[]>(tasksQueryKey)
      const previousGlobalTasks = queryClient.getQueryData<Task[]>(["tasks", "global"])

      const applyTaskMove = (currentTasks: Task[] = []) =>
        currentTasks.map((task) =>
          task.id === taskId
            ? { ...task, columnId, ...(status ? { status } : {}) }
            : task,
        )

      setLocalTasks((currentTasks) => applyTaskMove(currentTasks))
      queryClient.setQueryData<Task[]>(tasksQueryKey, applyTaskMove)
      queryClient.setQueryData<Task[]>(["tasks", "global"], applyTaskMove)

      return { previousLocalTasks, previousTasks, previousGlobalTasks }
    },
    onError: (_error, _variables, context) => {
      if (context?.previousLocalTasks) {
        setLocalTasks(context.previousLocalTasks)
      }
      if (context?.previousTasks) {
        queryClient.setQueryData(tasksQueryKey, context.previousTasks)
      }
      if (context?.previousGlobalTasks) {
        queryClient.setQueryData(["tasks", "global"], context.previousGlobalTasks)
      }
    },
    onSettled: () => {
      invalidateTaskCaches(queryClient, selectedBoardId)
    },
  })

  useEffect(() => {
    if (!moveMutation.isPending) {
      setLocalTasks(tasks)
    }
  }, [moveMutation.isPending, tasks])

  const createColumnMutation = useMutation({
    mutationFn: async (input: { name: string; color: string }) => {
      if (!selectedBoardId) {
        throw new Error("Selecciona un tablero para crear columnas.")
      }

      return createColumn(selectedBoardId, {
        name: input.name,
        color: input.color,
        position: persistedColumns.length,
      })
    },
    onSuccess: (createdColumn, variables) => {
      queryClient.setQueryData<BoardColumn[]>(columnsQueryKey, (currentColumns = []) => {
        const normalizedColumn = {
          ...createdColumn,
          color: createdColumn.color || variables.color,
        }

        const withoutDuplicate = currentColumns.filter(
          (column) => column.id !== normalizedColumn.id,
        )
        return [...withoutDuplicate, normalizedColumn].sort(
          (first, second) => first.position - second.position,
        )
      })
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

  const deleteColumnMutation = useMutation({
    mutationFn: (columnId: string) => deleteColumn(columnId),
    onMutate: async (columnId) => {
      await queryClient.cancelQueries({ queryKey: columnsQueryKey })
      const previousColumns = queryClient.getQueryData<BoardColumn[]>(columnsQueryKey)
      queryClient.setQueryData<BoardColumn[]>(columnsQueryKey, (currentColumns = []) =>
        currentColumns.filter((column) => column.id !== columnId),
      )
      return { previousColumns }
    },
    onError: (_error, _variables, context) => {
      if (context?.previousColumns) {
        queryClient.setQueryData(columnsQueryKey, context.previousColumns)
      }
    },
    onSuccess: () => {
      setColumnToDelete(null)
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: columnsQueryKey })
      invalidateTaskCaches(queryClient, selectedBoardId)
    },
  })

  function handleDragEnd(result: DropResult) {
    const { destination, draggableId, source } = result

    if (!destination || !selectedBoardId) {
      return
    }

    if (
      destination.droppableId === source.droppableId &&
      destination.index === source.index
    ) {
      return
    }

    const destinationColumn = visibleColumns.find(
      (column) => column.id === destination.droppableId,
    )
    if (!destinationColumn) {
      return
    }

    const status = statusForColumn(destinationColumn, visibleColumns)
    moveMutation.mutate({
      taskId: draggableId,
      columnId: destinationColumn.isFallback ? null : destinationColumn.id,
      ...(status ? { status } : {}),
    })
  }

  function handleMoveTaskByColumn(taskId: string, columnId: string) {
    const destinationColumn = visibleColumns.find((column) => column.id === columnId)
    if (!destinationColumn) {
      return
    }

    moveMutation.mutate({
      taskId,
      ...movePayloadForColumn(destinationColumn, visibleColumns),
    })
  }

  function handleEditTask(task: Task) {
    setTaskToEdit(task)
    setNewTaskStatus(undefined)
    setNewTaskColumnId(undefined)
    setEditDialogOpen(true)
  }

  function handleAddTask(columnId: string) {
    const column = visibleColumns.find((visibleColumn) => visibleColumn.id === columnId)
    if (!column) {
      return
    }

    setTaskToEdit(null)
    setNewTaskColumnId(column.isFallback ? undefined : column.id)
    setNewTaskStatus(statusForColumn(column, visibleColumns))
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
              tasks={localTasks
                .filter((task) => taskBelongsToColumn(task, column, visibleColumns))
                .map(taskResponseToKanbanTask)}
              selectedBoardId={selectedBoardId}
              columnOptions={visibleColumns}
              movePending={moveMutation.isPending}
              onEditTask={handleEditTask}
              onMoveTask={handleMoveTaskByColumn}
              onAddTask={handleAddTask}
              onUpdateColumn={(columnId, name, color) =>
                updateColumnMutation.mutate({ columnId, name, color })
              }
              onRequestDeleteColumn={(columnId, title) =>
                setColumnToDelete({ id: columnId, title })
              }
              updatePending={updateColumnMutation.isPending}
              disabled={Boolean(column.isFallback)}
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
                    className="flex-1 rounded-lg"
                    disabled={createColumnMutation.isPending}
                    onClick={handleCreateColumn}
                  >
                    {createColumnMutation.isPending ? "Creando..." : "Crear"}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    className="rounded-lg"
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
          <div aria-hidden="true" className="w-5 shrink-0 md:w-6" />
        </div>
      </DragDropContext>
      <NewTaskDialog
        open={editDialogOpen}
        onOpenChange={handleEditDialogOpenChange}
        selectedBoardId={selectedBoardId}
        taskToEdit={taskToEdit}
        initialStatus={newTaskStatus}
        initialColumnId={newTaskColumnId}
        onTaskSaved={(savedTask) => {
          setLocalTasks((currentTasks) => {
            const withoutDuplicate = currentTasks.filter((task) => task.id !== savedTask.id)
            return [savedTask, ...withoutDuplicate]
          })
        }}
      />
      <ConfirmDialog
        open={Boolean(columnToDelete)}
        onOpenChange={(open) => {
          if (!open) {
            setColumnToDelete(null)
          }
        }}
        title="Eliminar columna"
        description={`Se eliminará la columna "${columnToDelete?.title ?? ""}". Las tareas asociadas dejarán de mostrarse en esta vista hasta que se reasignen.`}
        confirmLabel="Eliminar columna"
        isPending={deleteColumnMutation.isPending}
        onConfirm={() => {
          if (columnToDelete) {
            deleteColumnMutation.mutate(columnToDelete.id)
          }
        }}
      />
    </main>
  )
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
