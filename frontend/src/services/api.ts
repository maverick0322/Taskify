import { useAuthStore } from "@/store/useAuthStore";

export const API_BASE_URL = "http://localhost:8080";

type ApiErrorResponse = {
  error?: string;
};

type TokenPairResponse = {
  accessToken?: string;
  refreshToken?: string;
};

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

let refreshTokenPromise: Promise<string> | null = null;

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  return requestWithAuth<T>(path, options, true);
}

async function requestWithAuth<T>(
  path: string,
  options: RequestInit,
  canRefresh: boolean,
): Promise<T> {
  const headers = new Headers(options.headers);
  const hasBody = options.body !== undefined && options.body !== null;

  if (hasBody && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const accessToken = localStorage.getItem("accessToken");
  if (accessToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  });

  if (response.status === 401 && canRefresh && shouldAttemptRefresh(path)) {
    const newAccessToken = await refreshAccessToken();
    const retryHeaders = new Headers(options.headers);

    if (hasBody && !retryHeaders.has("Content-Type")) {
      retryHeaders.set("Content-Type", "application/json");
    }

    retryHeaders.set("Authorization", `Bearer ${newAccessToken}`);

    return requestWithAuth<T>(
      path,
      {
        ...options,
        headers: retryHeaders,
      },
      false,
    );
  }

  if (!response.ok) {
    throw new ApiError(response.status, await errorMessageFromResponse(response));
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

function shouldAttemptRefresh(path: string): boolean {
  return !["/users/login", "/users/register", "/users/refresh"].includes(path);
}

async function refreshAccessToken(): Promise<string> {
  if (!refreshTokenPromise) {
    refreshTokenPromise = performRefresh()
      .catch(async (error) => {
        await useAuthStoreLogout();
        throw error;
      })
      .finally(() => {
        refreshTokenPromise = null;
      });
  }

  return refreshTokenPromise;
}

async function performRefresh(): Promise<string> {
  const refreshToken = localStorage.getItem("refreshToken");

  if (!refreshToken) {
    throw new ApiError(401, "missing refresh token");
  }

  const response = await fetch(`${API_BASE_URL}/users/refresh`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ refreshToken }),
  });

  if (!response.ok) {
    throw new ApiError(response.status, await errorMessageFromResponse(response));
  }

  const tokenPair = (await response.json()) as TokenPairResponse;

  if (!tokenPair.accessToken || !tokenPair.refreshToken) {
    throw new ApiError(401, "invalid refresh response");
  }

  localStorage.setItem("refreshToken", tokenPair.refreshToken);

  useAuthStore.getState().login(tokenPair.accessToken);

  return tokenPair.accessToken;
}

function useAuthStoreLogout() {
  useAuthStore.getState().logout();
}

async function errorMessageFromResponse(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as ApiErrorResponse;
    return payload.error || response.statusText;
  } catch {
    return response.statusText;
  }
}
