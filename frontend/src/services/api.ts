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
  readonly backendMessage?: string;
  readonly technicalMessage?: string;

  constructor(
    status: number,
    message: string,
    backendMessage?: string,
    technicalMessage?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.backendMessage = backendMessage;
    this.technicalMessage = technicalMessage;
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

  const response = await safeFetch(`${API_BASE_URL}${path}`, {
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
    throw apiErrorFromResponse(
      response.status,
      await errorMessageFromResponse(response),
    );
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
    throw new ApiError(
      401,
      "Tu sesion expiro. Inicia sesion nuevamente.",
      undefined,
      "missing refresh token",
    );
  }

  const response = await safeFetch(`${API_BASE_URL}/users/refresh`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ refreshToken }),
  });

  if (!response.ok) {
    throw apiErrorFromResponse(
      response.status,
      await errorMessageFromResponse(response),
    );
  }

  const tokenPair = (await response.json()) as TokenPairResponse;

  if (!tokenPair.accessToken || !tokenPair.refreshToken) {
    throw new ApiError(
      401,
      "Tu sesion expiro. Inicia sesion nuevamente.",
      undefined,
      "invalid refresh response",
    );
  }

  localStorage.setItem("refreshToken", tokenPair.refreshToken);

  useAuthStore.getState().login(tokenPair.accessToken);

  return tokenPair.accessToken;
}

function useAuthStoreLogout() {
  useAuthStore.getState().logout();
}

export function getFriendlyErrorMessage(
  error: unknown,
  fallback = "No pudimos completar la solicitud. Intentalo de nuevo.",
): string {
  return normalizeApiError(error, fallback).message;
}

export function normalizeApiError(
  error: unknown,
  fallback = "No pudimos completar la solicitud. Intentalo de nuevo.",
): ApiError {
  if (error instanceof ApiError) {
    return error;
  }

  if (isNetworkError(error)) {
    return networkApiError(error);
  }

  if (error instanceof Error) {
    return new ApiError(0, fallback, undefined, error.message);
  }

  return new ApiError(0, fallback);
}

async function errorMessageFromResponse(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as ApiErrorResponse;
    return payload.error || response.statusText;
  } catch {
    return response.statusText;
  }
}

async function safeFetch(input: RequestInfo | URL, init?: RequestInit) {
  try {
    return await fetch(input, init);
  } catch (error) {
    throw networkApiError(error);
  }
}

function networkApiError(error: unknown): ApiError {
  return new ApiError(
    0,
    "Error de conexion: El servidor interno no responde.",
    undefined,
    error instanceof Error ? error.message : String(error),
  );
}

function apiErrorFromResponse(status: number, backendMessage: string): ApiError {
  return new ApiError(
    status,
    friendlyMessageFromStatus(status, backendMessage),
    backendMessage,
  );
}

function friendlyMessageFromStatus(
  status: number,
  backendMessage?: string,
): string {
  if (status >= 500) {
    return "Ocurrio un error inesperado en el servidor.";
  }

  const normalizedMessage = normalizeBackendMessage(backendMessage);
  const translatedMessage = translatedBackendMessages[normalizedMessage];
  if (translatedMessage) {
    return translatedMessage;
  }

  if (status === 401) {
    return "Tu sesion expiro. Inicia sesion nuevamente.";
  }

  if (status === 403) {
    return "No tienes permiso para realizar esta accion.";
  }

  if (status === 404) {
    return "No encontramos el recurso solicitado.";
  }

  if (status === 409) {
    return "La informacion entra en conflicto con un registro existente.";
  }

  if (status >= 400) {
    return "No pudimos completar la solicitud. Revisa la informacion e intentalo de nuevo.";
  }

  return "No pudimos completar la solicitud. Intentalo de nuevo.";
}

function normalizeBackendMessage(message?: string): string {
  return message?.trim().toLowerCase() ?? "";
}

function isNetworkError(error: unknown): boolean {
  if (!(error instanceof Error)) {
    return false;
  }

  const message = error.message.toLowerCase();
  return (
    error.name === "TypeError" ||
    message.includes("failed to fetch") ||
    message.includes("networkerror") ||
    message.includes("network error")
  );
}

const translatedBackendMessages: Record<string, string> = {
  "invalid credentials": "Credenciales invalidas. Revisa tu correo y contrasena.",
  unauthorized: "Tu sesion expiro. Inicia sesion nuevamente.",
  "invalid refresh token": "Tu sesion expiro. Inicia sesion nuevamente.",
  "missing refresh token": "Tu sesion expiro. Inicia sesion nuevamente.",
  "invalid refresh response": "Tu sesion expiro. Inicia sesion nuevamente.",
  "invalid request body":
    "La informacion enviada no es valida. Revisa los campos e intentalo de nuevo.",
  "invalid birth date": "La fecha de nacimiento no es valida.",
  "invalid due date": "La fecha de entrega no es valida.",
  "invalid user data": "Revisa tu informacion personal e intentalo de nuevo.",
  "user already exists": "Ya existe una cuenta con ese correo.",
  "invalid board data": "Revisa los datos del tablero e intentalo de nuevo.",
  "board not found": "No encontramos ese tablero.",
  "column not found": "No encontramos esa columna.",
  "invalid column data": "Revisa los datos de la columna e intentalo de nuevo.",
  "invalid task data": "Revisa los datos de la tarea e intentalo de nuevo.",
  "task not found": "No encontramos esa tarea.",
  "invalid transaction data":
    "Revisa los datos de la transaccion e intentalo de nuevo.",
  "transaction not found": "No encontramos esa transaccion.",
  "invalid credit card data":
    "Revisa los datos de la tarjeta e intentalo de nuevo.",
  "credit card not found": "No encontramos esa tarjeta.",
  "internal server error": "Ocurrio un error inesperado en el servidor.",
};
