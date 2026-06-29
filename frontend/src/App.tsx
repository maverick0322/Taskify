import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { AuthScreen } from "@/components/auth/AuthScreen";
import { UpdateDialog } from "@/components/taskify/update-dialog";
import { ThemeProvider } from "@/components/theme-provider";
import { WindowTitlebar } from "@/components/taskify/window-titlebar";
import { ToastProvider } from "@/components/ui/toast-provider";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useDesktopUpdater } from "@/hooks/useDesktopUpdater";
import { restoreOrRefreshSession } from "@/services/api";
import { restoreDesktopSyncSession } from "@/services/systemService";
import { useAuthStore } from "@/store/useAuthStore";

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
  const [isBootstrappingSession, setIsBootstrappingSession] = useState(true);

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
        await restoreDesktopSyncSession().catch(() => undefined);
        login(restoredAccessToken);
      }
      setIsBootstrappingSession(false);
    })();

    return () => {
      isMounted = false;
    };
  }, [login]);

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
