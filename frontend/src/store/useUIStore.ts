import { create } from "zustand"

interface UIState {
  isNewTaskModalOpen: boolean
  isNewMovementModalOpen: boolean
  isHelpModalOpen: boolean
  isNotificationsOpen: boolean
  setNewTaskModalOpen: (open: boolean) => void
  toggleNewTaskModal: () => void
  setNewMovementModalOpen: (open: boolean) => void
  toggleNewMovementModal: () => void
  setHelpModalOpen: (open: boolean) => void
  toggleHelpModal: () => void
  setNotificationsOpen: (open: boolean) => void
  toggleNotifications: () => void
  closeGlobalOverlays: () => void
}

export const useUIStore = create<UIState>((set) => ({
  isNewTaskModalOpen: false,
  isNewMovementModalOpen: false,
  isHelpModalOpen: false,
  isNotificationsOpen: false,
  setNewTaskModalOpen: (open) => set({ isNewTaskModalOpen: open }),
  toggleNewTaskModal: () =>
    set((state) => ({ isNewTaskModalOpen: !state.isNewTaskModalOpen })),
  setNewMovementModalOpen: (open) => set({ isNewMovementModalOpen: open }),
  toggleNewMovementModal: () =>
    set((state) => ({
      isNewMovementModalOpen: !state.isNewMovementModalOpen,
    })),
  setHelpModalOpen: (open) => set({ isHelpModalOpen: open }),
  toggleHelpModal: () =>
    set((state) => ({ isHelpModalOpen: !state.isHelpModalOpen })),
  setNotificationsOpen: (open) => set({ isNotificationsOpen: open }),
  toggleNotifications: () =>
    set((state) => ({ isNotificationsOpen: !state.isNotificationsOpen })),
  closeGlobalOverlays: () =>
    set({
      isNewTaskModalOpen: false,
      isNewMovementModalOpen: false,
      isHelpModalOpen: false,
      isNotificationsOpen: false,
    }),
}))
