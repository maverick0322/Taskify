import { useCallback, useEffect, useMemo } from "react";
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

        await availableUpdate.handle.downloadAndInstall((event) => {
          if (event.event === "Started") {
            nextContentLength = event.data.contentLength ?? null;
            setDownloadProgress(0, nextContentLength);
            return;
          }

          if (event.event === "Progress") {
            setDownloadProgress(
              useUpdaterStore.getState().downloadedBytes + event.data.chunkLength,
              nextContentLength,
            );
            return;
          }

          if (event.event === "Finished") {
            setStage("installing");
          }
        });

        toast.success("Actualización instalada. Reiniciando Taskify...");
        setDialogOpen(false);
        setAvailableUpdate(null);
        resetProgress();
        await relaunch();
      } catch (error) {
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
