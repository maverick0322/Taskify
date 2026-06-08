import {
  DragDropContext,
  type DropResult,
} from "@hello-pangea/dnd"
import { useMutation, useQueryClient } from "@tanstack/react-query"

import { KanbanColumn } from "@/components/taskify/kanban-column"
import { Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  updateTaskStatus,
  type Task,
  type TaskPriority,
  type TaskStatus,
} from "@/services/taskService"

export type KanbanTaskPriority = "Alta" | "Media" | "Baja"

export interface KanbanTask {
  id: string
  title: string
  description?: string
  priority: KanbanTaskPriority
  dueDate: string
  tag?: string
  assignees?: { name: string; seed: string }[]
  comments?: number
  attachments?: number
}

const columns = [
  {
    id: "todo" as const,
    title: "Pendiente",
    accentColor: "bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300",
    dotColor: "bg-slate-400",
  },
  {
    id: "in_progress" as const,
    title: "En Progreso",
    accentColor: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/50 dark:text-indigo-300",
    dotColor: "bg-indigo-500",
  },
  {
    id: "done" as const,
    title: "Completado",
    accentColor: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300",
    dotColor: "bg-emerald-500",
  },
]

interface KanbanBoardProps {
  selectedBoardId?: string
  tasks: Task[]
}

interface UpdateTaskStatusVariables {
  taskId: string
  status: TaskStatus
}

interface UpdateTaskStatusContext {
  previousTasks?: Task[]
}

export function KanbanBoard({ selectedBoardId, tasks }: KanbanBoardProps) {
  const queryClient = useQueryClient()
  const tasksQueryKey = ["tasks", selectedBoardId]
  const mutation = useMutation<void, Error, UpdateTaskStatusVariables, UpdateTaskStatusContext>({
    mutationFn: updateTaskStatus,
    onMutate: async ({ taskId, status }) => {
      await queryClient.cancelQueries({ queryKey: tasksQueryKey })

      const previousTasks = queryClient.getQueryData<Task[]>(tasksQueryKey)

      queryClient.setQueryData<Task[]>(tasksQueryKey, (currentTasks = []) =>
        currentTasks.map((task) =>
          task.id === taskId ? { ...task, status } : task,
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
      queryClient.invalidateQueries({ queryKey: tasksQueryKey })
    },
  })

  function handleDragEnd(result: DropResult) {
    const { destination, draggableId, source } = result

    if (!destination || !selectedBoardId) {
      return
    }

    const nextStatus = destination.droppableId as TaskStatus
    const previousStatus = source.droppableId as TaskStatus

    if (nextStatus === previousStatus) {
      return
    }

    mutation.mutate({
      taskId: draggableId,
      status: nextStatus,
    })
  }

  return (
    <main
      className="flex-1 overflow-x-auto kanban-scroll bg-canvas"
      aria-label="Tablero Kanban"
    >
      <DragDropContext onDragEnd={handleDragEnd}>
        <div className="flex h-full gap-4 p-5 md:p-6">
          {columns.map((col) => (
            <KanbanColumn
              key={col.id}
              status={col.id}
              title={col.title}
              tasks={tasks
                .filter((task) => task.status === col.id)
                .map(taskResponseToKanbanTask)}
              accentColor={col.accentColor}
              dotColor={col.dotColor}
            />
          ))}

          {/* Add Column Button */}
          <div className="flex shrink-0 items-start pt-0.5">
            <Button
              variant="outline"
              className="h-12 w-44 gap-2 border-dashed border-border/80 text-muted-foreground hover:text-foreground hover:border-border hover:bg-column"
            >
              <Plus className="size-4" />
              Nueva columna
            </Button>
          </div>
        </div>
      </DragDropContext>
    </main>
  )
}

function taskResponseToKanbanTask(task: Task): KanbanTask {
  return {
    id: task.id,
    title: task.title,
    description: task.description || undefined,
    priority: priorityToKanbanPriority(task.priority),
    dueDate: task.dueDate || "Sin fecha",
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
