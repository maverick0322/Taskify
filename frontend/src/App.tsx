import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { TooltipProvider } from "@/components/ui/tooltip";

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <TaskifyDashboard />
      </TooltipProvider>
    </QueryClientProvider>
  );
}

export default App;
