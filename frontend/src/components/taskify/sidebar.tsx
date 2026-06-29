"use client";

import React, { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { isTauriRuntime } from "@/lib/runtime";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { invalidateTaskCaches } from "@/components/taskify/task-cache";
import type { CurrentView } from "@/components/taskify/navigation";
import type { Board } from "@/services/boardService";
import { deleteBoard } from "@/services/boardService";
import { useAuthStore } from "@/store/useAuthStore";
import { ProfileAvatar } from "@/components/profile-avatar";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NewBoardDialog } from "@/components/taskify/new-board-dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useToast } from "@/components/ui/toast-provider";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { getFriendlyErrorMessage } from "@/services/api";
import { checkpointSQLite, forceSync } from "@/services/systemService";
import { updateCurrentUserProfile } from "@/services/userService";
import { useUIStore } from "@/store/useUIStore";
import {
  LayoutDashboard,
  Plus,
  CheckSquare,
  Calendar,
  PieChart,
  Zap,
  Settings,
  HelpCircle,
  ChevronRight,
  LogOut,
  Trash2,
  Loader2,
} from "lucide-react";

const boardColors = [
  "bg-indigo-500",
  "bg-violet-500",
  "bg-amber-500",
  "bg-emerald-500",
];

const navItems: {
  icon: React.ElementType;
  label: string;
  view: CurrentView;
}[] = [
  { icon: LayoutDashboard, label: "Dashboard", view: "dashboard" },
  { icon: CheckSquare, label: "Mis Tareas", view: "tasks" },
  { icon: Calendar, label: "Agenda", view: "agenda" },
  { icon: PieChart, label: "Control financiero", view: "financial" },
];

interface SidebarProps {
  className?: string;
  activeView?: CurrentView;
  boards?: Board[];
  boardsError?: string;
  boardsLoading?: boolean;
  onViewChange?: (view: CurrentView) => void;
  selectedBoardId?: string | null;
  onBoardSelect?: (board: Board) => void;
}

