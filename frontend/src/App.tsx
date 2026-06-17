import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect } from "react";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { AuthScreen } from "@/components/auth/AuthScreen";
import { ThemeProvider } from "@/components/theme-provider";
import { WindowTitlebar } from "@/components/taskify/window-titlebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useAuthStore } from "@/store/useAuthStore";

const queryClient = new QueryClient();

function App() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const login = useAuthStore((state) => state.login);

  useEffect(() => {
    const storedToken = localStorage.getItem("accessToken");
    if (storedToken) {
      login(storedToken);
    }
  }, [login]);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="system" storageKey="taskify-theme">
        <TooltipProvider>
          <div className="flex h-dvh min-h-dvh flex-col overflow-hidden bg-background">
            <WindowTitlebar />
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              {accessToken ? <TaskifyDashboard /> : <AuthScreen />}
            </div>
          </div>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;
