import { create } from "zustand";

export type DesktopSyncStatus = "connected" | "pending" | "offline" | "error";

interface DesktopSyncState {
  status: DesktopSyncStatus;
  message: string | null;
  setStatus: (status: DesktopSyncStatus, message?: string | null) => void;
  reset: () => void;
}

export const useDesktopSyncStore = create<DesktopSyncState>((set) => ({
  status: "connected",
  message: null,
  setStatus: (status, message = null) => {
    set({ status, message });
  },
  reset: () => {
    set({ status: "connected", message: null });
  },
}));
