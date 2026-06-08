import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { AuthScreen } from "@/components/auth/AuthScreen";
import { TooltipProvider } from "@/components/ui/tooltip";

const queryClient = new QueryClient();

function App() {
  const [accessToken, setAccessToken] = useState(() =>
    localStorage.getItem("accessToken"),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        {accessToken ? (
          <TaskifyDashboard />
        ) : (
          <AuthScreen onLoginSuccess={setAccessToken} />
        )}
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
