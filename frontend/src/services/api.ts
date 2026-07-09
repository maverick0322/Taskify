import { useAuthStore } from "@/store/useAuthStore";
import { isTauriRuntime } from "@/lib/runtime";
import {
  appendDesktopDiagnostic,
  diagnosticErrorDetails,
} from "@/services/desktopDiagnostics";
import {
  clearStoredSession,
  loadStoredSession,
  persistSession,
} from "@/services/secureSession";
import { jwtDecode } from "jwt-decode";

type ApiErrorResponse = {
  error?: string;
};

type TokenPairResponse = {
  accessToken?: string;
  refreshToken?: string;
};

type ApiRequestOptions = RequestInit & {
  timeoutMs?: number;
  timeoutMessage?: string;
};

type RequestDiagnosticMetadata = {
  apiBaseUrl: string;
  path: string;
};

type AccessTokenClaims = {
  exp?: number;
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
  options: ApiRequestOptions = {},
): Promise<T> {
  return requestWithAuth<T>(path, options, true);
}

async function requestWithAuth<T>(
  path: string,
  options: ApiRequestOptions,
  canRefresh: boolean,
): Promise<T> {
  const apiBaseUrl = resolveApiBaseUrl();
  const requestUrl = `${apiBaseUrl}${path}`;
  const headers = new Headers(options.headers);
  const hasBody = options.body !== undefined && options.body !== null;

  if (hasBody && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const accessToken = useAuthStore.getState().accessToken;
  if (accessToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await safeFetch(requestUrl, {
    ...options,
    headers,
  }, {
    apiBaseUrl,
    path,
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
    void appendDesktopDiagnostic({
      context: "api_request_http_failed",
      area: "api",
      apiBaseUrl,
      path,
      requestUrl,
      method: options.method ?? "GET",
      port: safeUrlPort(requestUrl),
      timeoutMs: options.timeoutMs ?? null,
      hasAuthorization: headers.has("Authorization"),
      responseStatus: response.status,
      responseStatusText: response.statusText,
      navigatorOnline:
        typeof navigator !== "undefined" ? navigator.onLine : undefined,
    });
    if (typeof window !== "undefined") {
      console.error("[API] Request failed", {
        path,
        requestUrl,
        method: options.method ?? "GET",
        status: response.status,
      });
    }
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

export async function ensureValidAccessToken(): Promise<string> {
  const storeAccessToken = useAuthStore.getState().accessToken;
  if (storeAccessToken && !isTokenExpired(storeAccessToken)) {
    return storeAccessToken;
  }

  const storedSession = await loadStoredSession();
  if (storedSession?.accessToken && !isTokenExpired(storedSession.accessToken)) {
    return storedSession.accessToken;
  }

  if (!storedSession?.refreshToken) {
    throw new ApiError(
      401,
      "Tu sesión expiró. Inicia sesión nuevamente.",
      undefined,
      "missing refresh token",
    );
  }

  return refreshAccessToken();
}

export async function restoreOrRefreshSession(): Promise<string | null> {
  const storedSession = await loadStoredSession().catch(() => null);
  if (!storedSession?.accessToken || !storedSession.refreshToken) {
    return null;
  }

  if (!isTokenExpired(storedSession.accessToken)) {
    return storedSession.accessToken;
  }

  return ensureValidAccessToken();
}

async function performRefresh(): Promise<string> {
  const apiBaseUrl = resolveApiBaseUrl();
  const storedSession = await loadStoredSession();
  const refreshToken = storedSession?.refreshToken;

  if (!refreshToken) {
    throw new ApiError(
      401,
      "Tu sesión expiró. Inicia sesión nuevamente.",
      undefined,
      "missing refresh token",
    );
  }

  const response = await safeFetch(`${apiBaseUrl}/users/refresh`, {
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
      "Tu sesión expiró. Inicia sesión nuevamente.",
      undefined,
      "invalid refresh response",
    );
  }

  await persistSession({
    accessToken: tokenPair.accessToken,
    refreshToken: tokenPair.refreshToken,
    remoteAccessToken: storedSession?.remoteAccessToken,
    remoteRefreshToken: storedSession?.remoteRefreshToken,
  });
  useAuthStore.getState().login(tokenPair.accessToken);

  return tokenPair.accessToken;
}

async function useAuthStoreLogout() {
  await clearStoredSession().catch(() => undefined);
  useAuthStore
    .getState()
    .logout({ skipStorageClear: true, purgeDesktopData: true });
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

async function safeFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
  diagnosticMetadata?: RequestDiagnosticMetadata,
) {
  const timeoutOptions = init as ApiRequestOptions | undefined;
  const timeoutMs = timeoutOptions?.timeoutMs ?? 0;
  const timeoutMessage = timeoutOptions?.timeoutMessage;
  const controller = new AbortController();
  const signal = init?.signal;
  let timeoutID: ReturnType<typeof setTimeout> | undefined;

  const abortListener = () => controller.abort();
  if (signal) {
    if (signal.aborted) {
      controller.abort();
    } else {
      signal.addEventListener("abort", abortListener, { once: true });
    }
  }
  if (timeoutMs > 0) {
    timeoutID = setTimeout(() => controller.abort(), timeoutMs);
  }

  try {
    void appendDesktopDiagnostic({
      context: "api_request_started",
      area: "api",
      apiBaseUrl: diagnosticMetadata?.apiBaseUrl,
      path: diagnosticMetadata?.path,
      requestUrl: String(input),
      method: init?.method ?? "GET",
      port: safeUrlPort(input),
      timeoutMs: timeoutMs || null,
      hasAuthorization: new Headers(init?.headers).has("Authorization"),
      navigatorOnline:
        typeof navigator !== "undefined" ? navigator.onLine : undefined,
    });
    return await fetch(input, {
      ...init,
      signal: controller.signal,
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError" && timeoutMs > 0) {
      void appendDesktopDiagnostic({
        context: "api_request_network_failed",
        area: "api",
        apiBaseUrl: diagnosticMetadata?.apiBaseUrl,
        path: diagnosticMetadata?.path,
        requestUrl: String(input),
        method: init?.method ?? "GET",
        port: safeUrlPort(input),
        timeoutMs,
        hasAuthorization: new Headers(init?.headers).has("Authorization"),
        navigatorOnline:
          typeof navigator !== "undefined" ? navigator.onLine : undefined,
        technicalMessage: "request timeout",
        ...diagnosticErrorDetails(error),
      });
      throw new ApiError(
        408,
        timeoutMessage ?? "La sincronización inicial está tardando demasiado. Inténtalo de nuevo.",
        undefined,
        "request timeout",
      );
    }
    if (typeof window !== "undefined") {
      console.error("[API] Network request failed", {
        input: String(input),
        method: init?.method ?? "GET",
        error,
      });
    }
    void appendDesktopDiagnostic({
      context: "api_request_network_failed",
      area: "api",
      apiBaseUrl: diagnosticMetadata?.apiBaseUrl,
      path: diagnosticMetadata?.path,
      requestUrl: String(input),
      method: init?.method ?? "GET",
      port: safeUrlPort(input),
      timeoutMs: timeoutMs || null,
      hasAuthorization: new Headers(init?.headers).has("Authorization"),
      navigatorOnline:
        typeof navigator !== "undefined" ? navigator.onLine : undefined,
      ...diagnosticErrorDetails(error),
    });
    throw networkApiError(error);
  } finally {
    if (timeoutID) {
      clearTimeout(timeoutID);
    }
    signal?.removeEventListener("abort", abortListener);
  }
}

function safeUrlPort(input: RequestInfo | URL) {
  try {
    const url = typeof input === "string" ? new URL(input) : input instanceof URL ? input : new URL(input.url);
    if (url.port) {
      return url.port;
    }

    if (url.protocol === "https:") {
      return "443";
    }

    if (url.protocol === "http:") {
      return "80";
    }

    return null;
  } catch {
    return null;
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

function apiErrorFromResponse(
  status: number,
  backendMessage: string,
): ApiError {
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
    return "Ocurrió un error inesperado en el servidor.";
  }

  const normalizedMessage = normalizeBackendMessage(backendMessage);
  const translatedMessage = translatedBackendMessages[normalizedMessage];
  if (translatedMessage) {
    return translatedMessage;
  }

  if (status === 401) {
    return "Tu sesión expiró. Inicia sesión nuevamente.";
  }

  if (status === 403) {
    return "No tienes permiso para realizar esta acción.";
  }

  if (status === 404) {
    return "No encontramos el recurso solicitado.";
  }

  if (status === 409) {
    return "La información entra en conflicto con un registro existente.";
  }

  if (status >= 400) {
    return "No pudimos completar la solicitud. Revisa la información e inténtalo de nuevo.";
  }

  return "No pudimos completar la solicitud. Intentalo de nuevo.";
}

function normalizeBackendMessage(message?: string): string {
  return message?.trim().toLowerCase() ?? "";
}

function isTokenExpired(token: string) {
  try {
    const claims = jwtDecode<AccessTokenClaims>(token);
    if (!claims.exp) {
      return false;
    }
    return claims.exp * 1000 <= Date.now();
  } catch {
    return true;
  }
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
  "invalid credentials":
    "Credenciales inválidas. Revisa tu correo y contraseña.",
  unauthorized: "Tu sesión expiró. Inicia sesión nuevamente.",
  "invalid refresh token": "Tu sesión expiró. Inicia sesión nuevamente.",
  "missing refresh token": "Tu sesión expiró. Inicia sesión nuevamente.",
  "invalid refresh response": "Tu sesión expiró. Inicia sesión nuevamente.",
  "invalid request body":
    "La información enviada no es válida. Revisa los campos e inténtalo de nuevo.",
  "invalid birth date": "La fecha de nacimiento no es válida.",
  "invalid due date": "La fecha de entrega no es válida.",
  "invalid user data": "Revisa tu información personal e inténtalo de nuevo.",
  "name is required": "El nombre es obligatorio.",
  "cloud sync is not configured":
    "La sincronización en la nube no está configurada.",
  "sync failed": "No pudimos completar la sincronización.",
  "sqlite checkpoint failed":
    "No pudimos preparar la base de datos para el respaldo.",
  "user already exists": "Ya existe una cuenta con ese correo.",
  "invalid board data": "Revisa los datos del tablero e inténtalo de nuevo.",
  "board not found": "No encontramos ese tablero.",
  "column not found": "No encontramos esa columna.",
  "invalid column data": "Revisa los datos de la columna e inténtalo de nuevo.",
  "invalid task data": "Revisa los datos de la tarea e inténtalo de nuevo.",
  "task not found": "No encontramos esa tarea.",
  "invalid transaction data":
    "Revisa los datos de la transacción e inténtalo de nuevo.",
  "transaction not found": "No encontramos esa transacción.",
  "financial account not found": "No encontramos ese método de pago.",
  "invalid financial account data":
    "Revisa los datos de la cuenta financiera e inténtalo de nuevo.",
  "insufficient funds": "El saldo disponible no alcanza para este egreso.",
  "credit limit exceeded":
    "La compra supera el límite disponible de la tarjeta.",
  "invalid credit card data":
    "Revisa los datos de la tarjeta e inténtalo de nuevo.",
  "credit card not found": "No encontramos esa tarjeta.",
  "internal server error": "Ocurrió un error inesperado en el servidor.",
};

export function resolveApiBaseUrl() {
  if (isTauriRuntime()) {
    return "http://localhost:8080";
  }

  const viteApiUrl = import.meta.env.VITE_API_URL?.trim();
  if (viteApiUrl) {
    return viteApiUrl.replace(/\/$/, "");
  }

  if (import.meta.env.DEV) {
    return "http://localhost:8080";
  }

  throw new Error("Missing VITE_API_URL for the web/PWA runtime.");
}
