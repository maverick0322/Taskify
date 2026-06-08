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
  dueDate?: string;
}

export interface UpdateTaskStatusInput {
  taskId: string;
  status: TaskStatus;
}

export async function getTasks(): Promise<Task[]> {
  return apiRequest<Task[]>("/tasks");
}

export async function createTask(input: CreateTaskInput): Promise<Task> {
  return apiRequest<Task>("/tasks", {
    method: "POST",
    body: JSON.stringify({
      title: input.title,
      description: input.description,
      priority: input.priority,
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
