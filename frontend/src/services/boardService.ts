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
