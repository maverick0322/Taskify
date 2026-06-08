export const API_BASE_URL = "http://localhost:8080";

type ApiErrorResponse = {
  error?: string;
};

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
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

  if (!response.ok) {
    throw new ApiError(response.status, await errorMessageFromResponse(response));
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

async function errorMessageFromResponse(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as ApiErrorResponse;
    return payload.error || response.statusText;
  } catch {
    return response.statusText;
  }
}
