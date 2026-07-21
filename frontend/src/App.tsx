import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { AuthScreen } from "@/components/auth/AuthScreen";
import { UpdateDialog } from "@/components/taskify/update-dialog";
import { ThemeProvider } from "@/components/theme-provider";
import { WindowTitlebar } from "@/components/taskify/window-titlebar";
import { ToastProvider } from "@/components/ui/toast-provider";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useDesktopUpdater } from "@/hooks/useDesktopUpdater";
import { isTauriRuntime } from "@/lib/runtime";
import { restoreOrRefreshSession } from "@/services/api";
import { restoreDesktopSyncSession } from "@/services/systemService";
import { useAuthStore } from "@/store/useAuthStore";
import { useDesktopSyncStore } from "@/store/useDesktopSyncStore";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 0,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
    },
  },
});

function App() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const login = useAuthStore((state) => state.login);
  const setDesktopSyncStatus = useDesktopSyncStore((state) => state.setStatus);
  const resetDesktopSyncStatus = useDesktopSyncStore((state) => state.reset);
  const [isBootstrappingSession, setIsBootstrappingSession] = useState(true);
  const lastRestoreTokenRef = useRef<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    void (async () => {
      const restoredAccessToken = await restoreOrRefreshSession().catch(
        () => null,
      );
      if (!isMounted) {
        return;
      }

      if (restoredAccessToken) {
        login(restoredAccessToken);
      }
      setIsBootstrappingSession(false);
    })();

    return () => {
      isMounted = false;
    };
  }, [login]);

  useEffect(() => {
    if (!isTauriRuntime()) {
      return;
    }

    if (!accessToken) {
      lastRestoreTokenRef.current = null;
      resetDesktopSyncStatus();
      return;
    }

    if (lastRestoreTokenRef.current === accessToken) {
      return;
    }

    lastRestoreTokenRef.current = accessToken;
    setDesktopSyncStatus(
      "pending",
      "Restaurando la sincronización en la nube…",
    );

    void restoreDesktopSyncSession()
      .then((result) => {
        if (result?.initialSyncCompleted) {
          setDesktopSyncStatus("connected", null);
          return;
        }

        if (result?.syncState === "error") {
          setDesktopSyncStatus(
            "error",
            "No pudimos restaurar la sincronización remota. Tus datos locales siguen disponibles.",
          );
          return;
        }

        setDesktopSyncStatus(
          result?.restored === false ? "offline" : "pending",
          result?.restored === false
            ? "La sesión remota no está disponible. Seguimos trabajando con tus datos locales."
            : "La sincronización remota sigue pendiente. Tus datos locales siguen disponibles.",
        );
      })
      .catch(() => {
        setDesktopSyncStatus(
          "error",
          "La sincronización remota no respondió. Seguimos trabajando con tus datos locales.",
        );
      });
  }, [accessToken, resetDesktopSyncStatus, setDesktopSyncStatus]);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="system" storageKey="taskify-theme">
        <ToastProvider>
          <TooltipProvider>
            <DesktopUpdaterBridge isReady={!isBootstrappingSession} />
            <div className="flex h-dvh min-h-dvh flex-col overflow-hidden bg-background">
              <WindowTitlebar />
              <UpdateDialog />
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                {isBootstrappingSession ? null : accessToken ? (
                  <TaskifyDashboard />
                ) : (
                  <AuthScreen />
                )}
              </div>
            </div>
          </TooltipProvider>
        </ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;

function DesktopUpdaterBridge({ isReady }: { isReady: boolean }) {
  useDesktopUpdater({
    enableStartupCheck: true,
    isReady,
  });

  return null;
}
