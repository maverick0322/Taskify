import { APP_VERSION } from "@/lib/app-version";
import { isTauriRuntime } from "@/lib/runtime";

type DiagnosticArea = "api" | "auth-ui" | "updater";
type DiagnosticLogKind = "general" | "updater";

type DiagnosticEvent = {
  timestamp?: string;
  context: string;
  area: DiagnosticArea;
  isTauriRuntime?: boolean;
  appVersion?: string;
  [key: string]: unknown;
};

let writeQueue: Promise<void> = Promise.resolve();
const updaterWriteQueueByFile = new Map<string, Promise<void>>();
const UPDATER_LOG_MAX_BYTES = 2 * 1024 * 1024;
const UPDATER_LOG_MAX_LINES = 500;
let hasLoggedUpdaterTarget = false;

export async function appendDesktopDiagnostic(event: DiagnosticEvent): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  if (event.area === "updater") {
    return appendUpdaterDiagnostic(event);
  }

  writeQueue = writeQueue
    .catch(() => undefined)
    .then(async () => {
      const logPath = await resolveDiagnosticLogPath("general");
      await appendDiagnosticLine(logPath, event);
    });

  return writeQueue;
}

export async function appendUpdaterDiagnostic(event: DiagnosticEvent): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  const queueKey = "taskify-updater.log";
  const currentQueue = updaterWriteQueueByFile.get(queueKey) ?? Promise.resolve();
  const nextQueue = currentQueue
    .catch(() => undefined)
    .then(async () => {
      const logPath = await resolveDiagnosticLogPath("updater");
      await rotateUpdaterLogIfNeeded(logPath);
      if (!hasLoggedUpdaterTarget) {
        hasLoggedUpdaterTarget = true;
        await appendDiagnosticLine(logPath, {
          context: "updater_logger_target_resolved",
          area: "updater",
          resolvedLogPath: logPath,
        });
      }
      await appendDiagnosticLine(logPath, event);
    });

  updaterWriteQueueByFile.set(queueKey, nextQueue);
  await nextQueue;
}

export async function rotateUpdaterLogIfNeeded(logPath?: string): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  const [{ exists, readTextFile, stat, writeTextFile }] =
    await Promise.all([
      import("@tauri-apps/plugin-fs"),
    ]);

  const resolvedLogPath = logPath ?? (await resolveDiagnosticLogPath("updater"));
  const logExists = await exists(resolvedLogPath);
  if (!logExists) {
    return;
  }

  const metadata = await stat(resolvedLogPath);
  if ((metadata.size ?? 0) <= UPDATER_LOG_MAX_BYTES) {
    return;
  }

  const currentContents = await readTextFile(resolvedLogPath);
  const lines = currentContents
    .split(/\r?\n/)
    .filter((line) => line.trim().length > 0);
  const trimmedContents = `${lines.slice(-UPDATER_LOG_MAX_LINES).join("\n")}${lines.length > 0 ? "\n" : ""}`;

  await writeTextFile(resolvedLogPath, trimmedContents);
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

export async function resolveDiagnosticLogPath(kind: DiagnosticLogKind) {
  const [{ appDataDir, join }] = await Promise.all([
    import("@tauri-apps/api/path"),
  ]);

  return join(
    await appDataDir(),
    kind === "updater" ? "taskify-updater.log" : "taskify-debug.log",
  );
}

async function appendDiagnosticLine(logPath: string, event: DiagnosticEvent) {
  const entry = {
    timestamp: new Date().toISOString(),
    isTauriRuntime: true,
    appVersion: APP_VERSION,
    ...sanitizeRecord(event),
  };
  const line = `${JSON.stringify(entry)}\n`;

  const [{ exists, writeTextFile }] = await Promise.all([
    import("@tauri-apps/plugin-fs"),
  ]);
  const logExists = await exists(logPath);

  if (logExists) {
    await writeTextFile(logPath, line, { append: true });
    return;
  }

  await writeTextFile(logPath, line);
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
