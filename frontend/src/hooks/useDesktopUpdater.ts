import { useCallback, useEffect, useMemo } from "react";
import { invoke } from "@tauri-apps/api/core";
import { check } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";

import { useToast } from "@/components/ui/toast-provider";
import { isTauriRuntime } from "@/lib/runtime";
import { useUpdaterStore } from "@/store/useUpdaterStore";

let checkPromise: Promise<boolean> | null = null;
let installPromise: Promise<void> | null = null;

type UseDesktopUpdaterOptions = {
  enableStartupCheck?: boolean;
  isReady?: boolean;
};

type CheckForUpdatesOptions = {
  silentIfNoUpdate?: boolean;
};

const UPDATER_UP_TO_DATE_MESSAGE =
  "¡Todo al día! Estás usando la versión más reciente de Taskify.";
const UPDATER_DEBUG_LOG_FILE = "updater-debug.log";
const SIDECAR_SHUTDOWN_DELAY_MS = 250;

function serializeUpdaterError(error: unknown) {
  if (error instanceof Error) {
    const errorWithCause = error as Error & { cause?: unknown };

    return {
      name: error.name,
      message: error.message,
      stack: error.stack ?? null,
      cause:
        errorWithCause.cause instanceof Error
          ? {
              name: errorWithCause.cause.name,
              message: errorWithCause.cause.message,
              stack: errorWithCause.cause.stack ?? null,
            }
          : errorWithCause.cause ?? null,
    };
  }

  return {
    message: typeof error === "string" ? error : JSON.stringify(error, null, 2),
    stack: null,
    cause: null,
  };
}

async function appendUpdaterDebugLog(context: string, payload: Record<string, unknown>) {
  const timestamp = new Date().toISOString();
  const logEntry = `${JSON.stringify({ timestamp, context, ...payload })}\n`;

  console.error(`[updater][${context}]`, payload);

  if (!isTauriRuntime()) {
    return;
  }

  try {
    const [{ appDataDir, join }, { writeTextFile }] = await Promise.all([
      import("@tauri-apps/api/path"),
      import("@tauri-apps/plugin-fs"),
    ]);

    const logPath = await join(await appDataDir(), UPDATER_DEBUG_LOG_FILE);
    await writeTextFile(logPath, logEntry, {
      append: true,
      create: true,
    });
  } catch (loggingError) {
    console.error("[updater][logger_failed]", loggingError);
  }
}

function normalizeUpdaterError(error: unknown) {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }

  return "No pudimos completar la actualización en este momento.";
}

