import { APP_VERSION } from "@/lib/app-version";
import { isTauriRuntime } from "@/lib/runtime";

type DiagnosticArea = "api" | "auth-ui" | "updater";

type DiagnosticEvent = {
  timestamp?: string;
  context: string;
  area: DiagnosticArea;
  isTauriRuntime?: boolean;
  appVersion?: string;
  [key: string]: unknown;
};

let writeQueue: Promise<void> = Promise.resolve();

export async function appendDesktopDiagnostic(event: DiagnosticEvent): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  const entry = {
    timestamp: new Date().toISOString(),
    isTauriRuntime: true,
    appVersion: APP_VERSION,
    ...sanitizeRecord(event),
  };
  const line = `${JSON.stringify(entry)}\n`;

  writeQueue = writeQueue
    .catch(() => undefined)
    .then(async () => {
      const [{ appDataDir, join }, { exists, writeTextFile }] = await Promise.all([
        import("@tauri-apps/api/path"),
        import("@tauri-apps/plugin-fs"),
      ]);

      const logPath = await join(await appDataDir(), "taskify-debug.log");
      const logExists = await exists(logPath);

      if (logExists) {
        await writeTextFile(logPath, line, { append: true });
        return;
      }

      await writeTextFile(logPath, line);
    });

  return writeQueue;
}

export function diagnosticErrorDetails(error: unknown) {
  if (error instanceof Error) {
    return sanitizeRecord({
      errorName: error.name,
      errorMessage: error.message,
      errorStack: error.stack,
      errorCause: getErrorCause(error),
      errorCode: getErrorCode(error),
    });
  }

  return sanitizeRecord({
    errorMessage: typeof error === "string" ? error : JSON.stringify(error),
  });
}

function getErrorCode(error: Error) {
  const candidate = error as Error & { code?: unknown };
  return typeof candidate.code === "string" || typeof candidate.code === "number"
    ? candidate.code
    : undefined;
}

function getErrorCause(error: Error) {
  const candidate = error as Error & { cause?: unknown };
  return candidate.cause;
}

function sanitizeRecord(value: Record<string, unknown>) {
  return sanitizeForJson(value) as Record<string, unknown>;
}

function sanitizeForJson(value: unknown): unknown {
  if (value === undefined) {
    return undefined;
  }

  if (value === null || typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return value;
  }

  if (value instanceof Error) {
    return {
      name: value.name,
      message: value.message,
      stack: value.stack,
      cause: sanitizeForJson(getErrorCause(value)),
      code: getErrorCode(value),
    };
  }

  if (Array.isArray(value)) {
    return value.map((item) => sanitizeForJson(item));
  }

  if (typeof value === "object") {
    const sanitizedEntries = Object.entries(value).flatMap(([key, nestedValue]) => {
      const sanitizedValue = sanitizeForJson(nestedValue);
      if (sanitizedValue === undefined) {
        return [];
      }
      return [[key, sanitizedValue] as const];
    });

    return Object.fromEntries(sanitizedEntries);
  }

  return String(value);
}
