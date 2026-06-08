import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect } from "react";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { AuthScreen } from "@/components/auth/AuthScreen";
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
      <TooltipProvider>
        {accessToken ? <TaskifyDashboard /> : <AuthScreen />}
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
