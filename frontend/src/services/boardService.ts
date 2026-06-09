import { apiRequest } from "@/services/api";

export interface Board {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
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
