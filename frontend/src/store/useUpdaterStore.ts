import { create } from "zustand";
import type { Update } from "@tauri-apps/plugin-updater";

type UpdateStage =
  | "idle"
  | "checking"
  | "available"
  | "downloading"
  | "installing";

type AvailableUpdate = {
  currentVersion: string;
  version: string;
  body?: string;
  date?: string;
  rawJson: Record<string, unknown>;
  handle: Update;
};

interface UpdaterState {
  hasCheckedOnStartup: boolean;
  isDialogOpen: boolean;
  stage: UpdateStage;
  availableUpdate: AvailableUpdate | null;
  downloadedBytes: number;
  contentLength: number | null;
  markStartupChecked: () => void;
  setStage: (stage: UpdateStage) => void;
  setDialogOpen: (open: boolean) => void;
  setAvailableUpdate: (update: AvailableUpdate | null) => void;
  setDownloadProgress: (downloadedBytes: number, contentLength: number | null) => void;
  resetProgress: () => void;
}

export const useUpdaterStore = create<UpdaterState>((set) => ({
  hasCheckedOnStartup: false,
  isDialogOpen: false,
  stage: "idle",
  availableUpdate: null,
  downloadedBytes: 0,
  contentLength: null,
  markStartupChecked: () => set({ hasCheckedOnStartup: true }),
  setStage: (stage) => set({ stage }),
  setDialogOpen: (isDialogOpen) => set({ isDialogOpen }),
  setAvailableUpdate: (availableUpdate) =>
    set({
      availableUpdate,
      isDialogOpen: availableUpdate ? true : false,
      stage: availableUpdate ? "available" : "idle",
    }),
  setDownloadProgress: (downloadedBytes, contentLength) =>
    set({ downloadedBytes, contentLength }),
  resetProgress: () => set({ downloadedBytes: 0, contentLength: null }),
}));
