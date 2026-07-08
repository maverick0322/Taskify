import { useCallback, useEffect, useMemo } from "react";
import { invoke } from "@tauri-apps/api/core";
import { check } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";

import { useToast } from "@/components/ui/toast-provider";
import {
  appendUpdaterDiagnostic,
  diagnosticErrorDetails,
} from "@/services/desktopDiagnostics";
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
const SIDECAR_SHUTDOWN_DELAY_MS = 250;

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
        void appendUpdaterDiagnostic({
          context: "updater_check_started",
          area: "updater",
          stage: "checking",
        });

        try {
          const update = await check();
          if (!update) {
            void appendUpdaterDiagnostic({
              context: "updater_no_update",
              area: "updater",
              stage: "idle",
            });
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
          void appendUpdaterDiagnostic({
            context: "updater_update_available",
            area: "updater",
            stage: "available",
            currentVersion: update.currentVersion,
            targetVersion: update.version,
          });
          return true;
        } catch (error) {
          setStage("idle");
          void appendUpdaterDiagnostic({
            context: "updater_check_failed",
            area: "updater",
            stage: "idle",
            ...diagnosticErrorDetails(error),
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

        setStage("downloading");
        resetProgress();
        void appendUpdaterDiagnostic({
          context: "sidecar_shutdown_started",
          area: "updater",
          stage: "downloading",
          currentVersion: availableUpdate.currentVersion,
          targetVersion: availableUpdate.version,
        });

        try {
          const sidecarShutdownResult =
            await invoke<boolean>("shutdown_backend_sidecar");
          void appendUpdaterDiagnostic({
            context: "sidecar_shutdown_completed",
            area: "updater",
            stage: "downloading",
            currentVersion: availableUpdate.currentVersion,
            targetVersion: availableUpdate.version,
            sidecarShutdownResult,
          });
        } catch (error) {
          void appendUpdaterDiagnostic({
            context: "sidecar_shutdown_failed",
            area: "updater",
            stage: "available",
            currentVersion: availableUpdate.currentVersion,
            targetVersion: availableUpdate.version,
            ...diagnosticErrorDetails(error),
          });
          setStage("available");
          toast.error("No pudimos cerrar el motor local antes de actualizar.");
          return;
        }

        await delay(SIDECAR_SHUTDOWN_DELAY_MS);
        void appendUpdaterDiagnostic({
          context: "updater_download_started",
          area: "updater",
          stage: "downloading",
          currentVersion: availableUpdate.currentVersion,
          targetVersion: availableUpdate.version,
        });

        await availableUpdate.handle.downloadAndInstall((event) => {
          if (event.event === "Started") {
            nextContentLength = event.data.contentLength ?? null;
            setDownloadProgress(0, nextContentLength);
            void appendUpdaterDiagnostic({
              context: "updater_download_progress",
              area: "updater",
              stage: "downloading",
              currentVersion: availableUpdate.currentVersion,
              targetVersion: availableUpdate.version,
              downloadedBytes: 0,
              contentLength: nextContentLength,
              progressEvent: "Started",
            });
            return;
          }

          if (event.event === "Progress") {
            const downloadedBytes =
              useUpdaterStore.getState().downloadedBytes + event.data.chunkLength;
            setDownloadProgress(downloadedBytes, nextContentLength);
            void appendUpdaterDiagnostic({
              context: "updater_download_progress",
              area: "updater",
              stage: "downloading",
              currentVersion: availableUpdate.currentVersion,
              targetVersion: availableUpdate.version,
              downloadedBytes,
              contentLength: nextContentLength,
              progressEvent: "Progress",
              chunkLength: event.data.chunkLength,
            });
            return;
          }

          if (event.event === "Finished") {
            setStage("installing");
            void appendUpdaterDiagnostic({
              context: "updater_installing",
              area: "updater",
              stage: "installing",
              currentVersion: availableUpdate.currentVersion,
              targetVersion: availableUpdate.version,
              downloadedBytes: useUpdaterStore.getState().downloadedBytes,
              contentLength: nextContentLength,
              progressEvent: "Finished",
            });
          }
        });
        toast.success("Actualización instalada. Reiniciando Taskify...");
        setDialogOpen(false);
        setAvailableUpdate(null);
        resetProgress();
        await relaunch();
      } catch (error) {
        void appendUpdaterDiagnostic({
          context: "updater_download_install_failed",
          area: "updater",
          stage: useUpdaterStore.getState().stage,
          currentVersion: availableUpdate.currentVersion,
          targetVersion: availableUpdate.version,
          downloadedBytes: useUpdaterStore.getState().downloadedBytes,
          contentLength: useUpdaterStore.getState().contentLength,
          ...diagnosticErrorDetails(error),
        });
        setStage("available");
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