function isBenignMetadataError(error: unknown) {
  if (!(error instanceof Error)) {
    return false;
  }

  const message = error.message.toLowerCase();
  return (
    message.includes("latest.json") ||
    message.includes("404") ||
    message.includes("not found") ||
    message.includes("failed to deserialize") ||
    message.includes("could not parse") ||
    message.includes("unable to parse")
  );
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

export function useDesktopUpdater(options?: UseDesktopUpdaterOptions) {
  const enableStartupCheck = options?.enableStartupCheck ?? false;
  const isReady = options?.isReady ?? true;
  const toast = useToast();
  const hasCheckedOnStartup = useUpdaterStore((state) => state.hasCheckedOnStartup);
  const isDialogOpen = useUpdaterStore((state) => state.isDialogOpen);
  const stage = useUpdaterStore((state) => state.stage);
  const availableUpdate = useUpdaterStore((state) => state.availableUpdate);
  const downloadedBytes = useUpdaterStore((state) => state.downloadedBytes);
  const contentLength = useUpdaterStore((state) => state.contentLength);
  const markStartupChecked = useUpdaterStore((state) => state.markStartupChecked);
  const setStage = useUpdaterStore((state) => state.setStage);
  const setDialogOpen = useUpdaterStore((state) => state.setDialogOpen);
  const setAvailableUpdate = useUpdaterStore((state) => state.setAvailableUpdate);
  const setDownloadProgress = useUpdaterStore((state) => state.setDownloadProgress);
  const resetProgress = useUpdaterStore((state) => state.resetProgress);

  const canUseUpdater = isTauriRuntime() && import.meta.env.PROD;

  const checkForUpdates = useCallback(
    async (checkOptions?: CheckForUpdatesOptions) => {
      if (!canUseUpdater) {
        return false;
      }

      if (checkPromise) {
        return checkPromise;
      }

      const silentIfNoUpdate = checkOptions?.silentIfNoUpdate ?? false;

      checkPromise = (async () => {
        setStage("checking");
        resetProgress();

        try {
          const update = await check();
          if (!update) {
            setStage("idle");
            if (!silentIfNoUpdate) {
              toast.success(UPDATER_UP_TO_DATE_MESSAGE);
            }
            return false;
          }

          setAvailableUpdate({
            currentVersion: update.currentVersion,
            version: update.version,
            body: update.body,
            date: update.date,
            rawJson: update.rawJson,
            handle: update,
          });
          return true;
        } catch (error) {
          setStage("idle");
          await appendUpdaterDebugLog("check_failed", {
            canUseUpdater,
            production: import.meta.env.PROD,
            ...serializeUpdaterError(error),
          });
          if (isBenignMetadataError(error)) {
            if (!silentIfNoUpdate) {
              toast.success(UPDATER_UP_TO_DATE_MESSAGE);
            }
            return false;
          }
          toast.error(normalizeUpdaterError(error));
          return false;
        } finally {
          checkPromise = null;
        }
      })();

      return checkPromise;
    },
    [canUseUpdater, resetProgress, setAvailableUpdate, setStage, toast],
  );

  const installUpdate = useCallback(async () => {
    if (!canUseUpdater || !availableUpdate?.handle) {
      return;
    }

    if (installPromise) {
      return installPromise;
    }

    installPromise = (async () => {
      try {
        let nextContentLength: number | null = null;
        const progressEvents: Array<Record<string, unknown>> = [];

        setStage("downloading");
        resetProgress();

        await appendUpdaterDebugLog("sidecar_shutdown_started", {
          currentVersion: availableUpdate.currentVersion,
          targetVersion: availableUpdate.version,
        });

        let sidecarKilled = false;
        try {
          sidecarKilled = await invoke<boolean>("shutdown_backend_sidecar");
        } catch (error) {
          setStage("available");
          await appendUpdaterDebugLog("sidecar_shutdown_failed", {
            currentVersion: availableUpdate.currentVersion,
            targetVersion: availableUpdate.version,
            ...serializeUpdaterError(error),
          });
          toast.error("No pudimos cerrar el motor local antes de actualizar.");
          return;
        }

        await appendUpdaterDebugLog("sidecar_shutdown_completed", {
          currentVersion: availableUpdate.currentVersion,
          targetVersion: availableUpdate.version,
          result: sidecarKilled ? "killed" : "already_stopped",
          delayMs: SIDECAR_SHUTDOWN_DELAY_MS,
        });

        await delay(SIDECAR_SHUTDOWN_DELAY_MS);

        await availableUpdate.handle.downloadAndInstall((event) => {
          if (event.event === "Started") {
            nextContentLength = event.data.contentLength ?? null;
            progressEvents.push({
              event: event.event,
              contentLength: event.data.contentLength ?? null,
            });
            setDownloadProgress(0, nextContentLength);
            return;
          }

          if (event.event === "Progress") {
            progressEvents.push({
              event: event.event,
              chunkLength: event.data.chunkLength,
              downloadedBytes: useUpdaterStore.getState().downloadedBytes + event.data.chunkLength,
            });
            setDownloadProgress(
              useUpdaterStore.getState().downloadedBytes + event.data.chunkLength,
              nextContentLength,
            );
            return;
          }

          if (event.event === "Finished") {
            progressEvents.push({
              event: event.event,
              downloadedBytes: useUpdaterStore.getState().downloadedBytes,
              contentLength: nextContentLength,
            });
            setStage("installing");
          }
        });

        await appendUpdaterDebugLog("install_succeeded", {
          currentVersion: availableUpdate.currentVersion,
          targetVersion: availableUpdate.version,
          contentLength: nextContentLength,
          downloadedBytes: useUpdaterStore.getState().downloadedBytes,
          progressEvents,
        });
        toast.success("Actualización instalada. Reiniciando Taskify...");
        setDialogOpen(false);
        setAvailableUpdate(null);
        resetProgress();
        await relaunch();
      } catch (error) {
        setStage("available");
        await appendUpdaterDebugLog("install_failed", {
          currentVersion: availableUpdate.currentVersion,
          targetVersion: availableUpdate.version,
          downloadedBytes: useUpdaterStore.getState().downloadedBytes,
          contentLength: useUpdaterStore.getState().contentLength,
          stage: useUpdaterStore.getState().stage,
          ...serializeUpdaterError(error),
        });
        toast.error(normalizeUpdaterError(error));
      } finally {
        installPromise = null;
      }
    })();

    return installPromise;
  }, [
    availableUpdate,
    canUseUpdater,
    resetProgress,
    setAvailableUpdate,
    setDialogOpen,
    setDownloadProgress,
    setStage,
    toast,
  ]);

  const dismissUpdate = useCallback(() => {
    if (stage === "downloading" || stage === "installing") {
      return;
    }

    setDialogOpen(false);
  }, [setDialogOpen, stage]);

  useEffect(() => {
    if (!enableStartupCheck || !isReady || hasCheckedOnStartup) {
      return;
    }

    markStartupChecked();
    void checkForUpdates({ silentIfNoUpdate: true });
  }, [
    checkForUpdates,
    enableStartupCheck,
    hasCheckedOnStartup,
    isReady,
    markStartupChecked,
  ]);

  const progressPercent = useMemo(() => {
    if (!contentLength || contentLength <= 0) {
      return null;
    }

    return Math.max(
      0,
      Math.min(100, Math.round((downloadedBytes / contentLength) * 100)),
    );
  }, [contentLength, downloadedBytes]);

  return {
    canUseUpdater,
    isDialogOpen,
    stage,
    availableUpdate,
    downloadedBytes,
    contentLength,
    progressPercent,
    checkForUpdates,
    dismissUpdate,
    installUpdate,
  };
}
