import type { BoardColumn } from "@/services/boardService"
import type { Task, TaskStatus } from "@/services/taskService"
import type { ColumnColor } from "@/components/taskify/kanban-column"

export const DEFAULT_KANBAN_COLUMNS: Array<{
  name: string
  color: ColumnColor
  status: TaskStatus
}> = [
  { name: "Pendiente", color: "slate", status: "todo" },
  { name: "En Progreso", color: "indigo", status: "in_progress" },
  { name: "Completado", color: "emerald", status: "done" },
]

export interface DisplayColumn extends BoardColumn {
  isFallback?: boolean
  fallbackStatus?: TaskStatus
}

export function buildVisibleColumns(
  persistedColumns: BoardColumn[],
  boardId: string,
): DisplayColumn[] {
  if (persistedColumns.length > 0) {
    return persistedColumns
  }

  return DEFAULT_KANBAN_COLUMNS.map((column, index) => ({
    id: `fallback-${column.status}`,
    boardId,
    name: column.name,
    color: column.color,
    position: index,
    createdAt: "",
    updatedAt: "",
    isFallback: true,
    fallbackStatus: column.status,
  }))
}

export function statusForColumn(
  column: DisplayColumn,
  columns: DisplayColumn[],
): TaskStatus | undefined {
  if (column.fallbackStatus) {
    return column.fallbackStatus
  }

  const columnIndex = columns.findIndex((visibleColumn) => visibleColumn.id === column.id)
  return DEFAULT_KANBAN_COLUMNS[columnIndex]?.status
}

export function taskBelongsToColumn(
  task: Task,
  column: DisplayColumn,
  columns: DisplayColumn[],
) {
  if (task.columnId) {
    return task.columnId === column.id
  }

  return statusForColumn(column, columns) === task.status
}

export function columnIDForTask(task: Task, columns: DisplayColumn[]) {
  if (task.columnId && columns.some((column) => column.id === task.columnId)) {
    return task.columnId
  }

  return (
    columns.find((column) => statusForColumn(column, columns) === task.status)?.id ??
    columns[0]?.id ??
    ""
  )
}

export function movePayloadForColumn(column: DisplayColumn, columns: DisplayColumn[]) {
  return {
    columnId: column.isFallback ? null : column.id,
    status: statusForColumn(column, columns),
  }
}
