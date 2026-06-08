import { apiRequest } from "@/services/api";

export type LoginCredentials = {
  email: string;
  password: string;
};

export type LoginResponse = {
  accessToken: string;
  refreshToken: string;
};

export type RegisterData = {
  email: string;
  password: string;
  firstName: string;
  lastName: string;
  birthDate: string;
};

export type RegisterResponse = {
  id: string;
  email: string;
};

export async function login(
  credentials: LoginCredentials,
): Promise<LoginResponse> {
  return apiRequest<LoginResponse>("/users/login", {
    method: "POST",
    body: JSON.stringify(credentials),
  });
}

export async function register(data: RegisterData): Promise<RegisterResponse> {
  return apiRequest<RegisterResponse>("/users/register", {
    method: "POST",
    body: JSON.stringify(data),
  });
}
