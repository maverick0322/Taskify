import { apiRequest } from "@/services/api";

export interface Board {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}

export interface BoardColumn {
  id: string;
  boardId: string;
  name: string;
  color: string;
  position: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateColumnInput {
  name: string;
  color: string;
  position: number;
}

export interface UpdateColumnInput {
  name: string;
  color: string;
}

export async function getBoards(): Promise<Board[]> {
  return apiRequest<Board[]>("/boards");
}

export async function createBoard(name: string): Promise<Board> {
  return apiRequest<Board>("/boards", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function deleteBoard(boardId: string): Promise<void> {
  await apiRequest<void>(`/boards/${boardId}`, {
    method: "DELETE",
  });
}

export async function getBoardColumns(boardId: string): Promise<BoardColumn[]> {
  return apiRequest<BoardColumn[]>(`/boards/${boardId}/columns`);
}

export async function createColumn(
  boardId: string,
  input: CreateColumnInput,
): Promise<BoardColumn> {
  return apiRequest<BoardColumn>(`/boards/${boardId}/columns`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateColumn(
  columnId: string,
  input: UpdateColumnInput,
): Promise<void> {
  await apiRequest<void>(`/columns/${columnId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}
