import { TaskifyDashboard } from "@/components/TaskifyDashboard";
import { TooltipProvider } from "@/components/ui/tooltip";

function App() {
  return (
    <TooltipProvider>
      <TaskifyDashboard />
    </TooltipProvider>
  );
}

export default App;
