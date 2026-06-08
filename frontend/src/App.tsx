import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { LoginForm } from "@/components/auth/LoginForm";
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
          <LoginForm onLoginSuccess={setAccessToken} />
        )}
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
