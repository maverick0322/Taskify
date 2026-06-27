import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { AuthScreen } from "@/components/auth/AuthScreen";
import { ThemeProvider } from "@/components/theme-provider";
import { WindowTitlebar } from "@/components/taskify/window-titlebar";
import { ToastProvider } from "@/components/ui/toast-provider";
import { TooltipProvider } from "@/components/ui/tooltip";
import { loadStoredSession } from "@/services/secureSession";
import { restoreDesktopSyncSession } from "@/services/systemService";
import { useAuthStore } from "@/store/useAuthStore";

const queryClient = new QueryClient();

function App() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const login = useAuthStore((state) => state.login);
  const [isBootstrappingSession, setIsBootstrappingSession] = useState(true);

  useEffect(() => {
    let isMounted = true;

    void (async () => {
      const storedSession = await loadStoredSession().catch(() => null);
      if (!isMounted) {
        return;
      }

      if (
        storedSession?.remoteAccessToken &&
        storedSession.remoteRefreshToken
      ) {
        await restoreDesktopSyncSession({
          accessToken: storedSession.remoteAccessToken,
          refreshToken: storedSession.remoteRefreshToken,
        }).catch(() => undefined);
      }
      if (storedSession?.accessToken) {
        login(storedSession.accessToken);
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
            <div className="flex h-dvh min-h-dvh flex-col overflow-hidden bg-background">
              <WindowTitlebar />
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
