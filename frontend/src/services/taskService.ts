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
  boardId: string;
  columnId?: string;
  tag?: string;
  assignees?: TaskAssignee[];
  comments?: number;
  attachments?: number;
}

export interface CreateTaskInput {
  title: string;
  description: string;
  priority: TaskPriority;
  boardId: string;
  dueDate?: string;
}

export interface UpdateTaskStatusInput {
  taskId: string;
  status: TaskStatus;
}

export async function getTasks(boardId?: string): Promise<Task[]> {
  const query = boardId ? `?board_id=${encodeURIComponent(boardId)}` : "";
  return apiRequest<Task[]>(`/tasks${query}`);
}

export async function createTask(input: CreateTaskInput): Promise<Task> {
  return apiRequest<Task>("/tasks", {
    method: "POST",
    body: JSON.stringify({
      title: input.title,
      description: input.description,
      priority: input.priority,
      boardId: input.boardId,
      dueDate: input.dueDate ?? "",
    }),
  });
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
