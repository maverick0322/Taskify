import { apiRequest } from "@/services/api"

export interface UserProfileResponse {
  id: string
  email: string
  firstName: string
  lastName: string
  avatarLocalPath?: string
  avatarUrl?: string
}

export async function getCurrentUserProfile(): Promise<UserProfileResponse> {
  return apiRequest<UserProfileResponse>("/users/me")
}

export async function updateUserAvatar(
  userId: string,
  avatarLocalPath: string,
): Promise<UserProfileResponse> {
  return apiRequest<UserProfileResponse>(`/users/${userId}/avatar`, {
    method: "PATCH",
    body: JSON.stringify({ avatarLocalPath }),
  })
}
