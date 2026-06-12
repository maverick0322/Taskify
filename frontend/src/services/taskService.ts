import { apiRequest } from "@/services/api";

export type TaskPriority = "low" | "medium" | "high";
export type TaskStatus = "todo" | "in_progress" | "done";

export interface TaskAssignee {
  name: string;
  seed: string;
}

export interface Task {
  id: string;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  dueDate: string;
  createdAt: string;
  updatedAt: string;
  boardId?: string | null;
  columnId?: string | null;
  tag?: string;
  assignees?: TaskAssignee[];
  comments?: number;
  attachments?: number;
}

export interface CreateTaskInput {
  title: string;
  description: string;
  priority: TaskPriority;
  boardId?: string | null;
  columnId?: string | null;
  dueDate?: string;
  status?: TaskStatus;
}

export interface UpdateTaskStatusInput {
  taskId: string;
  status: TaskStatus;
}

export type UpdateTaskInput = Partial<Task>;

export async function getTasks(boardId?: string): Promise<Task[]> {
  const query = boardId ? `?board_id=${encodeURIComponent(boardId)}` : "";
  return apiRequest<Task[]>(`/tasks${query}`);
}

export async function createTask(input: CreateTaskInput): Promise<Task> {
  const task = await apiRequest<Task>("/tasks", {
    method: "POST",
    body: JSON.stringify({
      title: input.title,
      description: input.description,
      priority: input.priority,
      ...(input.boardId ? { boardId: input.boardId } : {}),
      ...(input.columnId ? { columnId: input.columnId } : {}),
      dueDate: input.dueDate ?? "",
    }),
  });

  if (input.status && input.status !== task.status) {
    await updateTaskStatus({ taskId: task.id, status: input.status });
    return { ...task, status: input.status };
  }

  return task;
}

export async function updateTaskStatus({
  taskId,
  status,
}: UpdateTaskStatusInput): Promise<void> {
  await apiRequest<void>(`/tasks/${taskId}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export async function updateTask(
  taskId: string,
  data: UpdateTaskInput,
): Promise<void> {
  await apiRequest<void>(`/tasks/${taskId}`, {
    method: "PATCH",
    body: JSON.stringify({
      title: data.title ?? "",
      description: data.description ?? "",
      columnId: data.columnId ?? null,
      priority: data.priority ?? "medium",
      dueDate: data.dueDate ?? "",
    }),
  });
}

export async function moveTaskToColumn(
  taskId: string,
  columnId: string | null,
): Promise<void> {
  await apiRequest<void>(`/tasks/${taskId}/column`, {
    method: "PATCH",
    body: JSON.stringify({ columnId }),
  });
}

export async function deleteTask(taskId: string): Promise<void> {
  await apiRequest<void>(`/tasks/${taskId}`, {
    method: "DELETE",
  });
}
