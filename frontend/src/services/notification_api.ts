import { apiRequest } from "@/services/api";

export interface AppNotification {
  id: string;
  userId: string;
  title: string;
  message: string;
  isRead: boolean;
  createdAt: string;
}

export async function getNotifications(): Promise<AppNotification[]> {
  return apiRequest<AppNotification[]>("/notifications");
}

export async function markNotificationAsRead(id: string): Promise<void> {
  await apiRequest<void>(`/notifications/${id}/read`, {
    method: "PATCH",
  });
}