export function Sidebar({
  className,
  activeView = "tasks",
  boards = [],
  boardsError,
  boardsLoading = false,
  onViewChange,
  selectedBoardId,
  onBoardSelect,
}: SidebarProps) {
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const updateUserProfile = useAuthStore((state) => state.updateUserProfile);
  const logout = useAuthStore((state) => state.logout);
  const toast = useToast();
  const isDesktopRuntime = isTauriRuntime();
  const [newBoardOpen, setNewBoardOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [displayNameDraft, setDisplayNameDraft] = useState(
    user?.fullName ?? "",
  );
  const [boardToDelete, setBoardToDelete] = useState<Board | null>(null);
  const openHelpModal = useUIStore((state) => state.setHelpModalOpen);
  const deleteBoardMutation = useMutation({
    mutationFn: deleteBoard,
    onSuccess: (_data, boardId) => {
      queryClient.invalidateQueries({ queryKey: ["boards"] });
      invalidateTaskCaches(queryClient, boardId);
      setBoardToDelete(null);
    },
  });
  const updateProfileMutation = useMutation({
    mutationFn: updateCurrentUserProfile,
    onSuccess: async (profile) => {
      updateUserProfile({
        email: profile.email,
        firstName: profile.firstName,
        lastName: profile.lastName,
        fullName: [profile.firstName, profile.lastName]
          .filter(Boolean)
          .join(" "),
        avatarLocalPath: profile.avatarLocalPath,
        avatarUrl: profile.avatarUrl,
      });
      queryClient.setQueryData(["users", "me"], profile);
      await queryClient.invalidateQueries({ queryKey: ["users", "me"] });
      toast.success("Nombre actualizado");
    },
    onError: (error) => {
      toast.error(getFriendlyErrorMessage(error));
    },
  });
  const forceSyncMutation = useMutation({
    mutationFn: forceSync,
    onSuccess: async () => {
      toast.success("Sincronización completada");
      await queryClient.invalidateQueries();
    },
    onError: (error) => {
      toast.error(getFriendlyErrorMessage(error));
    },
  });

  useEffect(() => {
    if (settingsOpen) {
      setDisplayNameDraft(user?.fullName ?? "");
    }
  }, [settingsOpen, user?.fullName]);

  function handleDeleteBoard(board: Board) {
    setBoardToDelete(board);
  }

  function handleConfirmDeleteBoard() {
    if (!boardToDelete) {
      return;
    }

    deleteBoardMutation.mutate(boardToDelete.id);
  }

  function handleSaveProfile() {
    updateProfileMutation.mutate(displayNameDraft);
  }

  async function handleExportBackup() {
    try {
      if (!isDesktopRuntime) {
        toast.error(
          "La copia de seguridad local solo está disponible en la app de escritorio.",
        );
        return;
      }

      const [{ save }, { copyFile }, { configDir, join }] = await Promise.all([
        import("@tauri-apps/plugin-dialog"),
        import("@tauri-apps/plugin-fs"),
        import("@tauri-apps/api/path"),
      ]);
      await checkpointSQLite();
      const destination = await save({
        defaultPath: "taskify_backup.db",
        filters: [{ name: "SQLite DB", extensions: ["db"] }],
      });

      if (!destination) {
        return;
      }

      const source = await join(await configDir(), "Taskify", "taskify.db");
      await copyFile(source, destination);
      toast.success("Respaldo guardado correctamente");
    } catch (error) {
      console.error("backup export failed", error);
      toast.error(
        getFriendlyErrorMessage(error, "No pudimos guardar el respaldo."),
      );
    }
  }

  return (
    <TooltipProvider delayDuration={0}>
      <NewBoardDialog open={newBoardOpen} onOpenChange={setNewBoardOpen} />
      <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
        <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Configuración</DialogTitle>
            <DialogDescription>
              Ajusta tu cuenta y prepara acciones de respaldo para tus datos.
            </DialogDescription>
          </DialogHeader>

          <Tabs defaultValue="account" className="mt-2">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="account">Cuenta</TabsTrigger>
              <TabsTrigger value="data">Datos</TabsTrigger>
            </TabsList>

            <TabsContent value="account" className="mt-4 space-y-4">
              <div className="space-y-2">
                <Label htmlFor="settings-display-name">Nombre</Label>
                <Input
                  id="settings-display-name"
                  value={displayNameDraft}
                  onChange={(event) => setDisplayNameDraft(event.target.value)}
                  placeholder="Tu nombre"
                />
              </div>
              <p className="rounded-md border border-border bg-muted/50 px-3 py-2 text-xs leading-relaxed text-muted-foreground">
                Este nombre se mostrará en tu perfil y se sincronizará con la
                nube.
              </p>
              <Button
                type="button"
                className="w-full sm:w-auto"
                disabled={
                  updateProfileMutation.isPending ||
                  displayNameDraft.trim() === ""
                }
                onClick={handleSaveProfile}
              >
                {updateProfileMutation.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : null}
                Guardar cambios
              </Button>
            </TabsContent>

            <TabsContent value="data" className="mt-4 space-y-3">
              <Button
                type="button"
                variant="outline"
                className="w-full justify-start"
                disabled={forceSyncMutation.isPending}
                onClick={() => forceSyncMutation.mutate()}
              >
                {forceSyncMutation.isPending ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : null}
                Sincronizar a la nube
              </Button>
              <Button
                type="button"
                variant="outline"
                className="w-full justify-start border-destructive/40 text-destructive hover:bg-destructive/10 hover:text-destructive"
                disabled={!isDesktopRuntime}
                onClick={handleExportBackup}
              >
                Exportar copia de seguridad
              </Button>
              <p className="text-xs leading-relaxed text-muted-foreground">
                Puedes forzar la sincronización o guardar una copia local de la
                base de datos.
              </p>
            </TabsContent>
          </Tabs>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(boardToDelete)}
        onOpenChange={(open) => {
          if (!open) {
            setBoardToDelete(null);
          }
        }}
        title="Eliminar tablero"
        description={
          boardToDelete
            ? `Se eliminará "${boardToDelete.name}" junto con sus columnas y tareas. Esta acción no se puede deshacer.`
            : ""
        }
        confirmLabel="Eliminar tablero"
        isPending={deleteBoardMutation.isPending}
        onConfirm={handleConfirmDeleteBoard}
      />
      <aside
        className={cn(
          "flex h-full w-64 flex-col bg-sidebar text-sidebar-foreground",
          className,
        )}
      >
        {/* Logo */}
        <div className="flex items-center gap-2.5 px-5 py-5">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary">
            <Zap className="size-4 text-primary-foreground" strokeWidth={2.5} />
          </div>
          <span className="text-lg font-bold tracking-tight text-sidebar-foreground">
            Taskify
          </span>
        </div>

        {/* Nav Items */}
        <nav className="flex flex-col gap-1 px-3 pt-4">
          {navItems.map(({ icon: Icon, label, view }) => {
            const isActive = view === activeView;
            return (
              <button
                key={label}
                onClick={() => onViewChange?.(view)}
                className={cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
                )}
              >
                <Icon className="size-4 shrink-0" />
                {label}
              </button>
            );
          })}
        </nav>

        {/* Boards Section */}
        <div className="mt-6 flex-1 overflow-y-auto px-3">
          <div className="mb-2 flex items-center justify-between px-3">
            <span className="text-xs font-semibold uppercase tracking-widest text-sidebar-foreground/40">
              Mis Tableros
            </span>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-6 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                  onClick={() => setNewBoardOpen(true)}
                >
                  <Plus className="size-3.5" />
                  <span className="sr-only">Crear tablero</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">Crear tablero</TooltipContent>
            </Tooltip>
          </div>

          <div className="flex flex-col gap-0.5">
            {boardsLoading ? (
              <p className="px-3 py-2 text-xs font-medium text-sidebar-foreground/50">
                Cargando tableros...
              </p>
            ) : null}

            {!boardsLoading && boardsError ? (
              <p className="px-3 py-2 text-xs font-medium text-red-300">
                {boardsError}
              </p>
            ) : null}

            {!boardsLoading && !boardsError && boards.length === 0 ? (
              <p className="px-3 py-2 text-xs font-medium text-sidebar-foreground/50">
                Aún no tienes tableros.
              </p>
            ) : null}

            {!boardsLoading && !boardsError
              ? boards.map((board, index) => (
                  <div
                    key={board.id}
                    className={cn(
                      "group flex items-center gap-1 rounded-md px-1 py-1 text-sm transition-colors",
                      selectedBoardId === board.id
                        ? "bg-sidebar-accent text-sidebar-foreground font-medium"
                        : "text-sidebar-foreground/65 hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
                    )}
                  >
                    <button
                      type="button"
                      onClick={() => {
                        onViewChange?.("tasks");
                        onBoardSelect?.(board);
                      }}
                      className="flex min-w-0 flex-1 items-center gap-3 rounded-md px-2 py-1.5 text-left"
                    >
                      <span
                        className={cn(
                          "size-2.5 shrink-0 rounded-full",
                          boardColors[index % boardColors.length],
                        )}
                      />
                      <span className="flex-1 truncate">{board.name}</span>
                      <ChevronRight
                        className={cn(
                          "size-3.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-70",
                          selectedBoardId === board.id && "opacity-70",
                        )}
                      />
                    </button>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          size="icon"
                          variant="ghost"
                          className={cn(
                            "size-7 shrink-0 text-sidebar-foreground/45 opacity-0 hover:bg-red-500/10 hover:text-red-300 group-hover:opacity-100",
                            selectedBoardId === board.id && "opacity-70",
                          )}
                          disabled={deleteBoardMutation.isPending}
                          onClick={() => handleDeleteBoard(board)}
                        >
                          <Trash2 className="size-3.5" />
                          <span className="sr-only">Eliminar tablero</span>
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent side="right">
                        Eliminar tablero
                      </TooltipContent>
                    </Tooltip>
                  </div>
                ))
              : null}
          </div>
        </div>

        {/* Bottom Actions */}
        <div className="border-t border-sidebar-foreground/10 px-3 py-3">
          <div className="flex gap-1 justify-center mb-3">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-8 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                  onClick={() => setSettingsOpen(true)}
                >
                  <Settings className="size-4" />
                  <span className="sr-only">Configuración</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">Configuración</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-8 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                  onClick={() => openHelpModal(true)}
                >
                  <HelpCircle className="size-4" />
                  <span className="sr-only">Ayuda</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">Ayuda</TooltipContent>
            </Tooltip>
          </div>

          {/* User Profile */}
          <div className="flex items-center gap-3 rounded-md px-2 py-2 hover:bg-sidebar-accent/60 transition-colors">
            <ProfileAvatar className="size-8" />
            <div className="flex-1 overflow-hidden">
              <p className="truncate text-sm font-medium text-sidebar-foreground">
                {user?.fullName ?? "Taskify User"}
              </p>
              <p className="truncate text-xs text-sidebar-foreground/50">
                {user?.email ?? "Sin correo"}
              </p>
            </div>
            <Button
              size="icon"
              variant="ghost"
              className="size-8 shrink-0 text-sidebar-foreground/50 hover:bg-sidebar-accent hover:text-sidebar-foreground"
              onClick={() => logout()}
              aria-label="Cerrar sesión"
            >
              <LogOut className="size-4" />
            </Button>
          </div>
        </div>
      </aside>
    </TooltipProvider>
  );
}
